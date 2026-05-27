package clients

import (
	"context"
	"io"
	"net/http"
	"time"
)

const defaultHTTPClientTimeout = 5 * time.Second

func normalizeTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return defaultHTTPClientTimeout
	}

	return timeout
}

func newRequestWithTimeout(
	ctx context.Context,
	timeout time.Duration,
	method string,
	url string,
	body io.Reader,
) (*http.Request, context.CancelFunc, error) {
	timeout = normalizeTimeout(timeout)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)

	req, err := http.NewRequestWithContext(ctxWithTimeout, method, url, body)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	return req, cancel, nil
}
