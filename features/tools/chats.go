package tools

import (
	"context"
	"fmt"
	"squad/features/bots"
	"squad/features/search"
)

func registerChatTools() {
	register("chats", Tool{
		Name:        "search_chats",
		Description: "Search earlier chats with this bot by keywords, to recall what was discussed or decided. Returns the best matching messages with their chat title and date.",
		Parameters: objectSchema(map[string]any{
			"query": stringProperty("Keywords to look for, such as names, topics or terms likely used in the chat."),
		}, "query"),
		Run: runSearchChats,
	})
}

func runSearchChats(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	matchQuery := search.BuildAnyWordQuery(args.Query)
	if matchQuery == "" {
		return "", fmt.Errorf("query is required")
	}

	matches, err := search.SearchBotMessages(bot.BotID, matchQuery)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "No earlier chats match.", nil
	}

	return toJSON(matches)
}
