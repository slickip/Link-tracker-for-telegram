package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type BotClient interface {
	SendUpdate(ctx context.Context, update api.LinkUpdate) error
}

type HTTPBotClient struct {
	baseURL string
	client  *http.Client
	timeout time.Duration
}

func NewHTTPBotClient(baseURL string, timeout time.Duration) *HTTPBotClient {
	timeout = normalizeTimeout(timeout)

	return &HTTPBotClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

func (c *HTTPBotClient) SendUpdate(ctx context.Context, update api.LinkUpdate) error {
	body, err := json.Marshal(update)
	if err != nil {
		return err
	}

	req, cancel, err := newRequestWithTimeout(
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
		return fmt.Errorf("bot returned status %d", resp.StatusCode)
	}

	return nil
}
