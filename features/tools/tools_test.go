package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"squad/commons/llm"
	"squad/commons/testutil"
	"squad/features/bots"
	"squad/features/connections"
	"strings"
	"testing"
)

func TestHTMLToText(t *testing.T) {
	page := `<html><head><title>Trip &amp; plans</title><style>p{}</style></head>
		<body><script>alert(1)</script><h1>Day one</h1><p>Visit the <b>museum</b>.</p><ul><li>Lunch</li><li>Park</li></ul></body></html>`

	text := htmlToText(page)
	expected := "# Trip & plans\n\nDay one\n\nVisit the museum.\n\nLunch\n\nPark"
	if text != expected {
		t.Fatalf("unexpected text:\n%q", text)
	}
}

func TestFetchFallsBackToExaWhenBlocked(t *testing.T) {
	blockedSite := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer blockedSite.Close()

	exa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/contents" || r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("unexpected exa request %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"results":[{"title":"Cached","url":"x","text":"From Exa"}],"statuses":[{"id":"x","status":"success"}]}`)
	}))
	defer exa.Close()

	exaBaseURL = exa.URL
	t.Setenv("EXA_API_KEY", "test-key")

	// The test sites listen on localhost, which the real fetch client refuses.
	defaultFetchClient := fetchClient
	fetchClient = httpClient
	defer func() { fetchClient = defaultFetchClient }()

	output, err := runFetchURL(context.Background(), bots.Bot{BotID: 1}, fmt.Sprintf(`{"url":%q}`, blockedSite.URL))
	if err != nil || output != "# Cached\n\nFrom Exa" {
		t.Fatalf("expected the Exa copy, got %q %v", output, err)
	}
}

func TestFetchRefusesNonPublicAddresses(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("a localhost site should never be reached")
	}))
	defer site.Close()

	_, err := runFetchURL(context.Background(), bots.Bot{BotID: 1}, fmt.Sprintf(`{"url":%q}`, site.URL))
	if err == nil || !strings.Contains(err.Error(), "non-public address") {
		t.Fatalf("expected a refusal, got %v", err)
	}

	for _, address := range []string{"169.254.169.254:80", "10.0.0.1:80", "192.168.1.1:443", "100.100.1.1:80", "[::1]:80", "[fd00::1]:80"} {
		if refuseNonPublicAddress("tcp", address, nil) == nil {
			t.Errorf("expected %s to be refused", address)
		}
	}

	if err := refuseNonPublicAddress("tcp", "93.184.215.14:443", nil); err != nil {
		t.Errorf("expected a public address to be allowed, got %v", err)
	}
}

func TestRedditRefreshesTokenOnce(t *testing.T) {
	tokenRequests := 0
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenRequests++
		fmt.Fprintf(w, `{"access_token":"token-%d","expires_in":3600}`, tokenRequests)
	}))
	defer auth.Close()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token-2" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, `{"data":{"children":[{"kind":"t3","data":{"title":"Hello","subreddit":"golang","score":5,"num_comments":1,"permalink":"/r/golang/comments/1/hello/"}}]}}`)
	}))
	defer api.Close()

	redditAuthURL = auth.URL
	redditAPIURL = api.URL
	redditToken = ""

	output, err := runRedditListing(context.Background(), bots.Bot{BotID: 1}, `{"subreddits":"golang"}`)
	if err != nil || !strings.Contains(output, "Hello") || tokenRequests != 2 {
		t.Fatalf("expected one refresh then success, got %q %v after %d token requests", output, err, tokenRequests)
	}
}

func setupZenBot(t *testing.T, zenURL string, tokensByName map[string]string) bots.Bot {
	t.Helper()
	testutil.SetupDB(t)
	t.Setenv("ZEN_URL", zenURL)
	registry = map[string]Tool{}
	registryOrder = []string{}
	registerMemoryTools()
	registerZenTools()

	bot, err := bots.CreateBot(bots.Bot{Name: "Helper", Color: "gray", Gateway: "fake", Model: "fake-model"})
	if err != nil {
		t.Fatal(err)
	}
	for name, token := range tokensByName {
		if _, err := connections.CreateConnection(connections.Connection{BotID: bot.BotID, GroupID: "zen", Name: name, Secret: token}); err != nil {
			t.Fatal(err)
		}
	}
	return bot
}

func TestZenForbiddenBecomesToolError(t *testing.T) {
	zen := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer bot-token" {
			t.Errorf("expected the connection's token, got %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"code":"FORBIDDEN_SCOPE","message":"no write access"}`)
	}))
	defer zen.Close()

	bot := setupZenBot(t, zen.URL, map[string]string{"Shesh": "bot-token"})

	output := Run(context.Background(), bot, llm.ToolCall{Function: llm.FunctionCall{Name: "zen_create_note", Arguments: `{"title":"x","content":"y","tagIds":[1]}`}})
	if !strings.HasPrefix(output, "Error:") || !strings.Contains(output, "FORBIDDEN_SCOPE") {
		t.Fatalf("expected a tool error, got %q", output)
	}
}

