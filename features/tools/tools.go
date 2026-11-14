package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"squad/commons/llm"
	"squad/features/bots"
	"time"
)

const maxOutputLength = 20000

type Tool struct {
	Group       string
	Name        string
	Description string
	Parameters  map[string]any
	Run         func(ctx context.Context, bot bots.Bot, arguments string) (string, error)
}

var registry = map[string]Tool{}
var registryOrder = []string{}

var httpClient = &http.Client{Timeout: 60 * time.Second}

// Init registers every tool whose credentials are present, so a missing key hides a tool instead of failing every call.
func Init() {
	registerMemoryTools()
	registerFileTools()
	registerFetchTool()
	registerHackerNewsTools()
	registerExaTools()
	registerRedditTools()
	registerZenTools()
}

func register(group string, tool Tool) {
	tool.Group = group
	registry[tool.Name] = tool
	registryOrder = append(registryOrder, tool.Name)
}

func GetDefinitions(bot bots.Bot) []llm.Tool {
	definitions := []llm.Tool{}
	for _, name := range registryOrder {
		tool := registry[name]
		if !IsGroupEnabled(bot, tool.Group) {
			continue
		}
		definitions = append(definitions, llm.Tool{
			Type:     "function",
			Function: llm.ToolFunction{Name: tool.Name, Description: tool.Description, Parameters: tool.Parameters},
		})
	}
	return definitions
}

// Run never fails the agent run, because a tool error is information the model can act on.
func Run(ctx context.Context, bot bots.Bot, toolCall llm.ToolCall) string {
	tool, ok := registry[toolCall.Function.Name]
	if !ok {
		return fmt.Sprintf("Error: unknown tool %q", toolCall.Function.Name)
	}

	// Earlier replies in the chat may show a tool that has since been turned off, and the model can still try to call it.
	if !IsGroupEnabled(bot, tool.Group) {
		return fmt.Sprintf("Error: tool %q is turned off for this bot", toolCall.Function.Name)
	}

	arguments := toolCall.Function.Arguments
	if arguments == "" {
		arguments = "{}"
	}

	output, err := tool.Run(ctx, bot, arguments)
	if err != nil {
		return "Error: " + err.Error()
	}

	return truncate(output)
}

func truncate(text string) string {
	runes := []rune(text)
	if len(runes) <= maxOutputLength {
		return text
	}
	return string(runes[:maxOutputLength]) + "\n\n[truncated]"
}

func parseArguments(arguments string, target any) error {
	if err := json.Unmarshal([]byte(arguments), target); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	return nil
}

func toJSON(value any) (string, error) {
	output, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error encoding result: %w", err)
	}
	return string(output), nil
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "object", "properties": properties, "required": required}
}

func stringProperty(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func integerProperty(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
}

func enumProperty(description string, values ...string) map[string]any {
	return map[string]any{"type": "string", "description": description, "enum": values}
}

type httpStatusError struct {
	StatusCode int
	Body       string
}

func (e httpStatusError) Error() string {
	return fmt.Sprintf("request failed with status %d: %s", e.StatusCode, e.Body)
}

func sendJSONRequest(ctx context.Context, method, url string, headers map[string]string, body any, target any) error {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("error encoding request: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	request, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 2000))
		return httpStatusError{StatusCode: response.StatusCode, Body: string(responseBody)}
	}

	if target == nil {
		return nil
	}

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	return nil
}
