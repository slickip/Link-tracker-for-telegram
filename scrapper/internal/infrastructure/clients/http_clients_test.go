package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

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
		return nil, fmt.Errorf("unexpected host: %s", req.URL.Host)
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

func TestGitHubClient_ErrorOnNon2xx(t *testing.T) {
	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer githubSrv.Close()

	stackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not used", http.StatusNotFound)
	}))
	defer stackSrv.Close()

	withRewriteTransport(t, githubSrv, stackSrv, func() {
		c := NewGitHubClient(GitHubClientConfig{})

		_, _, err := c.GetNewIssuesAndPullRequests(
			context.Background(),
			"user",
			"repo",
			"https://github.com/user/repo",
			1,
			time.Time{},
		)
		if err == nil {
			t.Fatalf("expected error on non-2xx")
		}
	})
}

func TestGitHubClient_ErrorOnInvalidJSON(t *testing.T) {
	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer githubSrv.Close()

	stackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not used", http.StatusNotFound)
	}))
	defer stackSrv.Close()

	withRewriteTransport(t, githubSrv, stackSrv, func() {
		c := NewGitHubClient(GitHubClientConfig{})

		_, _, err := c.GetNewIssuesAndPullRequests(
			context.Background(),
			"user",
			"repo",
			"https://github.com/user/repo",
			1,
			time.Time{},
		)
		if err == nil {
			t.Fatalf("expected error on invalid json")
		}
	})
}

func TestStackOverflowClient_ErrorOnEmptyItems(t *testing.T) {
	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not used", http.StatusNotFound)
	}))
	defer githubSrv.Close()

	stackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	}))
	defer stackSrv.Close()

	withRewriteTransport(t, githubSrv, stackSrv, func() {
		c := NewStackOverflowClient(StackOverflowClientConfig{})

		_, _, err := c.GetNewAnswersAndComments(
			context.Background(),
			123,
			"https://stackoverflow.com/questions/123",
			1,
			time.Time{},
		)
		if err == nil {
			t.Fatalf("expected error on empty items")
		}
	})
}

func TestStackOverflowClient_ErrorOnInvalidJSON(t *testing.T) {
	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not used", http.StatusNotFound)
	}))
	defer githubSrv.Close()

	stackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, "bad json")
	}))
	defer stackSrv.Close()

	withRewriteTransport(t, githubSrv, stackSrv, func() {
		c := NewStackOverflowClient(StackOverflowClientConfig{})

		_, _, err := c.GetNewAnswersAndComments(
			context.Background(),
			123,
			"https://stackoverflow.com/questions/123",
			1,
			time.Time{},
		)
		if err == nil {
			t.Fatalf("expected error on invalid json")
		}
	})
}

func TestStackOverflowClient_ErrorOnNon2xx(t *testing.T) {
	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not used", http.StatusNotFound)
	}))
	defer githubSrv.Close()

	stackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer stackSrv.Close()

	withRewriteTransport(t, githubSrv, stackSrv, func() {
		c := NewStackOverflowClient(StackOverflowClientConfig{})

		_, _, err := c.GetNewAnswersAndComments(
			context.Background(),
			123,
			"https://stackoverflow.com/questions/123",
			1,
			time.Time{},
		)
		if err == nil {
			t.Fatalf("expected error on non-2xx")
		}
	})
}

var _ = time.RFC3339