func TestZenPicksTheNamedConnection(t *testing.T) {
	var usedToken string
	zen := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		usedToken = r.Header.Get("Authorization")
		fmt.Fprint(w, `[]`)
	}))
	defer zen.Close()

	bot := setupZenBot(t, zen.URL, map[string]string{"Shesh": "shesh-token", "Ranjika": "ranjika-token"})

	definitions, err := GetDefinitions(bot)
	if err != nil {
		t.Fatal(err)
	}
	for _, definition := range definitions {
		if definition.Function.Name == "zen_list_tags" && !slices.Contains(definition.Function.Parameters.(map[string]any)["required"].([]string), "connection") {
			t.Fatal("expected a required connection argument with two connections")
		}
	}

	Run(context.Background(), bot, llm.ToolCall{Function: llm.FunctionCall{Name: "zen_list_tags", Arguments: `{"connection":"Ranjika"}`}})
	if usedToken != "Bearer ranjika-token" {
		t.Fatalf("expected Ranjika's token, got %q", usedToken)
	}

	output := Run(context.Background(), bot, llm.ToolCall{Function: llm.FunctionCall{Name: "zen_list_tags", Arguments: `{"connection":"Miransh"}`}})
	if !strings.Contains(output, "use one of") {
		t.Fatalf("expected an unknown connection error, got %q", output)
	}
}

func TestToolGroupsFollowBotSettings(t *testing.T) {
	bot := setupZenBot(t, "http://zen.test", map[string]string{})

	countGroup := func(bot bots.Bot, group string) int {
		definitions, err := GetDefinitions(bot)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, definition := range definitions {
			if registry[definition.Function.Name].Group == group {
				count++
			}
		}
		return count
	}

	if countGroup(bot, "memory") == 0 || countGroup(bot, "zen") != 0 {
		t.Fatal("expected memory on by default and zen off without a connection")
	}

	if _, err := connections.CreateConnection(connections.Connection{BotID: bot.BotID, GroupID: "zen", Name: "Shesh", Secret: "token"}); err != nil {
		t.Fatal(err)
	}
	if countGroup(bot, "zen") == 0 {
		t.Fatal("expected zen on once the bot has a connection")
	}

	bot.DisabledToolGroups = []string{"memory"}
	if countGroup(bot, "memory") != 0 {
		t.Fatal("expected memory off once disabled")
	}

	output := Run(context.Background(), bot, llm.ToolCall{Function: llm.FunctionCall{Name: "save_memory", Arguments: `{"content":"x"}`}})
	if !strings.Contains(output, "turned off") {
		t.Fatalf("expected a disabled tool to be refused, got %q", output)
	}
}

func TestTruncateKeepsHeadAndTail(t *testing.T) {
	text := "HEAD" + strings.Repeat("x", maxOutputLength*2) + "TAIL"
	output := truncate(text)

	if !strings.HasPrefix(output, "HEAD") || !strings.HasSuffix(output, "TAIL") || !strings.Contains(output, "characters cut]") {
		t.Fatalf("expected head, cut marker and tail, got %q…", output[:40])
	}
}
