package services

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/clients"
)

type mockBotClient struct {
	calls []api.LinkUpdate
}

func (m *mockBotClient) SendUpdate(ctx context.Context, update api.LinkUpdate) error {
	m.calls = append(m.calls, update)
	return nil
}

type fakeTrackingRepository struct {
	linksByID       map[int64]domain.Link
	subscribersByID map[int64]map[int64]struct{}
}

func newFakeTrackingRepository() *fakeTrackingRepository {
	return &fakeTrackingRepository{
		linksByID:       make(map[int64]domain.Link),
		subscribersByID: make(map[int64]map[int64]struct{}),
	}
}

func (r *fakeTrackingRepository) Add(chatID int64, link domain.Link) error {
	r.linksByID[link.ID] = link
	if _, ok := r.subscribersByID[link.ID]; !ok {
		r.subscribersByID[link.ID] = make(map[int64]struct{})
	}
	r.subscribersByID[link.ID][chatID] = struct{}{}
	return nil
}

func (r *fakeTrackingRepository) FindSubscribers(ctx context.Context, linkID int64) ([]int64, error) {
	subs := r.subscribersByID[linkID]
	result := make([]int64, 0, len(subs))
	for chatID := range subs {
		result = append(result, chatID)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func (r *fakeTrackingRepository) GetTrackedLinksBatch(ctx context.Context, limit, offset int) ([]domain.Link, error) {
	all := make([]domain.Link, 0, len(r.linksByID))
	for _, link := range r.linksByID {
		all = append(all, link)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })

	if offset >= len(all) {
		return nil, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	result := make([]domain.Link, 0, end-offset)
	result = append(result, all[offset:end]...)
	return result, nil
}

func (r *fakeTrackingRepository) UpdateLastUpdated(ctx context.Context, linkID int64, t time.Time) error {
	link := r.linksByID[linkID]
	link.LastUpdatedAt = t
	r.linksByID[linkID] = link
	return nil
}

func TestScheduler_ProcessLink_SendsUpdateOnlyToSubscribers(t *testing.T) {
	oldUpdatedAt := time.Now().Add(-2 * time.Hour)
	newUpdatedAt := time.Now().Add(2 * time.Hour)
	linkURL := "https://github.com/user/repo"
	linkID := int64(10)

	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"title":"t","body":"b","created_at":"` + newUpdatedAt.Format(time.RFC3339) + `","html_url":"` + linkURL + `","user":{"login":"u"}}]`))
	}))
	defer githubSrv.Close()

	stackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not used in this test", http.StatusNotFound)
	}))
	defer stackSrv.Close()

	repo := newFakeTrackingRepository()
	if err := repo.Add(1, domain.Link{ID: linkID, URL: linkURL, Tags: []string{"go"}, LastUpdatedAt: oldUpdatedAt}); err != nil {
		t.Fatalf("failed to add link: %v", err)
	}

	if err := repo.Add(2, domain.Link{ID: 11, URL: "https://github.com/other/repo2", Tags: []string{"go"}, LastUpdatedAt: oldUpdatedAt}); err != nil {
		t.Fatalf("failed to add link: %v", err)
	}

	botClient := &mockBotClient{}
	log := logger.New(slog.LevelInfo)

	s := NewScheduler(
		repo,
		clients.NewGitHubClient(clients.GitHubClientConfig{BaseURL: githubSrv.URL}),
		clients.NewStackOverflowClient(clients.StackOverflowClientConfig{BaseURL: stackSrv.URL}),
		botClient,
		log,
		30*time.Second,
		100,
		1,
	)

	if err := s.processLink(context.Background(), domain.Link{ID: linkID, URL: linkURL, LastUpdatedAt: oldUpdatedAt}); err != nil {
		t.Fatalf("unexpected error from processLink: %v", err)
	}

	if len(botClient.calls) != 1 {
		t.Fatalf("expected exactly 1 bot update, got %d", len(botClient.calls))
	}

	gotIDs := append([]int64(nil), botClient.calls[0].TgChatIDs...)
	sort.Slice(gotIDs, func(i, j int) bool { return gotIDs[i] < gotIDs[j] })

	if gotIDs[0] != 1 || len(gotIDs) != 1 {
		t.Fatalf("expected update TgChatIDs=[1], got %#v", botClient.calls[0].TgChatIDs)
	}
	if botClient.calls[0].URL != linkURL {
		t.Fatalf("unexpected update URL: %q", botClient.calls[0].URL)
	}
}

func TestScheduler_CheckLinks_DoesNotPanic_OnExternalAPIError(t *testing.T) {
	oldUpdatedAt := time.Now().Add(-2 * time.Hour)
	linkURL := "https://github.com/user/repo"
	linkID := int64(10)

	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer githubSrv.Close()

	stackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not used in this test", http.StatusNotFound)
	}))
	defer stackSrv.Close()

	repo := newFakeTrackingRepository()
	if err := repo.Add(1, domain.Link{ID: linkID, URL: linkURL, Tags: []string{"go"}, LastUpdatedAt: oldUpdatedAt}); err != nil {
		t.Fatalf("failed to add link: %v", err)
	}

	botClient := &mockBotClient{}
	log := logger.New(slog.LevelInfo)

	s := NewScheduler(
		repo,
		clients.NewGitHubClient(clients.GitHubClientConfig{BaseURL: githubSrv.URL}),
		clients.NewStackOverflowClient(clients.StackOverflowClientConfig{BaseURL: stackSrv.URL}),
		botClient,
		log,
		30*time.Second,
		100,
		1,
	)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected CheckLinks to not panic, got: %v", r)
		}
	}()

	s.CheckLinks()

	if len(botClient.calls) != 1 {
		t.Fatalf("expected 1 bot update (error report) on external API error, got %#v", botClient.calls)
	}
	if botClient.calls[0].Type != "processing_error" {
		t.Fatalf("expected processing_error update type, got %q", botClient.calls[0].Type)
	}
}

func TestScheduler_GitHubIssue_MessageContainsRequiredFields_AndPreviewTrimmed(t *testing.T) {
	oldUpdatedAt := time.Now().Add(-2 * time.Hour)
	newCreatedAt := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)

	linkURL := "https://github.com/user/repo"
	linkID := int64(10)

	longBody := make([]rune, 0, 500)
	for i := 0; i < 500; i++ {
		longBody = append(longBody, 'a')
	}

	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`[{"title":"ISSUE-1","body":"` + string(longBody) + `","created_at":"` + newCreatedAt.Format(time.RFC3339) + `","html_url":"` + linkURL + `","user":{"login":"octocat"}}]`,
		))
	}))
	defer githubSrv.Close()

	stackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not used in this test", http.StatusNotFound)
	}))
	defer stackSrv.Close()

	repo := newFakeTrackingRepository()
	requireAdd := func(err error) {
		if err != nil {
			t.Fatalf("failed to add link: %v", err)
		}
	}
	requireAdd(repo.Add(1, domain.Link{ID: linkID, URL: linkURL, LastUpdatedAt: oldUpdatedAt}))

	botClient := &mockBotClient{}
	log := logger.New(slog.LevelInfo)

	s := NewScheduler(
		repo,
		clients.NewGitHubClient(clients.GitHubClientConfig{BaseURL: githubSrv.URL}),
		clients.NewStackOverflowClient(clients.StackOverflowClientConfig{BaseURL: stackSrv.URL}),
		botClient,
		log,
		30*time.Second,
		100,
		1,
	)

	if err := s.processLink(context.Background(), domain.Link{ID: linkID, URL: linkURL, LastUpdatedAt: oldUpdatedAt}); err != nil {
		t.Fatalf("unexpected error from processLink: %v", err)
	}

	if len(botClient.calls) != 1 {
		t.Fatalf("expected exactly 1 bot update, got %d", len(botClient.calls))
	}

	got := botClient.calls[0]
	if got.Type != string(domain.UpdateTypeGitHubIssue) {
		t.Fatalf("expected type %q, got %q", domain.UpdateTypeGitHubIssue, got.Type)
	}
	if got.Title != "ISSUE-1" {
		t.Fatalf("expected title %q, got %q", "ISSUE-1", got.Title)
	}
	if got.Username != "octocat" {
		t.Fatalf("expected username %q, got %q", "octocat", got.Username)
	}
	if !got.CreatedAt.Equal(newCreatedAt) {
		t.Fatalf("expected createdAt=%s, got %s", newCreatedAt, got.CreatedAt)
	}
	if len([]rune(got.Preview)) != 200 {
		t.Fatalf("expected preview to be trimmed to 200 runes, got %d", len([]rune(got.Preview)))
	}
	if got.Description == "" {
		t.Fatalf("expected non-empty description")
	}
}

func TestScheduler_StackOverflowAnswer_MessageContainsRequiredFields_AndPreviewTrimmed(t *testing.T) {
	oldUpdatedAt := time.Now().Add(-2 * time.Hour)
	newCreatedAt := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)

	linkURL := "https://stackoverflow.com/questions/12345/title"
	linkID := int64(20)

	longBody := make([]rune, 0, 500)
	for i := 0; i < 500; i++ {
		longBody = append(longBody, 'b')
	}

	stackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/questions/12345":
			_, _ = w.Write([]byte(`{"items":[{"title":"SO title","last_activity_date":0}]}`))
		case r.URL.Path == "/questions/12345/answers":
			_, _ = w.Write([]byte(
				`{"items":[{"creation_date":` + fmt.Sprintf("%d", newCreatedAt.Unix()) + `,"body":"` + string(longBody) + `","owner":{"display_name":"so-user"}}]}`,
			))
		case r.URL.Path == "/questions/12345/comments":
			_, _ = w.Write([]byte(`{"items":[]}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer stackSrv.Close()

	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not used in this test", http.StatusNotFound)
	}))
	defer githubSrv.Close()

	repo := newFakeTrackingRepository()
	if err := repo.Add(1, domain.Link{ID: linkID, URL: linkURL, LastUpdatedAt: oldUpdatedAt}); err != nil {
		t.Fatalf("failed to add link: %v", err)
	}

	botClient := &mockBotClient{}
	log := logger.New(slog.LevelInfo)

	s := NewScheduler(
		repo,
		clients.NewGitHubClient(clients.GitHubClientConfig{BaseURL: githubSrv.URL}),
		clients.NewStackOverflowClient(clients.StackOverflowClientConfig{BaseURL: stackSrv.URL, Site: "stackoverflow"}),
		botClient,
		log,
		30*time.Second,
		100,
		1,
	)

	if err := s.processLink(context.Background(), domain.Link{ID: linkID, URL: linkURL, LastUpdatedAt: oldUpdatedAt}); err != nil {
		t.Fatalf("unexpected error from processLink: %v", err)
	}

	if len(botClient.calls) != 1 {
		t.Fatalf("expected exactly 1 bot update, got %d", len(botClient.calls))
	}

	got := botClient.calls[0]
	if got.Type != string(domain.UpdateTypeStackOverflowAnswer) {
		t.Fatalf("expected type %q, got %q", domain.UpdateTypeStackOverflowAnswer, got.Type)
	}
	if got.Title != "SO title" {
		t.Fatalf("expected title %q, got %q", "SO title", got.Title)
	}
	if got.Username != "so-user" {
		t.Fatalf("expected username %q, got %q", "so-user", got.Username)
	}
	if got.CreatedAt.Unix() != newCreatedAt.Unix() {
		t.Fatalf("expected createdAt unix=%d, got %d", newCreatedAt.Unix(), got.CreatedAt.Unix())
	}
	if len([]rune(got.Preview)) != 200 {
		t.Fatalf("expected preview to be trimmed to 200 runes, got %d", len([]rune(got.Preview)))
	}
	if got.Description == "" {
		t.Fatalf("expected non-empty description")
	}
}
