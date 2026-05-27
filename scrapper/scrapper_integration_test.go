package scrapper_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	httpserver "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/http"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/http/handlers"
)

type fakeChatRepository struct {
	chats map[int64]struct{}
}

func newFakeChatRepository() *fakeChatRepository {
	return &fakeChatRepository{chats: make(map[int64]struct{})}
}

func (r *fakeChatRepository) Add(ctx context.Context, chatID int64) error {
	r.chats[chatID] = struct{}{}
	return nil
}

func (r *fakeChatRepository) Remove(ctx context.Context, chatID int64) error {
	if _, ok := r.chats[chatID]; !ok {
		return pkg.ErrChatNotFound
	}
	delete(r.chats, chatID)
	return nil
}

func (r *fakeChatRepository) Exists(ctx context.Context, chatID int64) (bool, error) {
	_, ok := r.chats[chatID]
	return ok, nil
}

type fakeLinkRepository struct {
	linksByChat map[int64][]domain.Link
}

func newFakeLinkRepository() *fakeLinkRepository {
	return &fakeLinkRepository{linksByChat: make(map[int64][]domain.Link)}
}

func (r *fakeLinkRepository) Add(ctx context.Context, chatID int64, link domain.Link) error {
	r.linksByChat[chatID] = append(r.linksByChat[chatID], link)
	return nil
}

func (r *fakeLinkRepository) Remove(ctx context.Context, chatID int64, url string) error {
	links := r.linksByChat[chatID]
	for i, link := range links {
		if link.URL == url {
			r.linksByChat[chatID] = append(links[:i], links[i+1:]...)
			return nil
		}
	}
	return pkg.ErrLinkNotFound
}

func (r *fakeLinkRepository) RemoveByTag(ctx context.Context, chatID int64, tag string) (int64, error) {
	links := r.linksByChat[chatID]
	filtered := make([]domain.Link, 0, len(links))
	var removed int64
	for _, link := range links {
		hasTag := false
		for _, currentTag := range link.Tags {
			if currentTag == tag {
				hasTag = true
				break
			}
		}
		if hasTag {
			removed++
			continue
		}
		filtered = append(filtered, link)
	}
	r.linksByChat[chatID] = filtered
	return removed, nil
}

func (r *fakeLinkRepository) List(ctx context.Context, chatID int64) ([]domain.Link, error) {
	links := r.linksByChat[chatID]
	result := make([]domain.Link, len(links))
	copy(result, links)
	return result, nil
}

func (r *fakeLinkRepository) ListByTag(ctx context.Context, chatID int64, tag string) ([]domain.Link, error) {
	links := r.linksByChat[chatID]
	var result []domain.Link
	for _, link := range links {
		for _, currentTag := range link.Tags {
			if currentTag == tag {
				result = append(result, link)
				break
			}
		}
	}
	return result, nil
}

type fakeTagRepository struct {
	tags   map[int64]map[string]domain.Tag
	nextID int64
}

func newFakeTagRepository() *fakeTagRepository {
	return &fakeTagRepository{
		tags:   make(map[int64]map[string]domain.Tag),
		nextID: 1,
	}
}

func (r *fakeTagRepository) Create(ctx context.Context, chatID int64, name string) error {
	if r.tags[chatID] == nil {
		r.tags[chatID] = make(map[string]domain.Tag)
	}

	if _, exists := r.tags[chatID][name]; exists {
		return pkg.ErrTagExists
	}

	r.tags[chatID][name] = domain.Tag{
		ID:     r.nextID,
		ChatID: chatID,
		Name:   name,
	}
	r.nextID++

	return nil
}

func (r *fakeTagRepository) List(ctx context.Context, chatID int64) ([]domain.Tag, error) {
	chatTags := r.tags[chatID]

	result := make([]domain.Tag, 0, len(chatTags))
	for _, tag := range chatTags {
		result = append(result, tag)
	}

	return result, nil
}

func (r *fakeTagRepository) Rename(ctx context.Context, chatID int64, oldName, newName string) error {
	chatTags := r.tags[chatID]

	tag, exists := chatTags[oldName]
	if !exists {
		return pkg.ErrTagNotFound
	}

	if _, exists := chatTags[newName]; exists {
		return pkg.ErrTagExists
	}

	delete(chatTags, oldName)

	tag.Name = newName
	chatTags[newName] = tag

	return nil
}

func (r *fakeTagRepository) Delete(ctx context.Context, chatID int64, name string) error {
	chatTags := r.tags[chatID]

	if _, exists := chatTags[name]; !exists {
		return pkg.ErrTagNotFound
	}

	delete(chatTags, name)

	return nil
}

func setupScrapperRouter() http.Handler {
	chatRepo := newFakeChatRepository()
	linkRepo := newFakeLinkRepository()
	tagRepo := newFakeTagRepository()

	chatService := services.NewChatService(chatRepo)
	linkService := services.NewLinkService(linkRepo, chatRepo)
	tagService := services.NewTagService(tagRepo, chatRepo)

	chatHandler := handlers.NewChatHandler(chatService)
	linkHandler := handlers.NewLinkHandler(linkService)
	tagHandler := handlers.NewTagHandler(tagService)

	return httpserver.NewRouter(chatHandler, linkHandler, tagHandler)
}

const tgChatIDHeader = "Tg-Chat-Id"

func withChatID(req *http.Request, chatID int64) *http.Request {
	req.Header.Set(tgChatIDHeader, strconv.FormatInt(chatID, 10))
	return req
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
		"url":  "https://github.com/golang/go",
		"tags": []string{"test"},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/links", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	withChatID(req, 1)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for add link, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/list", nil)
	withChatID(req, 1)
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
		"url":  "https://github.com/golang/go",
		"tags": []string{"test"},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/links", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	withChatID(req, 1)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for add link, got %d", w.Code)
	}

	deleteBody := map[string]any{
		"url": "https://github.com/golang/go",
	}
	deleteData, err := json.Marshal(deleteBody)
	if err != nil {
		t.Fatalf("failed to marshal delete body: %v", err)
	}

	req = httptest.NewRequest(http.MethodDelete, "/links", bytes.NewBuffer(deleteData))
	req.Header.Set("Content-Type", "application/json")
	withChatID(req, 1)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for delete link, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/list", nil)
	withChatID(req, 1)
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
		"url":  "https://github.com/golang/go",
		"tags": []string{"test"},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/links", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	withChatID(req, 1)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for add link, got %d", w.Code)
	}

	deleteBody := map[string]any{
		"url": "https://github.com/golang/go",
	}
	deleteData, err := json.Marshal(deleteBody)
	if err != nil {
		t.Fatalf("failed to marshal delete body: %v", err)
	}

	req = httptest.NewRequest(http.MethodDelete, "/links", bytes.NewBuffer(deleteData))
	req.Header.Set("Content-Type", "application/json")
	withChatID(req, 999)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("expected non-200 when deleting from non-existing chat")
	}

	req = httptest.NewRequest(http.MethodGet, "/list", nil)
	withChatID(req, 1)
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
		"url":  "https://github.com/golang/go",
		"tags":   []string{"test"},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/links", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	withChatID(req, 2)
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
		"url":  "https://github.com/golang/go",
		"tags": []string{"test"},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/links", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	withChatID(req, 1)
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
