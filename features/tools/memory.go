package tools

import (
	"context"
	"fmt"
	"squad/features/bots"
	"squad/features/memories"
	"strings"
)

func registerMemoryTools() {
	register("memory", Tool{
		Name:        "save_memory",
		Description: "Save a fact worth remembering across all future conversations with this bot, such as a preference, a decision or a detail about someone. Saved memories appear in your instructions with their ids.",
		Parameters: objectSchema(map[string]any{
			"content": stringProperty("The fact to remember, written as one self-contained sentence."),
		}, "content"),
		Run: runSaveMemory,
	})

	register("memory", Tool{
		Name:        "delete_memory",
		Description: "Delete a saved memory that is wrong or no longer true. To change a memory, delete it and save the new version.",
		Parameters: objectSchema(map[string]any{
			"memoryId": integerProperty("The id shown next to the memory in your instructions."),
		}, "memoryId"),
		Run: runDeleteMemory,
	})
}

func runSaveMemory(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		Content string `json:"content"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	content := strings.TrimSpace(args.Content)
	if content == "" {
		return "", fmt.Errorf("content is required")
	}

	memory, err := memories.CreateMemory(bot.BotID, content)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Saved memory %d.", memory.MemoryID), nil
}

func runDeleteMemory(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		MemoryID int `json:"memoryId"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	if err := memories.DeleteMemory(bot.BotID, args.MemoryID); err != nil {
		return "", fmt.Errorf("memory %d was not found", args.MemoryID)
	}

	return fmt.Sprintf("Deleted memory %d.", args.MemoryID), nil
}
