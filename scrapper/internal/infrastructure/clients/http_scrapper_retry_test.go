package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	h "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/helpers"
)

const (
	retryClientTimeout     = time.Second
	retryBackoffDelay      = 100 * time.Millisecond
	retryIntervalTolerance = 90 * time.Millisecond
)

func TestScrapperHTTPClientsRetryOnRetryable5xx(t *testing.T) {
	for _, tt := range scrapperRetryClientCases() {
		t.Run(tt.name, func(t *testing.T) {
			server := newRetrySequenceServer([]int{
				http.StatusInternalServerError,
				http.StatusInternalServerError,
				http.StatusOK,
			}, tt.successBody)
			defer server.Close()

			err := tt.call(server.URL(), h.HTTPRetryConfig{
				MaxAttempts:           3,
				Delay:                 10 * time.Millisecond,
				RetryableHTTPStatuses: []int{http.StatusInternalServerError},
			})

			if err != nil {
				t.Fatalf("expected success after retry, got error: %v", err)
			}

			if got := server.RequestCount(); got != 3 {
				t.Fatalf("expected 3 requests, got %d", got)
			}
		})
	}
}

func TestScrapperHTTPClientsDoNotRetryOnNonRetryable4xx(t *testing.T) {
	for _, tt := range scrapperRetryClientCases() {
		t.Run(tt.name, func(t *testing.T) {
			server := newRetrySequenceServer([]int{
				http.StatusBadRequest,
			}, tt.successBody)
			defer server.Close()

			err := tt.call(server.URL(), h.HTTPRetryConfig{
				MaxAttempts:           3,
				Delay:                 10 * time.Millisecond,
				RetryableHTTPStatuses: []int{http.StatusInternalServerError},
			})

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if got := server.RequestCount(); got != 1 {
				t.Fatalf("expected 1 request without retry, got %d", got)
			}
		})
	}
}

func TestScrapperHTTPClientsUseConstantBackoff(t *testing.T) {
	for _, tt := range scrapperRetryClientCases() {
		t.Run(tt.name, func(t *testing.T) {
			server := newRetrySequenceServer([]int{
				http.StatusInternalServerError,
				http.StatusInternalServerError,
				http.StatusOK,
			}, tt.successBody)
			defer server.Close()

			err := tt.call(server.URL(), h.HTTPRetryConfig{
				MaxAttempts:           3,
				Delay:                 retryBackoffDelay,
				RetryableHTTPStatuses: []int{http.StatusInternalServerError},
			})

			if err != nil {
				t.Fatalf("expected success after retry, got error: %v", err)
			}

			times := server.RequestTimes()
			if len(times) != 3 {
				t.Fatalf("expected 3 requests, got %d", len(times))
			}

			firstInterval := times[1].Sub(times[0])
			secondInterval := times[2].Sub(times[1])

			assertConstantBackoff(t, firstInterval, secondInterval, retryBackoffDelay)
		})
	}
}

type scrapperRetryClientCase struct {
	name        string
	successBody string
	call        func(baseURL string, retryConfig h.HTTPRetryConfig) error
}

func scrapperRetryClientCases() []scrapperRetryClientCase {
	return []scrapperRetryClientCase{
		{
			name:        "GitHub client",
			successBody: `[]`,
			call: func(baseURL string, retryConfig h.HTTPRetryConfig) error {
				client := NewGitHubClient(GitHubClientConfig{
					BaseURL:        baseURL,
					Timeout:        retryClientTimeout,
					Retry:          retryConfig,
					CircuitBreaker: h.CircuitBreakerConfig{},
				})

				_, _, err := client.GetNewIssuesAndPullRequests(
					context.Background(),
					"owner",
					"repo",
					"https://github.com/owner/repo",
					1,
					time.Time{},
				)

				return err
			},
		},
		{
			name:        "StackOverflow client",
			successBody: `{"items":[]}`,
			call: func(baseURL string, retryConfig h.HTTPRetryConfig) error {
				client := NewStackOverflowClient(StackOverflowClientConfig{
					BaseURL:        baseURL,
					Site:           "stackoverflow",
					Timeout:        retryClientTimeout,
					Retry:          retryConfig,
					CircuitBreaker: h.CircuitBreakerConfig{},
				})

				var result map[string]any

				return client.getJSON(context.Background(), baseURL, &result)
			},
		},
		{
			name:        "HTTP bot client",
			successBody: ``,
			call: func(baseURL string, retryConfig h.HTTPRetryConfig) error {
				client := NewHTTPBotClient(
					baseURL,
					retryClientTimeout,
					retryConfig,
				)

				return client.SendUpdate(context.Background(), api.LinkUpdate{})
			},
		},
	}
}

type retrySequenceServer struct {
	server   *httptest.Server
	mu       sync.Mutex
	statuses []int
	body     string
	requests int
	times    []time.Time
}

func newRetrySequenceServer(statuses []int, body string) *retrySequenceServer {
	s := &retrySequenceServer{
		statuses: statuses,
		body:     body,
		times:    make([]time.Time, 0, len(statuses)),
	}

	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()

		s.requests++
		attempt := s.requests
		s.times = append(s.times, time.Now())

		status := s.statuses[len(s.statuses)-1]
		if attempt <= len(s.statuses) {
			status = s.statuses[attempt-1]
		}

		s.mu.Unlock()

		w.WriteHeader(status)

		if s.body != "" {
			_, _ = w.Write([]byte(s.body))
		}
	}))

	return s
}

func (s *retrySequenceServer) URL() string {
	return s.server.URL
}

func (s *retrySequenceServer) Close() {
	s.server.Close()
}

func (s *retrySequenceServer) RequestCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.requests
}

func (s *retrySequenceServer) RequestTimes() []time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]time.Time, len(s.times))
	copy(result, s.times)

	return result
}

func assertConstantBackoff(
	t *testing.T,
	firstInterval time.Duration,
	secondInterval time.Duration,
	expectedDelay time.Duration,
) {
	t.Helper()

	if firstInterval < expectedDelay {
		t.Fatalf("expected first retry interval >= %s, got %s", expectedDelay, firstInterval)
	}

	if secondInterval < expectedDelay {
		t.Fatalf("expected second retry interval >= %s, got %s", expectedDelay, secondInterval)
	}

	diff := absDuration(firstInterval - secondInterval)
	if diff > retryIntervalTolerance {
		t.Fatalf(
			"expected constant backoff; first interval=%s, second interval=%s, diff=%s",
			firstInterval,
			secondInterval,
			diff,
		)
	}
}

func absDuration(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}

	return value
}
