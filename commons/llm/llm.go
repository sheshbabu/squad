package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"
)

// Gateway is any OpenAI-compatible chat completions API, so adding Ollama later is a new entry here and nothing else.
type Gateway struct {
	Name                    string
	BaseURL                 string
	APIKey                  string
	SupportsProviderRouting bool
}

type Message struct {
	Role             string           `json:"role"`
	Content          any              `json:"content"`
	ToolCalls        []ToolCall       `json:"tool_calls,omitempty"`
	ToolCallID       string           `json:"tool_call_id,omitempty"`
	ReasoningDetails []map[string]any `json:"reasoning_details,omitempty"`
}

type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
	File     *File     `json:"file,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type File struct {
	Filename string `json:"filename"`
	FileData string `json:"file_data"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type Request struct {
	Model           string
	Provider        string
	ReasoningEffort string
	User            string
	Messages        []Message
	Tools           []Tool
}

type Response struct {
	Content          string
	ToolCalls        []ToolCall
	ReasoningDetails []map[string]any
	PromptTokens     int
	CompletionTokens int
	Cost             float64
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e APIError) Error() string {
	return fmt.Sprintf("gateway error %d: %s", e.StatusCode, e.Message)
}

var gateways = map[string]Gateway{}

var httpClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ResponseHeaderTimeout: 2 * time.Minute,
	},
}

func Init() {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey != "" {
		RegisterGateway(Gateway{Name: "openrouter", BaseURL: "https://openrouter.ai/api/v1", APIKey: apiKey, SupportsProviderRouting: true})
	}
}

func RegisterGateway(gateway Gateway) {
	gateways[gateway.Name] = gateway
}

func GetGateway(name string) (Gateway, bool) {
	gateway, ok := gateways[name]
	return gateway, ok
}

func GetGatewayNames() []string {
	names := []string{}
	for name := range gateways {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// IsRetryable treats rate limits, server errors and broken connections as transient, and other API errors as needing a human.
func IsRetryable(err error) bool {
	var apiErr APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusRequestTimeout || apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= 500
	}

	return !errors.Is(err, context.Canceled)
}

func (g Gateway) Complete(ctx context.Context, request Request) (Response, error) {
	var response Response

	httpResponse, err := g.post(ctx, "/chat/completions", g.buildBody(request, false))
	if err != nil {
		return response, err
	}
	defer httpResponse.Body.Close()

	var body struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage *usage `json:"usage"`
	}
	if err := json.NewDecoder(httpResponse.Body).Decode(&body); err != nil {
		return response, fmt.Errorf("error decoding completion: %w", err)
	}

	if len(body.Choices) > 0 {
		response.Content = body.Choices[0].Message.Content
	}
	if body.Usage != nil {
		response.PromptTokens = body.Usage.PromptTokens
		response.CompletionTokens = body.Usage.CompletionTokens
		response.Cost = body.Usage.Cost
	}

	return response, nil
}

func (g Gateway) buildBody(request Request, isStream bool) map[string]any {
	body := map[string]any{
		"model":    request.Model,
		"messages": request.Messages,
		"stream":   isStream,
	}

	if len(request.Tools) > 0 {
		body["tools"] = request.Tools
	}

	if request.User != "" {
		body["user"] = request.User
	}

	if g.SupportsProviderRouting {
		body["usage"] = map[string]any{"include": true}
		if request.Provider != "" {
			body["provider"] = map[string]any{"order": []string{request.Provider}, "allow_fallbacks": false}
		}
		if request.ReasoningEffort != "" {
			body["reasoning"] = map[string]any{"effort": request.ReasoningEffort}
		}
	}

	return body
}

func (g Gateway) post(ctx context.Context, path string, body any) (*http.Response, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error encoding request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, g.BaseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+g.APIKey)

	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("error calling gateway: %w", err)
	}

	if httpResponse.StatusCode >= 300 {
		defer httpResponse.Body.Close()
		return nil, readAPIError(httpResponse)
	}

	return httpResponse, nil
}

func (g Gateway) get(ctx context.Context, path string, target any) error {
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, g.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+g.APIKey)

	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("error calling gateway: %w", err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode >= 300 {
		return readAPIError(httpResponse)
	}

	if err := json.NewDecoder(httpResponse.Body).Decode(target); err != nil {
		return fmt.Errorf("error decoding gateway response: %w", err)
	}

	return nil
}

func readAPIError(httpResponse *http.Response) error {
	bodyBytes, _ := io.ReadAll(io.LimitReader(httpResponse.Body, 64*1024))

	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	message := string(bodyBytes)
	if json.Unmarshal(bodyBytes, &body) == nil && body.Error.Message != "" {
		message = body.Error.Message
	}

	return APIError{StatusCode: httpResponse.StatusCode, Message: message}
}

type usage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	Cost             float64 `json:"cost"`
}
