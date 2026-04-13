package scrapper_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/application/services"
	httpserver "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/http"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/http/handlers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/repositories"
)

func setupScrapperRouter() http.Handler {
	chatRepo := repositories.NewInMemoryChatRepository()
	linkRepo := repositories.NewInMemoryLinkRepository()

	chatService := services.NewChatService(chatRepo)
	linkService := services.NewLinkService(linkRepo, chatRepo)

	chatHandler := handlers.NewChatHandler(chatService)
	linkHandler := handlers.NewLinkHandler(linkService)

	return httpserver.NewRouter(chatHandler, linkHandler)
}

func TestScrapper_AddAndGetLink(t *testing.T) {
	router := setupScrapperRouter()

	req := httptest.NewRequest(http.MethodPost, "/tg-chat/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for chat registration, got %d", w.Code)
	}

	body := map[string]any{
		"chatId": 1,
		"url":    "https://github.com/golang/go",
		"tags":   []string{"test"},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/links", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for add link, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/links?chatId=1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for get links, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "https://github.com/golang/go") {
		t.Fatalf("expected response to contain added link, got %s", w.Body.String())
	}
}

func TestScrapper_AddAndDeleteLink(t *testing.T) {
	router := setupScrapperRouter()

	req := httptest.NewRequest(http.MethodPost, "/tg-chat/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for chat registration, got %d", w.Code)
	}

	body := map[string]any{
		"chatId": 1,
		"url":    "https://github.com/golang/go",
		"tags":   []string{"test"},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/links", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for add link, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/links", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for delete link, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/links?chatId=1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for get links, got %d", w.Code)
	}

	if strings.Contains(w.Body.String(), "https://github.com/golang/go") {
		t.Fatalf("expected deleted link to be absent, got %s", w.Body.String())
	}
}

func TestScrapper_DeleteLinkFromNonExistingChat(t *testing.T) {
	router := setupScrapperRouter()

	req := httptest.NewRequest(http.MethodPost, "/tg-chat/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for chat registration, got %d", w.Code)
	}

	body := map[string]any{
		"chatId": 1,
		"url":    "https://github.com/golang/go",
		"tags":   []string{"test"},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/links", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for add link, got %d", w.Code)
	}

	wrongBody := map[string]any{
		"chatId": 999,
		"url":    "https://github.com/golang/go",
	}
	wrongData, err := json.Marshal(wrongBody)
	if err != nil {
		t.Fatalf("failed to marshal wrong body: %v", err)
	}

	req = httptest.NewRequest(http.MethodDelete, "/links", bytes.NewBuffer(wrongData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("expected non-200 when deleting from non-existing chat")
	}

	req = httptest.NewRequest(http.MethodGet, "/links?chatId=1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for get links, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "https://github.com/golang/go") {
		t.Fatalf("expected original link to remain, got %s", w.Body.String())
	}
}

func TestScrapper_AddLinkToNonExistingChat(t *testing.T) {
	router := setupScrapperRouter()

	req := httptest.NewRequest(http.MethodPost, "/tg-chat/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for chat registration, got %d", w.Code)
	}

	body := map[string]any{
		"chatId": 2,
		"url":    "https://github.com/golang/go",
		"tags":   []string{"test"},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/links", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("expected non-200 when adding link to non-existing chat")
	}
}

func TestScrapper_DeletedChatCannotAddLinks(t *testing.T) {
	router := setupScrapperRouter()

	req := httptest.NewRequest(http.MethodPost, "/tg-chat/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for chat registration, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/tg-chat/1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for chat deletion, got %d", w.Code)
	}

	body := map[string]any{
		"chatId": 1,
		"url":    "https://github.com/golang/go",
		"tags":   []string{"test"},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/links", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("expected non-200 when adding link to deleted chat")
	}
}

func TestScrapper_DeleteNonExistingChat(t *testing.T) {
	router := setupScrapperRouter()

	req := httptest.NewRequest(http.MethodDelete, "/tg-chat/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for deleting non-existing chat, got %d", w.Code)
	}
}
