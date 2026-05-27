package clients

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

const (
	testClientTimeout = 50 * time.Millisecond
	testServerDelay   = 500 * time.Millisecond
)

func TestBotScrapperHTTPClientTimeoutWhenExternalServiceIsSlow(t *testing.T) {
	tests := []struct {
		name string
		call func(client ScrapperClient) error
	}{
		{
			name: "RegisterChat",
			call: func(client ScrapperClient) error {
				return client.RegisterChat(context.Background(), 123)
			},
		},
		{
			name: "DeleteChat",
			call: func(client ScrapperClient) error {
				return client.DeleteChat(context.Background(), 123)
			},
		},
		{
			name: "AddLink",
			call: func(client ScrapperClient) error {
				return client.AddLink(
					context.Background(),
					123,
					"https://github.com/owner/repo",
					[]string{"go"},
				)
			},
		},
		{
			name: "RemoveLink",
			call: func(client ScrapperClient) error {
				return client.RemoveLink(
					context.Background(),
					123,
					"https://github.com/owner/repo",
				)
			},
		},
		{
			name: "ListLinks",
			call: func(client ScrapperClient) error {
				_, err := client.ListLinks(context.Background(), 123)
				return err
			},
		},
		{
			name: "RemoveLinksByTag",
			call: func(client ScrapperClient) error {
				_, err := client.RemoveLinksByTag(context.Background(), 123, "l")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newSlowHTTPServer()
			defer server.Close()

			client := NewScrapperClient(server.URL, testClientTimeout)

			start := time.Now()

			err := tt.call(client)

			elapsed := time.Since(start)

			assertTimeoutAndFast(t, err, elapsed)
		})
	}
}

func newSlowHTTPServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timer := time.NewTimer(testServerDelay)
		defer timer.Stop()

		select {
		case <-timer.C:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		case <-r.Context().Done():
			return
		}
	}))
}

func assertTimeoutAndFast(t *testing.T, err error, elapsed time.Duration) {
	t.Helper()

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !isTimeoutError(err) {
		t.Fatalf("expected timeout error, got: %v", err)
	}

	if elapsed >= testServerDelay {
		t.Fatalf(
			"expected request to finish before external service delay; elapsed=%s, serverDelay=%s",
			elapsed,
			testServerDelay,
		)
	}
}

func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	if os.IsTimeout(err) {
		return true
	}

	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
