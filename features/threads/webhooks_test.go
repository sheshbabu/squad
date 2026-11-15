package threads

import (
	"net/http"
	"net/http/httptest"
	"squad/commons/testutil"
	"squad/features/bots"
	"strings"
	"testing"
)

func postWebhook(token, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/hooks/"+token+"/", strings.NewReader(body))
	request.SetPathValue("token", token)
	recorder := httptest.NewRecorder()
	HandleReceiveWebhook(recorder, request)
	return recorder
}

func TestWebhookStartsChatWithBot(t *testing.T) {
	testutil.SetupDB(t)

	bot, err := bots.CreateBot(bots.Bot{Name: "Finance", Color: "gray", Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bots.UpdateBotWebhookToken(bot.BotID, "secret"); err != nil {
		t.Fatal(err)
	}

	if postWebhook("wrong", "hi").Code != http.StatusNotFound || postWebhook("", "hi").Code != http.StatusNotFound {
		t.Fatal("expected unknown and empty tokens to be rejected")
	}
	if postWebhook("secret", "  ").Code != http.StatusBadRequest {
		t.Fatal("expected an empty message to be rejected")
	}

	if code := postWebhook("secret", "Log lunch, $14").Code; code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", code)
	}

	threadList, err := GetThreads(bot.BotID)
	if err != nil {
		t.Fatal(err)
	}
	if len(threadList) != 1 || threadList[0].Status != STATUS_WORKING {
		t.Fatalf("expected one working chat, got %+v", threadList)
	}
}
