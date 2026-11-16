package tools

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"squad/features/bots"
	"strings"
	"time"
)

var exaBaseURL = "https://api.exa.ai"

type exaResult struct {
	Title         string   `json:"title"`
	URL           string   `json:"url"`
	Author        string   `json:"author"`
	PublishedDate string   `json:"publishedDate"`
	Highlights    []string `json:"highlights"`
	Text          string   `json:"text"`
}

func isExaConfigured() bool {
	return os.Getenv("EXA_API_KEY") != ""
}

func registerExaTools() {
	if !isExaConfigured() {
		return
	}

	register("web", Tool{
		Name:        "web_search",
		Description: "Search the web with Exa, a semantic search engine. Phrase the query as a description of the page you want, not as keywords. Returns titles, URLs and highlights; use fetch_url to read a result in full.",
		Parameters: objectSchema(map[string]any{
			"query":      stringProperty("A description of the page to find."),
			"timeFilter": enumProperty("Only return pages published within this period.", "day", "week", "month", "year", "all"),
			"limit":      integerProperty("Number of results, 1 to 10. Defaults to 5."),
		}, "query"),
		Run: runWebSearch,
	})
}

func runWebSearch(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		Query      string `json:"query"`
		TimeFilter string `json:"timeFilter"`
		Limit      int    `json:"limit"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	body := map[string]any{
		"query": args.Query,
		"type":  "auto",
		// Exa charges extra for every result past 10.
		"numResults": clampLimit(args.Limit, 5, 10),
		"contents":   map[string]any{"highlights": true},
	}
	if seconds, ok := secondsByTimeFilter[args.TimeFilter]; ok {
		cutoff := time.Now().Add(-time.Duration(seconds) * time.Second).UTC()
		body["startPublishedDate"] = cutoff.Format("2006-01-02T15:04:05Z")
	}

	var response struct {
		Results []exaResult `json:"results"`
	}
	if err := callExa(ctx, "/search", body, &response); err != nil {
		return "", err
	}

	if len(response.Results) == 0 {
		return "No results.", nil
	}

	lines := []string{}
	for i, result := range response.Results {
		lines = append(lines, fmt.Sprintf("%d. %s\n%s", i+1, result.Title, result.URL))
		if result.PublishedDate != "" {
			lines = append(lines, "Published: "+result.PublishedDate)
		}
		for _, highlight := range result.Highlights {
			lines = append(lines, "> "+strings.TrimSpace(highlight))
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n"), nil
}

func getExaContents(ctx context.Context, pageURL string) (string, error) {
	var response struct {
		Results  []exaResult `json:"results"`
		Statuses []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			Error  any    `json:"error"`
		} `json:"statuses"`
	}

	body := map[string]any{"urls": []string{pageURL}, "text": map[string]any{"maxCharacters": maxOutputLength}}
	if err := callExa(ctx, "/contents", body, &response); err != nil {
		return "", err
	}

	// Exa answers 200 even when a page fails, the failure only shows up in statuses.
	for _, status := range response.Statuses {
		if status.Status != "success" {
			return "", fmt.Errorf("the site blocked the request and Exa could not fetch it either: %v", status.Error)
		}
	}

	if len(response.Results) == 0 {
		return "", fmt.Errorf("the site blocked the request and Exa had no copy of it")
	}

	result := response.Results[0]
	return "# " + result.Title + "\n\n" + result.Text, nil
}

func callExa(ctx context.Context, path string, body any, target any) error {
	headers := map[string]string{"x-api-key": os.Getenv("EXA_API_KEY")}
	return sendJSONRequest(ctx, http.MethodPost, exaBaseURL+path, headers, body, target)
}
