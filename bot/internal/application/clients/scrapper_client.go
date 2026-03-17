package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
)

type ScrapperClient interface {
	RegisterChat(ctx context.Context, chatID int64) error
	DeleteChat(ctx context.Context, chatID int64) error
	AddLink(ctx context.Context, chatID int64, url string, tags []string) error
	RemoveLink(ctx context.Context, chatID int64, url string) error
	ListLinks(ctx context.Context, chatID int64) ([]domain.Link, error)
}

type HTTPscrapperClient struct {
	baseURL string
	client  *http.Client
}

func NewScrapperClient(baseURL string) *HTTPscrapperClient {
	return &HTTPscrapperClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

type addLinkRequest struct {
	ChatID int64    `json:"chatId"`
	URL    string   `json:"url"`
	Tags   []string `json:"tags"`
}

func (c *HTTPscrapperClient) RegisterChat(ctx context.Context, chatID int64) error {

	url := fmt.Sprintf("%s/tg-chat/%d", c.baseURL, chatID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("scrapper returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *HTTPscrapperClient) DeleteChat(ctx context.Context, chatID int64) error {

	url := fmt.Sprintf("%s/tg-chat/%d", c.baseURL, chatID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("scrapper returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *HTTPscrapperClient) AddLink(ctx context.Context, chatID int64, urlStr string, tags []string) error {

	body := addLinkRequest{
		ChatID: chatID,
		URL:    urlStr,
		Tags:   tags,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/links",
		bytes.NewBuffer(data),
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

	if resp.StatusCode >= 300 {
		return fmt.Errorf("scrapper returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *HTTPscrapperClient) RemoveLink(ctx context.Context, chatID int64, urlStr string) error {

	body := addLinkRequest{
		ChatID: chatID,
		URL:    urlStr,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		c.baseURL+"/links",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("scrapper returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *HTTPscrapperClient) ListLinks(ctx context.Context, chatID int64) ([]domain.Link, error) {

	url := fmt.Sprintf("%s/links?chatId=%d", c.baseURL, chatID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("scrapper returned status %d", resp.StatusCode)
	}

	var links []domain.Link

	err = json.NewDecoder(resp.Body).Decode(&links)
	if err != nil {
		return nil, pkg.ErrInvalidAPIResponse
	}

	return links, nil
}
