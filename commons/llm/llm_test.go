package llm

import "testing"

func TestBuildBodySendsReasoningEffortOnlyWhenSet(t *testing.T) {
	gateway := Gateway{SupportsProviderRouting: true}

	body := gateway.buildBody(Request{Model: "m", ReasoningEffort: "low"}, true)
	if reasoning, ok := body["reasoning"].(map[string]any); !ok || reasoning["effort"] != "low" {
		t.Fatalf("expected reasoning effort low, got %v", body["reasoning"])
	}

	body = gateway.buildBody(Request{Model: "m"}, true)
	if _, ok := body["reasoning"]; ok {
		t.Fatal("expected no reasoning field when the bot uses the model default")
	}
}
