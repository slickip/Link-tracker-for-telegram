package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type BotClient interface {
	SendUpdate(ctx context.Context, update api.LinkUpdate) error
}

type HTTPBotClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPBotClient(baseURL string) *HTTPBotClient {
	return &HTTPBotClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (c *HTTPBotClient) SendUpdate(ctx context.Context, update api.LinkUpdate) error {
	body, err := json.Marshal(update)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/updates",
		bytes.NewBuffer(body),
	)

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bot returned status %d", resp.StatusCode)
	}

	return nil
}
