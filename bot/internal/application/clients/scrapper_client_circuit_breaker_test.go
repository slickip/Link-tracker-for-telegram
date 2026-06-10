package clients

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"

	h "github.com/slickip/link-tracker/pkg/helpers"
)

const (
	cbClientTimeout = time.Second
	cbWaitDuration  = 100 * time.Millisecond
	cbFastCallLimit = 50 * time.Millisecond
)

func TestBotScrapperCircuitBreakerTransitionsToOpen(t *testing.T) {
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewScrapperClient(
		server.URL,
		cbClientTimeout,
		h.HTTPRetryConfig{
			MaxAttempts:           1,
			Delay:                 time.Millisecond,
			RetryableHTTPStatuses: []int{http.StatusInternalServerError},
		},
		testCircuitBreakerConfig(),
	)

	_ = client.RegisterChat(context.Background(), 123)
	_ = client.RegisterChat(context.Background(), 123)

	if client.circuitBreaker.State() != gobreaker.StateOpen {
		t.Fatalf("expected circuit breaker OPEN, got %s", client.circuitBreaker.State())
	}

	before := requests.Load()

	start := time.Now()
	err := client.RegisterChat(context.Background(), 123)
	elapsed := time.Since(start)

	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Fatalf("expected ErrOpenState, got %v", err)
	}

	if elapsed > cbFastCallLimit {
		t.Fatalf("expected immediate failure, elapsed=%s", elapsed)
	}

	if got := requests.Load(); got != before {
		t.Fatalf("expected no external request in OPEN state; before=%d, after=%d", before, got)
	}
}

func TestBotScrapperCircuitBreakerHalfOpenToClosed(t *testing.T) {
	var status atomic.Int32
	status.Store(http.StatusInternalServerError)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(status.Load()))
	}))
	defer server.Close()

	client := NewScrapperClient(
		server.URL,
		cbClientTimeout,
		h.HTTPRetryConfig{
			MaxAttempts:           1,
			Delay:                 time.Millisecond,
			RetryableHTTPStatuses: []int{http.StatusInternalServerError},
		},
		testCircuitBreakerConfig(),
	)

	_ = client.RegisterChat(context.Background(), 123)
	_ = client.RegisterChat(context.Background(), 123)

	if client.circuitBreaker.State() != gobreaker.StateOpen {
		t.Fatalf("expected circuit breaker OPEN, got %s", client.circuitBreaker.State())
	}

	status.Store(http.StatusOK)
	time.Sleep(cbWaitDuration + 30*time.Millisecond)

	for i := 0; i < 2; i++ {
		if err := client.RegisterChat(context.Background(), 123); err != nil {
			t.Fatalf("expected successful half-open probe, got %v", err)
		}
	}

	if client.circuitBreaker.State() != gobreaker.StateClosed {
		t.Fatalf("expected circuit breaker CLOSED, got %s", client.circuitBreaker.State())
	}
}

func TestBotScrapperCircuitBreakerHalfOpenToOpen(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewScrapperClient(
		server.URL,
		cbClientTimeout,
		h.HTTPRetryConfig{
			MaxAttempts:           1,
			Delay:                 time.Millisecond,
			RetryableHTTPStatuses: []int{http.StatusInternalServerError},
		},
		testCircuitBreakerConfig(),
	)

	_ = client.RegisterChat(context.Background(), 123)
	_ = client.RegisterChat(context.Background(), 123)

	if client.circuitBreaker.State() != gobreaker.StateOpen {
		t.Fatalf("expected circuit breaker OPEN, got %s", client.circuitBreaker.State())
	}

	time.Sleep(cbWaitDuration + 30*time.Millisecond)

	for i := 0; i < 2; i++ {
		_ = client.RegisterChat(context.Background(), 123)
	}

	if client.circuitBreaker.State() != gobreaker.StateOpen {
		t.Fatalf("expected circuit breaker OPEN, got %s", client.circuitBreaker.State())
	}
}

func testCircuitBreakerConfig() h.CircuitBreakerConfig {
	return h.CircuitBreakerConfig{
		Enabled:                       true,
		FailureRateThreshold:          50,
		MinimumRequests:               2,
		SlidingWindowInterval:         time.Second,
		SlidingWindowBucketPeriod:     100 * time.Millisecond,
		WaitDurationInOpenState:       cbWaitDuration,
		PermittedCallsInHalfOpenState: 2,
	}
}
