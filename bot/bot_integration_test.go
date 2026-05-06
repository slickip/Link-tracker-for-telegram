package bot_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httpserver "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/http"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/http/handlers"
)

type mockBot struct {
	messages []sentMessage
}

type sentMessage struct {
	chatID int64
	text   string
}

func (m *mockBot) SendMessage(chatID int64, text string) error {
	m.messages = append(m.messages, sentMessage{
		chatID: chatID,
		text:   text,
	})
	return nil
}

func setupBotRouter() http.Handler {
	bot := &mockBot{}
	updatesHandler := handlers.NewUpdatesHandler(bot)
	return httpserver.NewBotRouter(updatesHandler)
}

func TestBotUpdates_ValidRequest(t *testing.T) {
	bot := &mockBot{}
	updatesHandler := handlers.NewUpdatesHandler(bot)
	router := httpserver.NewBotRouter(updatesHandler)

	body := map[string]any{
		"url":       "https://github.com/golang/go",
		"tgChatIds": []int64{1, 2},
		"description": "test description",
	}

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	if len(bot.messages) != 2 {
		t.Fatalf("expected 2 messages sent, got %d", len(bot.messages))
	}
	for _, msg := range bot.messages {
		if msg.text != "test description" {
			t.Fatalf("expected message text=%q, got %q", "test description", msg.text)
		}
	}
}

func TestBotUpdates_InvalidRequest(t *testing.T) {
	router := setupBotRouter()

	body := map[string]any{
		"wrongField": "oops",
	}

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("expected non-200 response, got %d", w.Code)
	}
}
