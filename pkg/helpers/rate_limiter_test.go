package helpers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitMiddlewareExceededLimitReturns429(t *testing.T) {
	cfg := RateLimiterConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		Burst:             2,
	}

	successCalls := 0

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		successCalls++
		w.WriteHeader(http.StatusOK)
	})

	handler := RateLimitMiddleware(cfg)(next)

	statuses := make([]int, 0, 3)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.0.2.1:12345"

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		statuses = append(statuses, rec.Code)
	}

	if statuses[0] != http.StatusOK {
		t.Fatalf("expected first request status 200, got %d", statuses[0])
	}

	if statuses[1] != http.StatusOK {
		t.Fatalf("expected second request status 200, got %d", statuses[1])
	}

	if statuses[2] != http.StatusTooManyRequests {
		t.Fatalf("expected third request status 429, got %d", statuses[2])
	}

	if successCalls != 2 {
		t.Fatalf("expected handler to be called 2 times, got %d", successCalls)
	}
}
