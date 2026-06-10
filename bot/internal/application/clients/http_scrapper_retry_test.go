package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	h "github.com/slickip/link-tracker/pkg/helpers"
)

const (
	retryClientTimeout     = time.Second
	retryBackoffDelay      = 100 * time.Millisecond
	retryIntervalTolerance = 90 * time.Millisecond
)

func TestBotScrapperHTTPClientRetriesOnRetryable5xx(t *testing.T) {
	server := newRetrySequenceServer([]int{
		http.StatusInternalServerError,
		http.StatusInternalServerError,
		http.StatusOK,
	}, "")
	defer server.Close()

	client := NewScrapperClient(
		server.URL(),
		retryClientTimeout,
		h.HTTPRetryConfig{
			MaxAttempts:           3,
			Delay:                 10 * time.Millisecond,
			RetryableHTTPStatuses: []int{http.StatusInternalServerError},
		},
		h.CircuitBreakerConfig{},
	)

	err := client.RegisterChat(context.Background(), 123)
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}

	if got := server.RequestCount(); got != 3 {
		t.Fatalf("expected 3 requests, got %d", got)
	}
}

func TestBotScrapperHTTPClientDoesNotRetryOnNonRetryable4xx(t *testing.T) {
	server := newRetrySequenceServer([]int{
		http.StatusBadRequest,
	}, "")
	defer server.Close()

	client := NewScrapperClient(
		server.URL(),
		retryClientTimeout,
		h.HTTPRetryConfig{
			MaxAttempts:           3,
			Delay:                 10 * time.Millisecond,
			RetryableHTTPStatuses: []int{http.StatusInternalServerError},
		},
		h.CircuitBreakerConfig{},
	)

	err := client.RegisterChat(context.Background(), 123)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if got := server.RequestCount(); got != 1 {
		t.Fatalf("expected 1 request without retry, got %d", got)
	}
}

func TestBotScrapperHTTPClientUsesConstantBackoff(t *testing.T) {
	server := newRetrySequenceServer([]int{
		http.StatusInternalServerError,
		http.StatusInternalServerError,
		http.StatusOK,
	}, "")
	defer server.Close()

	client := NewScrapperClient(
		server.URL(),
		retryClientTimeout,
		h.HTTPRetryConfig{
			MaxAttempts:           3,
			Delay:                 retryBackoffDelay,
			RetryableHTTPStatuses: []int{http.StatusInternalServerError},
		},
		h.CircuitBreakerConfig{},
	)

	err := client.RegisterChat(context.Background(), 123)
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
