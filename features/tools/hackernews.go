package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"squad/features/bots"
	"strconv"
	"strings"
	"time"
)

const hackerNewsBaseURL = "https://hn.algolia.com/api/v1"

var hackerNewsListingTags = map[string]string{
	"top":  "front_page",
	"new":  "story",
	"ask":  "ask_hn",
	"show": "show_hn",
	"job":  "job",
}

type hackerNewsHit struct {
	ObjectID    string `json:"objectID"`
	StoryID     int    `json:"story_id"`
	Title       string `json:"title"`
	StoryTitle  string `json:"story_title"`
	URL         string `json:"url"`
	StoryURL    string `json:"story_url"`
	StoryText   string `json:"story_text"`
	CommentText string `json:"comment_text"`
	Author      string `json:"author"`
	Points      int    `json:"points"`
	NumComments int    `json:"num_comments"`
	CreatedAtI  int64  `json:"created_at_i"`
}

type hackerNewsItem struct {
	Type     string           `json:"type"`
	Author   string           `json:"author"`
	Text     string           `json:"text"`
	Children []hackerNewsItem `json:"children"`
}

func registerHackerNewsTools() {
	register("hackernews", Tool{
		Name:        "hackernews_search",
		Description: "Search Hacker News stories or comments by keyword through the Algolia API.",
		Parameters: objectSchema(map[string]any{
			"query":      stringProperty("Search terms."),
			"tags":       stringProperty("Algolia tag filter: story (default), comment, ask_hn, show_hn."),
			"sort":       enumProperty("relevance (default) or date for newest first.", "relevance", "date"),
			"timeFilter": enumProperty("Only include items from this period.", "hour", "day", "week", "month", "year", "all"),
			"limit":      integerProperty("Number of results, up to 50. Defaults to 20."),
		}, "query"),
		Run: runHackerNewsSearch,
	})

	register("hackernews", Tool{
		Name:        "hackernews_listing",
		Description: "List current Hacker News stories: top (front page), new, ask, show or job.",
		Parameters: objectSchema(map[string]any{
			"category": enumProperty("Which listing to fetch.", "top", "new", "ask", "show", "job"),
			"limit":    integerProperty("Number of stories, up to 50. Defaults to 30."),
		}, "category"),
		Run: runHackerNewsListing,
	})

	register("hackernews", Tool{
		Name:        "hackernews_comments",
		Description: "Read the comment thread of a Hacker News story, flattened with indentation for replies.",
		Parameters: objectSchema(map[string]any{
			"itemId":   stringProperty("The story id from a search or listing result."),
			"maxDepth": integerProperty("How deep to follow replies. Defaults to 3."),
			"limit":    integerProperty("Maximum comments to return. Defaults to 100."),
		}, "itemId"),
		Run: runHackerNewsComments,
	})
}

func runHackerNewsSearch(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		Query      string `json:"query"`
		Tags       string `json:"tags"`
		Sort       string `json:"sort"`
		TimeFilter string `json:"timeFilter"`
		Limit      int    `json:"limit"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	if args.Tags == "" {
		args.Tags = "story"
	}

	params := url.Values{}
	params.Set("query", args.Query)
	params.Set("tags", args.Tags)
	params.Set("hitsPerPage", strconv.Itoa(clampLimit(args.Limit, 20, 50)))

	seconds := map[string]int64{"hour": 3600, "day": 86400, "week": 604800, "month": 2629800, "year": 31556952}
	if value, ok := seconds[args.TimeFilter]; ok {
		params.Set("numericFilters", fmt.Sprintf("created_at_i>%d", time.Now().Unix()-value))
	}

	endpoint := "/search"
	if args.Sort == "date" {
		endpoint = "/search_by_date"
	}

	return getHackerNewsHits(ctx, endpoint, params)
}

func runHackerNewsListing(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		Category string `json:"category"`
		Limit    int    `json:"limit"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	tag, ok := hackerNewsListingTags[args.Category]
	if !ok {
		tag = "front_page"
	}

	params := url.Values{}
	params.Set("tags", tag)
	params.Set("hitsPerPage", strconv.Itoa(clampLimit(args.Limit, 30, 50)))

	endpoint := "/search_by_date"
	if tag == "front_page" {
		endpoint = "/search"
	}

	return getHackerNewsHits(ctx, endpoint, params)
}

func getHackerNewsHits(ctx context.Context, endpoint string, params url.Values) (string, error) {
	var response struct {
		Hits []hackerNewsHit `json:"hits"`
	}
	if err := sendJSONRequest(ctx, http.MethodGet, hackerNewsBaseURL+endpoint+"?"+params.Encode(), nil, nil, &response); err != nil {
		return "", err
	}

	if len(response.Hits) == 0 {
		return "No results.", nil
	}

	lines := []string{}
	for i, hit := range response.Hits {
		title := firstNonEmpty(hit.Title, hit.StoryTitle)
		link := firstNonEmpty(hit.URL, hit.StoryURL)
		permalink := "https://news.ycombinator.com/item?id=" + hit.ObjectID

		lines = append(lines, fmt.Sprintf("%d. %s (id %s, %d points, %d comments, by %s, %s)", i+1, title, hit.ObjectID, hit.Points, hit.NumComments, hit.Author, time.Unix(hit.CreatedAtI, 0).Format("2006-01-02")))
		if link != "" {
			lines = append(lines, "   "+link)
		}
		lines = append(lines, "   "+permalink)

		body := strings.TrimSpace(htmlToText(firstNonEmpty(hit.StoryText, hit.CommentText)))
		if body != "" {
			lines = append(lines, "   "+strings.ReplaceAll(body, "\n", "\n   "))
		}
	}

	return strings.Join(lines, "\n"), nil
}

func runHackerNewsComments(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		ItemID   string `json:"itemId"`
		MaxDepth int    `json:"maxDepth"`
		Limit    int    `json:"limit"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	maxDepth := clampLimit(args.MaxDepth, 3, 10)
	limit := clampLimit(args.Limit, 100, 300)

	var story hackerNewsItem
	if err := sendJSONRequest(ctx, http.MethodGet, hackerNewsBaseURL+"/items/"+url.PathEscape(args.ItemID), nil, nil, &story); err != nil {
		return "", err
	}

	lines := []string{}
	var walk func(item hackerNewsItem, depth int)
	walk = func(item hackerNewsItem, depth int) {
		if len(lines) >= limit {
			return
		}

		text := strings.TrimSpace(htmlToText(item.Text))
		if item.Type == "comment" && text != "" {
			indent := strings.Repeat("  ", depth)
			lines = append(lines, fmt.Sprintf("%s- %s: %s", indent, item.Author, strings.ReplaceAll(text, "\n", "\n"+indent+"  ")))
		}

		if depth < maxDepth {
			for _, child := range item.Children {
				walk(child, depth+1)
			}
		}
	}

	for _, child := range story.Children {
		walk(child, 0)
	}

	if len(lines) == 0 {
		return "No comments.", nil
	}

	return strings.Join(lines, "\n"), nil
}

func clampLimit(value, defaultValue, maxValue int) int {
	if value <= 0 {
		return defaultValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
