package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	h "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/helpers"
)

type BotClient interface {
	SendUpdate(ctx context.Context, update api.LinkUpdate) error
}

type HTTPBotClient struct {
	baseURL     string
	client      *http.Client
	timeout     time.Duration
	retryConfig h.HTTPRetryConfig
}

func NewHTTPBotClient(baseURL string, timeout time.Duration, retryConfigs ...h.HTTPRetryConfig) *HTTPBotClient {
	timeout = h.NormalizeTimeout(timeout)

	var retryConfig h.HTTPRetryConfig
	if len(retryConfigs) > 0 {
		retryConfig = retryConfigs[0]
	}

	return &HTTPBotClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
		timeout:     timeout,
		retryConfig: h.NormalizeRetryConfig(retryConfig),
	}
}

func (c *HTTPBotClient) SendUpdate(ctx context.Context, update api.LinkUpdate) error {
	body, err := json.Marshal(update)
	if err != nil {
		return err
	}

	return h.DoWithHTTPRetry(ctx, c.retryConfig, func() error {
		req, cancel, err := h.NewRequestWithTimeout(
			ctx,
			c.timeout,
			http.MethodPost,
			c.baseURL+"/updates",
			bytes.NewBuffer(body),
		)
		if err != nil {
			return err
		}
		defer cancel()

		req.Header.Set("Content-Type", "application/json")

		resp, err := c.client.Do(req)
		if err != nil {
			return fmt.Errorf("bot http request failed: %w", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			return h.NewHTTPStatusError("bot", resp.StatusCode, c.retryConfig)
		}

		return nil
	})
}
