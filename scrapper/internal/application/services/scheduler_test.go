package services

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/clients"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/repositories"
)

type mockBotClient struct {
	calls []domain.LinkUpdate
}

func (m *mockBotClient) SendUpdate(ctx context.Context, update domain.LinkUpdate) error {
	m.calls = append(m.calls, update)
	return nil
}

type rewriteTransport struct {
	old        http.RoundTripper
	githubBase *url.URL
	stackBase  *url.URL
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var base *url.URL
	switch req.URL.Host {
	case "api.github.com":
		base = t.githubBase
	case "api.stackexchange.com":
		base = t.stackBase
	default:
		return nil, &url.Error{Op: req.Method, URL: req.URL.String(), Err: http.ErrNotSupported}
	}

	target := *base
	target.Path = req.URL.Path
	target.RawQuery = req.URL.RawQuery

	targetReq, err := http.NewRequestWithContext(req.Context(), req.Method, target.String(), nil)
	if err != nil {
		return nil, err
	}
	targetReq.Header = req.Header.Clone()
	return t.old.RoundTrip(targetReq)
}

func withRewriteTransport(t *testing.T, githubSrv, stackSrv *httptest.Server, fn func()) {
	t.Helper()

	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })

	githubURL, err := url.Parse(githubSrv.URL)
	if err != nil {
		t.Fatalf("failed to parse github server url: %v", err)
	}
	stackURL, err := url.Parse(stackSrv.URL)
	if err != nil {
		t.Fatalf("failed to parse stack server url: %v", err)
	}

	http.DefaultTransport = &rewriteTransport{
		old:        old,
		githubBase: githubURL,
		stackBase:  stackURL,
	}

	fn()
}

func TestScheduler_ProcessLink_SendsUpdateOnlyToSubscribers(t *testing.T) {
	oldUpdatedAt := time.Now().Add(-2 * time.Hour)
	newUpdatedAt := time.Now().Add(2 * time.Hour)
	linkURL := "https://github.com/user/repo"

	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"updated_at":"` + newUpdatedAt.Format(time.RFC3339) + `"}`))
	}))
	defer githubSrv.Close()

	stackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not used in this test", http.StatusNotFound)
	}))
	defer stackSrv.Close()

	repo := repositories.NewInMemoryLinkRepository()
	if err := repo.Add(1, domain.Link{URL: linkURL, Tags: []string{"go"}, LastUpdatedAt: oldUpdatedAt}); err != nil {
		t.Fatalf("failed to add link: %v", err)
	}

	if err := repo.Add(2, domain.Link{URL: "https://github.com/other/repo2", Tags: []string{"go"}, LastUpdatedAt: oldUpdatedAt}); err != nil {
		t.Fatalf("failed to add link: %v", err)
	}

	botClient := &mockBotClient{}
	log := logger.New(slog.LevelInfo)

	withRewriteTransport(t, githubSrv, stackSrv, func() {
		s := NewScheduler(
			repo,
			clients.NewGitHubClient(),
			clients.NewStackOverflowClient(),
			botClient,
			log,
		)

		if err := s.processLink(domain.Link{URL: linkURL, LastUpdatedAt: oldUpdatedAt}); err != nil {
			t.Fatalf("unexpected error from processLink: %v", err)
		}
	})

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

	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer githubSrv.Close()

	stackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not used in this test", http.StatusNotFound)
	}))
	defer stackSrv.Close()

	repo := repositories.NewInMemoryLinkRepository()
	if err := repo.Add(1, domain.Link{URL: linkURL, Tags: []string{"go"}, LastUpdatedAt: oldUpdatedAt}); err != nil {
		t.Fatalf("failed to add link: %v", err)
	}

	botClient := &mockBotClient{}
	log := logger.New(slog.LevelInfo)

	withRewriteTransport(t, githubSrv, stackSrv, func() {
		s := NewScheduler(
			repo,
			clients.NewGitHubClient(),
			clients.NewStackOverflowClient(),
			botClient,
			log,
		)

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected CheckLinks to not panic, got: %v", r)
			}
		}()

		s.CheckLinks()
	})

	if len(botClient.calls) != 0 {
		t.Fatalf("expected no bot updates on external API error, got %#v", botClient.calls)
	}
}
