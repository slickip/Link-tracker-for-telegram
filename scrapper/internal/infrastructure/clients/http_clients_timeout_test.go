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

	"github.com/slickip/link-tracker/pkg/api"
)

const (
	testClientTimeout = 50 * time.Millisecond
	testServerDelay   = 500 * time.Millisecond
)

func TestScrapperHTTPClientsTimeoutWhenExternalServiceIsSlow(t *testing.T) {
	tests := []struct {
		name string
		call func(baseURL string) error
	}{
		{
			name: "GitHub client",
			call: func(baseURL string) error {
				client := NewGitHubClient(GitHubClientConfig{
					BaseURL: baseURL,
					Timeout: testClientTimeout,
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
			name: "StackOverflow client",
			call: func(baseURL string) error {
				client := NewStackOverflowClient(StackOverflowClientConfig{
					BaseURL: baseURL,
					Site:    "stackoverflow",
					Timeout: testClientTimeout,
				})

				_, _, err := client.GetNewAnswersAndComments(
					context.Background(),
					123,
					"https://stackoverflow.com/questions/123",
					1,
					time.Time{},
				)

				return err
			},
		},
		{
			name: "HTTP bot client",
			call: func(baseURL string) error {
				client := NewHTTPBotClient(baseURL, testClientTimeout)

				return client.SendUpdate(context.Background(), api.LinkUpdate{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newSlowHTTPServer()
			defer server.Close()

			start := time.Now()

			err := tt.call(server.URL)

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
