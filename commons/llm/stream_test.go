package llm

import (
	"errors"
	"strings"
	"testing"
)

func TestParseStreamAccumulatesTextToolCallsAndUsage(t *testing.T) {
	body := strings.Join([]string{
		": OPENROUTER PROCESSING",
		"",
		`data: {"choices":[{"delta":{"content":"Hel"}}]}`,
		`data: {"choices":[{"delta":{"content":"lo"}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"web_search","arguments":"{\"que"}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ry\":\"go\"}"}}]}}]}`,
		`data: {"choices":[{"delta":{"reasoning_details":[{"type":"reasoning.text","index":0,"text":"Thin"}]}}]}`,
		`data: {"choices":[{"delta":{"reasoning_details":[{"type":"reasoning.text","index":0,"text":"king","signature":"sig"}]}}]}`,
		`data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"cost":0.001}}`,
		"data: [DONE]",
	}, "\n")

	deltas := []string{}
	response, err := parseStream(strings.NewReader(body), func(delta string) { deltas = append(deltas, delta) }, func() {})
	if err != nil {
		t.Fatal(err)
	}

	if response.Content != "Hello" || strings.Join(deltas, "|") != "Hel|lo" {
		t.Fatalf("unexpected content %q deltas %v", response.Content, deltas)
	}

	if len(response.ToolCalls) != 1 || response.ToolCalls[0].ID != "call_1" || response.ToolCalls[0].Function.Arguments != `{"query":"go"}` {
		t.Fatalf("unexpected tool calls %+v", response.ToolCalls)
	}

	if len(response.ReasoningDetails) != 1 || response.ReasoningDetails[0]["text"] != "Thinking" || response.ReasoningDetails[0]["signature"] != "sig" {
		t.Fatalf("unexpected reasoning %+v", response.ReasoningDetails)
	}

	if response.PromptTokens != 10 || response.CompletionTokens != 5 || response.Cost != 0.001 {
		t.Fatalf("unexpected usage %+v", response)
	}
}

func TestParseStreamErrorChunkIsAPIError(t *testing.T) {
	body := `data: {"error":{"code":429,"message":"slow down"},"choices":[{"delta":{},"finish_reason":"error"}]}`

	_, err := parseStream(strings.NewReader(body), func(string) {}, func() {})

	var apiErr APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 429 || !IsRetryable(err) {
		t.Fatalf("expected retryable 429, got %v", err)
	}
}

func TestParseStreamWithoutDoneIsRetryable(t *testing.T) {
	body := `data: {"choices":[{"delta":{"content":"Hi"}}]}`

	_, err := parseStream(strings.NewReader(body), func(string) {}, func() {})
	if !errors.Is(err, errStreamEnded) || !IsRetryable(err) {
		t.Fatalf("expected retryable early end, got %v", err)
	}
}

func TestClientErrorsAreNotRetryable(t *testing.T) {
	if IsRetryable(APIError{StatusCode: 401}) || IsRetryable(APIError{StatusCode: 400}) {
		t.Fatal("4xx errors should need a human")
	}
	if !IsRetryable(APIError{StatusCode: 503}) {
		t.Fatal("5xx errors should be retried")
	}
}
