package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"squad/features/bots"
	"strconv"
	"strings"
	"sync"
	"time"
)

type redditPost struct {
	ID          string  `json:"id"`
	Subreddit   string  `json:"subreddit"`
	Title       string  `json:"title"`
	SelfText    string  `json:"selftext"`
	Score       int     `json:"score"`
	NumComments int     `json:"num_comments"`
	Permalink   string  `json:"permalink"`
	CreatedUTC  float64 `json:"created_utc"`
}

type redditListing struct {
	Data struct {
		Children []struct {
			Kind string          `json:"kind"`
			Data json.RawMessage `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type redditComment struct {
	Body    string          `json:"body"`
	Score   int             `json:"score"`
	Author  string          `json:"author"`
	Replies json.RawMessage `json:"replies"`
}

type flatComment struct {
	Depth  int
	Score  int
	Author string
	Body   string
}

var redditAuthURL = "https://www.reddit.com/api/v1/access_token"
var redditAPIURL = "https://oauth.reddit.com"

var redditTokenMutex sync.Mutex
var redditToken string
var redditTokenExpiresAt time.Time

func registerRedditTools() {
	if os.Getenv("REDDIT_CLIENT_ID") == "" || os.Getenv("REDDIT_CLIENT_SECRET") == "" || os.Getenv("REDDIT_USERNAME") == "" || os.Getenv("REDDIT_PASSWORD") == "" {
		return
	}

	subredditsProperty := stringProperty("One subreddit name, or several joined with +, for example golang+programming.")
	timeFilterProperty := enumProperty("Time window for top and controversial sorts.", "hour", "day", "week", "month", "year", "all")
	limitProperty := integerProperty("Number of posts, up to 100. Defaults to 25.")

	register("reddit", Tool{
		Name:        "reddit_search",
		Description: "Search posts within subreddits. Reddit search matches keywords, it is not semantic.",
		Parameters: objectSchema(map[string]any{
			"subreddits": subredditsProperty,
			"query":      stringProperty("Search keywords."),
			"sort":       enumProperty("Result order. Defaults to top.", "relevance", "top", "new", "comments"),
			"timeFilter": timeFilterProperty,
			"limit":      limitProperty,
		}, "subreddits", "query"),
		Run: runRedditSearch,
	})

	register("reddit", Tool{
		Name:        "reddit_listing",
		Description: "List posts from subreddits, for example the top posts of the week.",
		Parameters: objectSchema(map[string]any{
			"subreddits": subredditsProperty,
			"sort":       enumProperty("Listing to fetch. Defaults to top.", "hot", "new", "top", "rising", "controversial"),
			"timeFilter": timeFilterProperty,
			"limit":      limitProperty,
		}, "subreddits"),
		Run: runRedditListing,
	})

	register("reddit", Tool{
		Name:        "reddit_comments",
		Description: "Read the comments of a Reddit post, flattened and sorted by score. Low-scoring comments are dropped because that is where the substance tends to be.",
		Parameters: objectSchema(map[string]any{
			"permalink": stringProperty("The post permalink from a search or listing result, starting with /r/."),
			"minScore":  integerProperty("Drop comments scoring below this. Defaults to 15."),
			"maxDepth":  integerProperty("How deep to follow replies. Defaults to 3."),
		}, "permalink"),
		Run: runRedditComments,
	})
}

func runRedditSearch(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		Subreddits string `json:"subreddits"`
		Query      string `json:"query"`
		Sort       string `json:"sort"`
		TimeFilter string `json:"timeFilter"`
		Limit      int    `json:"limit"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("q", args.Query)
	params.Set("restrict_sr", "1")
	params.Set("sort", firstNonEmpty(args.Sort, "top"))
	params.Set("t", firstNonEmpty(args.TimeFilter, "year"))
	params.Set("limit", strconv.Itoa(clampLimit(args.Limit, 25, 100)))

	var listing redditListing
	if err := getReddit(ctx, "/r/"+url.PathEscape(args.Subreddits)+"/search", params, &listing); err != nil {
		return "", err
	}

	return formatRedditPosts(listing)
}

func runRedditListing(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		Subreddits string `json:"subreddits"`
		Sort       string `json:"sort"`
		TimeFilter string `json:"timeFilter"`
		Limit      int    `json:"limit"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	sortOrder := firstNonEmpty(args.Sort, "top")

	params := url.Values{}
	params.Set("limit", strconv.Itoa(clampLimit(args.Limit, 25, 100)))
	if sortOrder == "top" || sortOrder == "controversial" {
		params.Set("t", firstNonEmpty(args.TimeFilter, "week"))
	}

	var listing redditListing
	if err := getReddit(ctx, "/r/"+url.PathEscape(args.Subreddits)+"/"+url.PathEscape(sortOrder), params, &listing); err != nil {
		return "", err
	}

	return formatRedditPosts(listing)
}

func runRedditComments(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		Permalink string `json:"permalink"`
		MinScore  *int   `json:"minScore"`
		MaxDepth  int    `json:"maxDepth"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	minScore := 15
	if args.MinScore != nil {
		minScore = *args.MinScore
	}
	maxDepth := clampLimit(args.MaxDepth, 3, 10)

	permalink := strings.TrimPrefix(args.Permalink, "https://www.reddit.com")
	if !strings.HasPrefix(permalink, "/r/") {
		return "", fmt.Errorf("permalink must start with /r/")
	}

	params := url.Values{}
	params.Set("limit", "120")
	params.Set("depth", strconv.Itoa(maxDepth))
	params.Set("sort", "top")

	var listings []redditListing
	if err := getReddit(ctx, permalink, params, &listings); err != nil {
		return "", err
	}
	if len(listings) < 2 {
		return "No comments.", nil
	}

	comments := []flatComment{}
	var walk func(kind string, data json.RawMessage, depth int)
	walk = func(kind string, data json.RawMessage, depth int) {
		if kind != "t1" {
			return
		}

		var comment redditComment
		if err := json.Unmarshal(data, &comment); err != nil {
			return
		}

		body := strings.TrimSpace(comment.Body)
		if body != "" && body != "[deleted]" && body != "[removed]" && comment.Score >= minScore {
			comments = append(comments, flatComment{Depth: depth, Score: comment.Score, Author: comment.Author, Body: body})
		}

		// Reddit sends replies as an empty string when there are none, and as a listing otherwise.
		var replies redditListing
		if depth < maxDepth && json.Unmarshal(comment.Replies, &replies) == nil {
			for _, child := range replies.Data.Children {
				walk(child.Kind, child.Data, depth+1)
			}
		}
	}

	for _, child := range listings[1].Data.Children {
		walk(child.Kind, child.Data, 0)
	}

	if len(comments) == 0 {
		return "No comments above the score threshold.", nil
	}

	sort.SliceStable(comments, func(i, j int) bool {
		return comments[i].Score > comments[j].Score
	})

	lines := []string{}
	for _, comment := range comments {
		lines = append(lines, fmt.Sprintf("[%d points, depth %d, %s] %s", comment.Score, comment.Depth, comment.Author, comment.Body))
	}

	return strings.Join(lines, "\n\n"), nil
}

func formatRedditPosts(listing redditListing) (string, error) {
	lines := []string{}

	for i, child := range listing.Data.Children {
		var post redditPost
		if err := json.Unmarshal(child.Data, &post); err != nil {
			return "", fmt.Errorf("error decoding post: %w", err)
		}

		lines = append(lines, fmt.Sprintf("%d. %s (r/%s, %d points, %d comments, %s)", i+1, post.Title, post.Subreddit, post.Score, post.NumComments, time.Unix(int64(post.CreatedUTC), 0).Format("2006-01-02")))
		lines = append(lines, "   permalink: "+post.Permalink)

		body := strings.TrimSpace(post.SelfText)
		if body != "" {
			runes := []rune(body)
			if len(runes) > 500 {
				body = string(runes[:500]) + "…"
			}
			lines = append(lines, "   "+strings.ReplaceAll(body, "\n", "\n   "))
		}
	}

	if len(lines) == 0 {
		return "No posts.", nil
	}

	return strings.Join(lines, "\n"), nil
}

func getReddit(ctx context.Context, path string, params url.Values, target any) error {
	token, err := getRedditToken(ctx, false)
	if err != nil {
		return err
	}

	err = sendRedditRequest(ctx, token, path, params, target)

	// Tokens last an hour, so a 401 most likely means it expired early and one refresh is worth trying.
	var statusErr httpStatusError
	if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusUnauthorized {
		token, err = getRedditToken(ctx, true)
		if err != nil {
			return err
		}
		return sendRedditRequest(ctx, token, path, params, target)
	}

	return err
}

func sendRedditRequest(ctx context.Context, token, path string, params url.Values, target any) error {
	headers := map[string]string{"Authorization": "Bearer " + token, "User-Agent": getRedditUserAgent()}
	return sendJSONRequest(ctx, http.MethodGet, redditAPIURL+path+"?"+params.Encode(), headers, nil, target)
}

func getRedditToken(ctx context.Context, shouldRefresh bool) (string, error) {
	redditTokenMutex.Lock()
	defer redditTokenMutex.Unlock()

	if !shouldRefresh && redditToken != "" && time.Now().Before(redditTokenExpiresAt) {
		return redditToken, nil
	}

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", os.Getenv("REDDIT_USERNAME"))
	form.Set("password", os.Getenv("REDDIT_PASSWORD"))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, redditAuthURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("error creating reddit token request: %w", err)
	}
	request.SetBasicAuth(os.Getenv("REDDIT_CLIENT_ID"), os.Getenv("REDDIT_CLIENT_SECRET"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", getRedditUserAgent())

	response, err := httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("error requesting reddit token: %w", err)
	}
	defer response.Body.Close()

	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("error decoding reddit token: %w", err)
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("reddit login failed: %s", firstNonEmpty(body.Error, response.Status))
	}

	redditToken = body.AccessToken
	redditTokenExpiresAt = time.Now().Add(time.Duration(body.ExpiresIn)*time.Second - time.Minute)

	return redditToken, nil
}

func getRedditUserAgent() string {
	return firstNonEmpty(os.Getenv("REDDIT_USER_AGENT"), "script-squad")
}
