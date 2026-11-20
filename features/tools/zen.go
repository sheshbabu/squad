package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"squad/features/bots"
	"strconv"
	"strings"
)

type zenTag struct {
	TagID int    `json:"tagId"`
	Name  string `json:"name"`
}

type zenNote struct {
	NoteID    int      `json:"noteId"`
	Title     string   `json:"title"`
	Snippet   string   `json:"snippet"`
	Content   string   `json:"content"`
	Tags      []zenTag `json:"tags"`
	UpdatedAt string   `json:"updatedAt"`
}

type zenNoteInput struct {
	Title   string           `json:"title"`
	Content string           `json:"content"`
	Tags    []map[string]int `json:"tags"`
}

func registerZenTools() {
	if os.Getenv("ZEN_URL") == "" {
		return
	}

	tagIDsProperty := map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "Tag ids from zen_list_tags. Writes are only allowed on notes whose tags the token can write to."}

	register("zen", Tool{
		Name:        "zen_search_notes",
		Description: "Search the notes in Zen, the user's notes app, by keywords.",
		Parameters: objectSchema(map[string]any{
			"query": stringProperty("Search keywords."),
		}, "query"),
		Run: runZenSearchNotes,
	})

	register("zen", Tool{
		Name:        "zen_get_note",
		Description: "Read a note from Zen in full.",
		Parameters: objectSchema(map[string]any{
			"noteId": integerProperty("The note id from a search result."),
		}, "noteId"),
		Run: runZenGetNote,
	})

	register("zen", Tool{
		Name:        "zen_list_tags",
		Description: "List the Zen tags you can see, with their ids.",
		Parameters:  objectSchema(map[string]any{}),
		Run:         runZenListTags,
	})

	register("zen", Tool{
		Name:        "zen_create_note",
		Description: "Create a markdown note in Zen.",
		Parameters: objectSchema(map[string]any{
			"title":   stringProperty("The note title."),
			"content": stringProperty("The markdown content."),
			"tagIds":  tagIDsProperty,
		}, "title", "content", "tagIds"),
		Run: runZenCreateNote,
	})

	register("zen", Tool{
		Name:        "zen_update_note",
		Description: "Replace a Zen note's title, content and tags. Read the note first and send back everything you want to keep.",
		Parameters: objectSchema(map[string]any{
			"noteId":  integerProperty("The note id."),
			"title":   stringProperty("The note title."),
			"content": stringProperty("The full markdown content."),
			"tagIds":  tagIDsProperty,
		}, "noteId", "title", "content", "tagIds"),
		Run: runZenUpdateNote,
	})
}

func runZenSearchNotes(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	connection, err := pickConnection(bot.BotID, "zen", arguments)
	if err != nil {
		return "", err
	}

	var response struct {
		LexicalNotes []zenNote `json:"lexicalNotes"`
	}
	if err := callZen(ctx, connection.Secret, http.MethodGet, "/api/v1/search/?query="+url.QueryEscape(args.Query), nil, &response); err != nil {
		return "", err
	}

	if len(response.LexicalNotes) == 0 {
		return "No notes found.", nil
	}

	lines := []string{}
	for _, note := range response.LexicalNotes {
		lines = append(lines, fmt.Sprintf("- [%d] %s (%s): %s", note.NoteID, note.Title, formatZenTags(note.Tags), note.Snippet))
	}

	return strings.Join(lines, "\n"), nil
}

func runZenGetNote(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		NoteID int `json:"noteId"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	connection, err := pickConnection(bot.BotID, "zen", arguments)
	if err != nil {
		return "", err
	}

	var note zenNote
	if err := callZen(ctx, connection.Secret, http.MethodGet, "/api/v1/notes/"+strconv.Itoa(args.NoteID)+"/", nil, &note); err != nil {
		return "", err
	}

	return formatZenNote(note), nil
}

func runZenListTags(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	connection, err := pickConnection(bot.BotID, "zen", arguments)
	if err != nil {
		return "", err
	}

	var tags []zenTag
	if err := callZen(ctx, connection.Secret, http.MethodGet, "/api/v1/tags/", nil, &tags); err != nil {
		return "", err
	}

	if len(tags) == 0 {
		return "No tags.", nil
	}

	lines := []string{}
	for _, tag := range tags {
		lines = append(lines, fmt.Sprintf("- [%d] %s", tag.TagID, tag.Name))
	}

	return strings.Join(lines, "\n"), nil
}

func runZenCreateNote(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		TagIDs  []int  `json:"tagIds"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	connection, err := pickConnection(bot.BotID, "zen", arguments)
	if err != nil {
		return "", err
	}

	var note zenNote
	if err := callZen(ctx, connection.Secret, http.MethodPost, "/api/v1/notes/", buildZenNoteInput(args.Title, args.Content, args.TagIDs), &note); err != nil {
		return "", err
	}

	return fmt.Sprintf("Created note %d.", note.NoteID), nil
}

func runZenUpdateNote(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		NoteID  int    `json:"noteId"`
		Title   string `json:"title"`
		Content string `json:"content"`
		TagIDs  []int  `json:"tagIds"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	connection, err := pickConnection(bot.BotID, "zen", arguments)
	if err != nil {
		return "", err
	}

	var note zenNote
	if err := callZen(ctx, connection.Secret, http.MethodPut, "/api/v1/notes/"+strconv.Itoa(args.NoteID)+"/", buildZenNoteInput(args.Title, args.Content, args.TagIDs), &note); err != nil {
		return "", err
	}

	return fmt.Sprintf("Updated note %d.", note.NoteID), nil
}

func buildZenNoteInput(title, content string, tagIDs []int) zenNoteInput {
	tags := []map[string]int{}
	for _, tagID := range tagIDs {
		tags = append(tags, map[string]int{"tagId": tagID})
	}
	return zenNoteInput{Title: title, Content: content, Tags: tags}
}

func callZen(ctx context.Context, apiToken, method, path string, body any, target any) error {
	baseURL := strings.TrimSuffix(os.Getenv("ZEN_URL"), "/")
	headers := map[string]string{"Authorization": "Bearer " + apiToken}
	return sendJSONRequest(ctx, method, baseURL+path, headers, body, target)
}

func formatZenNote(note zenNote) string {
	return fmt.Sprintf("# %s\n\nNote id: %d\nTags: %s\nUpdated: %s\n\n%s", note.Title, note.NoteID, formatZenTags(note.Tags), note.UpdatedAt, note.Content)
}

func formatZenTags(tags []zenTag) string {
	names := []string{}
	for _, tag := range tags {
		names = append(names, fmt.Sprintf("%s [%d]", tag.Name, tag.TagID))
	}
	if len(names) == 0 {
		return "no tags"
	}
	return strings.Join(names, ", ")
}
