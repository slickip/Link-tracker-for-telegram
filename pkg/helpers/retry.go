package helpers

import (
	"context"
	"errors"
	"fmt"
	"time"

	retry "github.com/avast/retry-go/v5"
)

const (
	defaultHTTPRetryMaxAttempts uint = 1
	defaultHTTPRetryDelay            = 500 * time.Millisecond
)

type HTTPRetryConfig struct {
	MaxAttempts           uint
	Delay                 time.Duration
	RetryableHTTPStatuses []int
}

type retryableHTTPStatusError struct {
	service    string
	statusCode int
}

func (e retryableHTTPStatusError) Error() string {
	return fmt.Sprintf("%s returned retryable status %d", e.service, e.statusCode)
}

func NormalizeRetryConfig(cfg HTTPRetryConfig) HTTPRetryConfig {
	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = defaultHTTPRetryMaxAttempts
	}

	if cfg.Delay <= 0 {
		cfg.Delay = defaultHTTPRetryDelay
	}

	return cfg
}

func isRetryableStatus(statusCode int, statuses []int) bool {
	for _, status := range statuses {
		if status == statusCode {
			return true
		}
	}

	return false
}

func NewHTTPStatusError(service string, statusCode int, cfg HTTPRetryConfig) error {
	if isRetryableStatus(statusCode, cfg.RetryableHTTPStatuses) {
		return retryableHTTPStatusError{
			service:    service,
			statusCode: statusCode,
		}
	}

	return fmt.Errorf("%s returned status %d", service, statusCode)
}

func DoWithHTTPRetry(
	ctx context.Context,
	cfg HTTPRetryConfig,
	operation func() error,
) error {
	cfg = NormalizeRetryConfig(cfg)

	retrier := retry.New(
		retry.Attempts(cfg.MaxAttempts),
		retry.Delay(cfg.Delay),
		retry.DelayType(retry.FixedDelay),
		retry.Context(ctx),
		retry.LastErrorOnly(true),
		retry.RetryIf(func(err error) bool {
			var statusErr retryableHTTPStatusError
			return errors.As(err, &statusErr)
		}),
	)

	return retrier.Do(operation)
}
