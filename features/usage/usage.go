package usage

import (
	"cmp"
	"net/http"
	"slices"
	"squad/commons/utils"
	"time"
)

const topChatsLimit = 5

type BotCost struct {
	TotalCost float64      `json:"totalCost"`
	ChatCount int          `json:"chatCount"`
	Buckets   []CostBucket `json:"buckets"`
	Models    []ModelCost  `json:"models"`
	TopChats  []ChatCost   `json:"topChats"`
}

type CostBucket struct {
	Start         time.Time `json:"start"`
	ChatCost      float64   `json:"chatCost"`
	ScheduledCost float64   `json:"scheduledCost"`
}

type ModelCost struct {
	Model string  `json:"model"`
	Cost  float64 `json:"cost"`
}

type ChatCost struct {
	ThreadID int     `json:"threadId"`
	Title    string  `json:"title"`
	Cost     float64 `json:"cost"`
}

func HandleGetBotCost(w http.ResponseWriter, r *http.Request) {
	botID, ok := utils.ParsePathID(w, r, "botId", "INVALID_BOT_ID", "Invalid bot")
	if !ok {
		return
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	var since time.Time
	isMonthly := false
	switch r.URL.Query().Get("range") {
	case "7d":
		since = today.AddDate(0, 0, -6)
	case "30d":
		since = today.AddDate(0, 0, -29)
	case "all":
		isMonthly = true
	default:
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_RANGE", "Invalid range", nil)
		return
	}

	entries, err := GetBotCostEntries(botID, since)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "COST_READ_FAILED", "Failed to load cost", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, summarizeCosts(entries, since, today, isMonthly))
}

// Days and months follow TZ like schedules do, so a morning run lands on the day it ran rather than the UTC day.
func summarizeCosts(entries []CostEntry, since, today time.Time, isMonthly bool) BotCost {
	summary := BotCost{Buckets: []CostBucket{}, Models: []ModelCost{}, TopChats: []ChatCost{}}

	bucketIndexes := map[string]int{}
	if isMonthly {
		first := today
		if len(entries) > 0 {
			first = entries[0].CreatedAt.In(time.Local)
		}
		thisMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.Local)
		for month := time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, time.Local); !month.After(thisMonth); month = month.AddDate(0, 1, 0) {
			bucketIndexes[getBucketKey(month, isMonthly)] = len(summary.Buckets)
			summary.Buckets = append(summary.Buckets, CostBucket{Start: month})
		}
	} else {
		for day := since; !day.After(today); day = day.AddDate(0, 0, 1) {
			bucketIndexes[getBucketKey(day, isMonthly)] = len(summary.Buckets)
			summary.Buckets = append(summary.Buckets, CostBucket{Start: day})
		}
	}

	chatIndexes := map[int]int{}
	modelIndexes := map[string]int{}
	for _, entry := range entries {
		summary.TotalCost += entry.Cost

		index, ok := bucketIndexes[getBucketKey(entry.CreatedAt.In(time.Local), isMonthly)]
		if ok {
			if entry.IsScheduled {
				summary.Buckets[index].ScheduledCost += entry.Cost
			} else {
				summary.Buckets[index].ChatCost += entry.Cost
			}
		}

		chatIndex, ok := chatIndexes[entry.ThreadID]
		if !ok {
			chatIndex = len(summary.TopChats)
			chatIndexes[entry.ThreadID] = chatIndex
			summary.TopChats = append(summary.TopChats, ChatCost{ThreadID: entry.ThreadID, Title: entry.Title})
		}
		summary.TopChats[chatIndex].Cost += entry.Cost

		modelIndex, ok := modelIndexes[entry.Model]
		if !ok {
			modelIndex = len(summary.Models)
			modelIndexes[entry.Model] = modelIndex
			summary.Models = append(summary.Models, ModelCost{Model: entry.Model})
		}
		summary.Models[modelIndex].Cost += entry.Cost
	}

	summary.ChatCount = len(summary.TopChats)

	slices.SortFunc(summary.Models, func(a, b ModelCost) int { return cmp.Compare(b.Cost, a.Cost) })
	slices.SortFunc(summary.TopChats, func(a, b ChatCost) int { return cmp.Compare(b.Cost, a.Cost) })
	if len(summary.TopChats) > topChatsLimit {
		summary.TopChats = summary.TopChats[:topChatsLimit]
	}

	return summary
}

func getBucketKey(t time.Time, isMonthly bool) string {
	if isMonthly {
		return t.Format("2006-01")
	}
	return t.Format(time.DateOnly)
}
