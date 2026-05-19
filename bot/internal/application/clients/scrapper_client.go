package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
)

const (
	tgChatIDHeader = "Tg-Chat-Id"
	decimalNum     = 10
)

type ScrapperClient interface {
	RegisterChat(ctx context.Context, chatID int64) error
	DeleteChat(ctx context.Context, chatID int64) error
	AddLink(ctx context.Context, chatID int64, url string, tags []string) error
	RemoveLink(ctx context.Context, chatID int64, url string) error
	RemoveLinksByTag(ctx context.Context, chatID int64, tag string) (int64, error)
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
	URL  string   `json:"url"`
	Tags []string `json:"tags"`
}

type removeLinkRequest struct {
	URL string `json:"url"`
}
type removeByTagRequest struct {
	ChatID int64  `json:"chatId"`
	Tag    string `json:"tag"`
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

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
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

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("scrapper returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *HTTPscrapperClient) AddLink(ctx context.Context, chatID int64, urlStr string, tags []string) error {
	body := addLinkRequest{
		URL:  urlStr,
		Tags: tags,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/list",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	setChatIDHeader(req, chatID)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("scrapper returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *HTTPscrapperClient) RemoveLink(ctx context.Context, chatID int64, urlStr string) error {
	body := removeLinkRequest{
		URL: urlStr,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		c.baseURL+"/list",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	setChatIDHeader(req, chatID)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("scrapper returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *HTTPscrapperClient) ListLinks(ctx context.Context, chatID int64) ([]domain.Link, error) {
	url := c.baseURL + "/list"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	setChatIDHeader(req, chatID)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scrapper returned status %d", resp.StatusCode)
	}

	var links []domain.Link

	err = json.NewDecoder(resp.Body).Decode(&links)
	if err != nil {
		return nil, pkg.ErrInvalidAPIResponse
	}

	return links, nil
}

func (c *HTTPscrapperClient) RemoveLinksByTag(ctx context.Context, chatID int64, tag string) (int64, error) {
	body := removeByTagRequest{
		ChatID: chatID,
		Tag:    tag,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		c.baseURL+"/links/by-tag",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("scrapper returned status %d", resp.StatusCode)
	}

	type removeByTagResponse struct {
		RemovedCount int64 `json:"removedCount"`
	}

	var result removeByTagResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, pkg.ErrInvalidAPIResponse
	}

	return result.RemovedCount, nil
}

func setChatIDHeader(req *http.Request, chatID int64) {
	req.Header.Set(tgChatIDHeader, strconv.FormatInt(chatID, decimalNum))
}
