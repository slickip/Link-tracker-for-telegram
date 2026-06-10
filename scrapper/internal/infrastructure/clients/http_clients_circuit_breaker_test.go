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

type circuitBreakerTestHarness struct {
	doRequest    func(ctx context.Context) error
	breakerState func() gobreaker.State
}

func TestScrapperExternalClientsCircuitBreakerTransitionsToOpen(t *testing.T) {
	for _, tt := range externalCircuitBreakerClientCases() {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer server.Close()

			harness := tt.newHarness(server.URL, testExternalCircuitBreakerConfig())

			_ = harness.doRequest(context.Background())
			_ = harness.doRequest(context.Background())

			if harness.breakerState() != gobreaker.StateOpen {
				t.Fatalf("expected circuit breaker OPEN, got %s", harness.breakerState())
			}

			before := requests.Load()

			start := time.Now()
			err := harness.doRequest(context.Background())
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
		})
	}
}

func TestScrapperExternalClientsCircuitBreakerHalfOpenToClosed(t *testing.T) {
	for _, tt := range externalCircuitBreakerClientCases() {
		t.Run(tt.name, func(t *testing.T) {
			var status atomic.Int32
			status.Store(http.StatusInternalServerError)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				code := int(status.Load())
				w.WriteHeader(code)
				if code == http.StatusOK {
					_, _ = w.Write([]byte(tt.successBody))
				}
			}))
			defer server.Close()

			harness := tt.newHarness(server.URL, testExternalCircuitBreakerConfig())

			_ = harness.doRequest(context.Background())
			_ = harness.doRequest(context.Background())

			if harness.breakerState() != gobreaker.StateOpen {
				t.Fatalf("expected circuit breaker OPEN, got %s", harness.breakerState())
			}

			status.Store(http.StatusOK)
			time.Sleep(cbWaitDuration + 30*time.Millisecond)

			for i := 0; i < 2; i++ {
				if err := harness.doRequest(context.Background()); err != nil {
					t.Fatalf("expected successful half-open probe, got %v", err)
				}
			}

			if harness.breakerState() != gobreaker.StateClosed {
				t.Fatalf("expected circuit breaker CLOSED, got %s", harness.breakerState())
			}
		})
	}
}

func TestScrapperExternalClientsCircuitBreakerHalfOpenToOpen(t *testing.T) {
	for _, tt := range externalCircuitBreakerClientCases() {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer server.Close()

			harness := tt.newHarness(server.URL, testExternalCircuitBreakerConfig())

			_ = harness.doRequest(context.Background())
			_ = harness.doRequest(context.Background())

			if harness.breakerState() != gobreaker.StateOpen {
				t.Fatalf("expected circuit breaker OPEN, got %s", harness.breakerState())
			}

			time.Sleep(cbWaitDuration + 30*time.Millisecond)

			for i := 0; i < 2; i++ {
				_ = harness.doRequest(context.Background())
			}

			if harness.breakerState() != gobreaker.StateOpen {
				t.Fatalf("expected circuit breaker OPEN, got %s", harness.breakerState())
			}
		})
	}
}

type externalCircuitBreakerClientCase struct {
	name        string
	successBody string
	newHarness  func(serverURL string, cfg h.CircuitBreakerConfig) circuitBreakerTestHarness
}

func externalCircuitBreakerClientCases() []externalCircuitBreakerClientCase {
	retryConfig := h.HTTPRetryConfig{
		MaxAttempts:           1,
		Delay:                 time.Millisecond,
		RetryableHTTPStatuses: []int{http.StatusInternalServerError},
	}

	return []externalCircuitBreakerClientCase{
		{
			name:        "GitHub client",
			successBody: `[]`,
			newHarness: func(serverURL string, cfg h.CircuitBreakerConfig) circuitBreakerTestHarness {
				client := NewGitHubClient(GitHubClientConfig{
					BaseURL:        serverURL,
					Timeout:        cbClientTimeout,
					Retry:          retryConfig,
					CircuitBreaker: cfg,
				})

				return circuitBreakerTestHarness{
					doRequest: func(ctx context.Context) error {
						_, _, err := client.GetNewIssuesAndPullRequests(
							ctx,
							"owner",
							"repo",
							"https://github.com/owner/repo",
							1,
							time.Time{},
						)

						return err
					},
					breakerState: func() gobreaker.State {
						return client.circuitBreaker.State()
					},
				}
			},
		},
		{
			name:        "StackOverflow client",
			successBody: `{}`,
			newHarness: func(serverURL string, cfg h.CircuitBreakerConfig) circuitBreakerTestHarness {
				client := NewStackOverflowClient(StackOverflowClientConfig{
					BaseURL:        serverURL,
					Site:           "stackoverflow",
					Timeout:        cbClientTimeout,
					Retry:          retryConfig,
					CircuitBreaker: cfg,
				})

				return circuitBreakerTestHarness{
					doRequest: func(ctx context.Context) error {
						var result map[string]any

						return client.getJSON(ctx, serverURL, &result)
					},
					breakerState: func() gobreaker.State {
						return client.circuitBreaker.State()
					},
				}
			},
		},
	}
}

func testExternalCircuitBreakerConfig() h.CircuitBreakerConfig {
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
