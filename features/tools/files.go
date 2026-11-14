package tools

import (
	"context"
	"fmt"
	"squad/features/bots"
	"squad/features/files"
	"strings"
)

func registerFileTools() {
	nameProperty := stringProperty("The file name, ending in .md, for example packing-list.md.")

	register("files", Tool{
		Name:        "list_files",
		Description: "List the markdown files in your folder. Your folder is shared with the user, who can read and edit the files too.",
		Parameters:  objectSchema(map[string]any{}),
		Run:         runListFiles,
	})

	register("files", Tool{
		Name:        "read_file",
		Description: "Read a markdown file from your folder.",
		Parameters:  objectSchema(map[string]any{"name": nameProperty}, "name"),
		Run:         runReadFile,
	})

	register("files", Tool{
		Name:        "write_file",
		Description: "Create a markdown file in your folder, or replace its whole content. Prefer edit_file for small changes to an existing file.",
		Parameters: objectSchema(map[string]any{
			"name":    nameProperty,
			"content": stringProperty("The full markdown content of the file."),
		}, "name", "content"),
		Run: runWriteFile,
	})

	register("files", Tool{
		Name:        "edit_file",
		Description: "Replace one exact piece of text in a markdown file. oldText must appear exactly once, so include enough surrounding text to make it unique. Read the file first.",
		Parameters: objectSchema(map[string]any{
			"name":    nameProperty,
			"oldText": stringProperty("The exact text to replace."),
			"newText": stringProperty("The replacement text."),
		}, "name", "oldText", "newText"),
		Run: runEditFile,
	})

	register("files", Tool{
		Name:        "delete_file",
		Description: "Delete a markdown file from your folder.",
		Parameters:  objectSchema(map[string]any{"name": nameProperty}, "name"),
		Run:         runDeleteFile,
	})
}

func runListFiles(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	fileList, err := files.ListFiles(bot.BotID)
	if err != nil {
		return "", err
	}

	if len(fileList) == 0 {
		return "Your folder is empty.", nil
	}

	lines := []string{}
	for _, file := range fileList {
		lines = append(lines, fmt.Sprintf("%s (%d bytes, updated %s)", file.Name, file.Size, file.UpdatedAt.Format("2006-01-02 15:04")))
	}

	return strings.Join(lines, "\n"), nil
}

func runReadFile(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		Name string `json:"name"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	fileContent, err := files.ReadFile(bot.BotID, args.Name)
	if err != nil {
		return "", fmt.Errorf("could not read %s: %w", args.Name, err)
	}

	return fileContent.Content, nil
}

func runWriteFile(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	if err := files.WriteFile(bot.BotID, args.Name, args.Content); err != nil {
		return "", err
	}

	return fmt.Sprintf("Saved %s.", args.Name), nil
}

func runEditFile(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		Name    string `json:"name"`
		OldText string `json:"oldText"`
		NewText string `json:"newText"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	if err := files.EditFile(bot.BotID, args.Name, args.OldText, args.NewText); err != nil {
		return "", err
	}

	return fmt.Sprintf("Edited %s.", args.Name), nil
}

func runDeleteFile(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		Name string `json:"name"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	if err := files.DeleteFile(bot.BotID, args.Name); err != nil {
		return "", err
	}

	return fmt.Sprintf("Deleted %s.", args.Name), nil
}
