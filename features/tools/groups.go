package tools

import (
	"net/http"
	"slices"
	"squad/commons/utils"
	"squad/features/bots"
)

type ToolGroup struct {
	GroupID     string `json:"groupId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsAvailable bool   `json:"isAvailable"`
	SetupHint   string `json:"setupHint"`
}

var groups = []ToolGroup{
	{GroupID: "memory", Name: "Memory", Description: "Save and forget facts across chats."},
	{GroupID: "files", Name: "Files", Description: "Read and write the bot's markdown files."},
	{GroupID: "web", Name: "Web", Description: "Read web pages, and search the web when EXA_API_KEY is set."},
	{GroupID: "hackernews", Name: "Hacker News", Description: "Search stories and read comments."},
	{GroupID: "reddit", Name: "Reddit", Description: "Search posts and read comments.", SetupHint: "Set the REDDIT_ environment variables to use Reddit."},
	{GroupID: "zen", Name: "Zen", Description: "Search, read and write notes in Zen with this bot's API token.", SetupHint: "Set ZEN_URL to use Zen."},
}

func HandleGetToolGroups(w http.ResponseWriter, r *http.Request) {
	response := []ToolGroup{}
	for _, group := range groups {
		group.IsAvailable = isGroupRegistered(group.GroupID)
		response = append(response, group)
	}

	utils.SendJSONResponse(w, http.StatusOK, response)
}

// IsGroupEnabled needs a Zen token on the bot as well, since each bot can be limited to its own tags in Zen.
func IsGroupEnabled(bot bots.Bot, groupID string) bool {
	if slices.Contains(bot.DisabledToolGroups, groupID) {
		return false
	}
	if groupID == "zen" && bot.ZenAPIToken == "" {
		return false
	}
	return true
}

func isGroupRegistered(groupID string) bool {
	for _, tool := range registry {
		if tool.Group == groupID {
			return true
		}
	}
	return false
}
