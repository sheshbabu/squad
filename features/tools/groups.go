package tools

import (
	"fmt"
	"net/http"
	"slices"
	"squad/commons/utils"
	"squad/features/bots"
	"squad/features/connections"
	"strings"
)

type ToolGroup struct {
	GroupID     string          `json:"groupId"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	IsAvailable bool            `json:"isAvailable"`
	SetupHint   string          `json:"setupHint"`
	Connection  *ConnectionSpec `json:"connection"`
}

type ConnectionSpec struct {
	SecretLabel string `json:"secretLabel"`
	Hint        string `json:"hint"`
}

var groups = []ToolGroup{
	{GroupID: "memory", Name: "Memory", Description: "Save and forget facts across chats."},
	{GroupID: "files", Name: "Files", Description: "Read and write the bot's markdown files."},
	{GroupID: "chats", Name: "Past Chats", Description: "Search earlier chats with this bot."},
	{GroupID: "web", Name: "Web", Description: "Read web pages, and search the web when EXA_API_KEY is set."},
	{GroupID: "hackernews", Name: "Hacker News", Description: "Search stories and read comments."},
	{GroupID: "reddit", Name: "Reddit", Description: "Search posts and read comments.", SetupHint: "Set the REDDIT_ environment variables to use Reddit."},
	{GroupID: "zen", Name: "Zen", Description: "Search, read and write notes in Zen.", SetupHint: "Set ZEN_URL to use Zen.", Connection: &ConnectionSpec{SecretLabel: "API token", Hint: "Create a token in Zen under Settings, API Tokens. Its tag scopes decide which notes this connection can read and write."}},
}

func HandleGetToolGroups(w http.ResponseWriter, r *http.Request) {
	response := []ToolGroup{}
	for _, group := range groups {
		group.IsAvailable = isGroupRegistered(group.GroupID)
		response = append(response, group)
	}

	utils.SendJSONResponse(w, http.StatusOK, response)
}

func isGroupEnabled(bot bots.Bot, botConnections []connections.Connection, groupID string) bool {
	if slices.Contains(bot.DisabledToolGroups, groupID) {
		return false
	}
	if getGroup(groupID).Connection != nil && len(getGroupConnections(botConnections, groupID)) == 0 {
		return false
	}
	return true
}

func getGroup(groupID string) ToolGroup {
	for _, group := range groups {
		if group.GroupID == groupID {
			return group
		}
	}
	return ToolGroup{}
}

func getGroupConnections(botConnections []connections.Connection, groupID string) []connections.Connection {
	groupConnections := []connections.Connection{}
	for _, connection := range botConnections {
		if connection.GroupID == groupID {
			groupConnections = append(groupConnections, connection)
		}
	}
	return groupConnections
}

func pickConnection(botID int, groupID string, arguments string) (connections.Connection, error) {
	botConnections, err := connections.GetConnections(botID)
	if err != nil {
		return connections.Connection{}, err
	}

	groupConnections := getGroupConnections(botConnections, groupID)
	if len(groupConnections) == 1 {
		return groupConnections[0], nil
	}

	var args struct {
		Connection string `json:"connection"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return connections.Connection{}, err
	}

	names := []string{}
	for _, connection := range groupConnections {
		if connection.Name == args.Connection {
			return connection, nil
		}
		names = append(names, connection.Name)
	}

	return connections.Connection{}, fmt.Errorf("unknown connection %q, use one of: %s", args.Connection, strings.Join(names, ", "))
}

func isGroupRegistered(groupID string) bool {
	for _, tool := range registry {
		if tool.Group == groupID {
			return true
		}
	}
	return false
}
