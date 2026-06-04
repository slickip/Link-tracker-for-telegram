package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sony/gobreaker/v2"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
	h "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/helpers"
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
	baseURL        string
	client         *http.Client
	timeout        time.Duration
	retryConfig    h.HTTPRetryConfig
	circuitBreaker *gobreaker.CircuitBreaker[struct{}]
}

func NewScrapperClient(
	baseURL string,
	timeout time.Duration,
	retryConfig h.HTTPRetryConfig,
	circuitBreakerConfig h.CircuitBreakerConfig,
) *HTTPscrapperClient {
	timeout = h.NormalizeTimeout(timeout)
	return &HTTPscrapperClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
		timeout:        timeout,
		retryConfig:    h.NormalizeRetryConfig(retryConfig),
		circuitBreaker: h.NewCircuitBreaker("bot-to-scrapper", circuitBreakerConfig),
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
	Tag string `json:"tag"`
}

func (c *HTTPscrapperClient) RegisterChat(ctx context.Context, chatID int64) error {
	url := fmt.Sprintf("%s/tg-chat/%d", c.baseURL, chatID)

	return h.DoWithCircuitBreaker(c.circuitBreaker, func() error {
		return h.DoWithHTTPRetry(ctx, c.retryConfig, func() error {
			req, cancel, err := h.NewRequestWithTimeout(ctx, c.timeout, http.MethodPost, url, nil)
			if err != nil {
				return err
			}
			defer cancel()

			resp, err := c.client.Do(req)
			if err != nil {
				return err
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != http.StatusOK {
				return h.NewHTTPStatusError("scrapper", resp.StatusCode, c.retryConfig)
			}

			return nil
		})
	})
}

func (c *HTTPscrapperClient) DeleteChat(ctx context.Context, chatID int64) error {
	url := fmt.Sprintf("%s/tg-chat/%d", c.baseURL, chatID)

	return h.DoWithCircuitBreaker(c.circuitBreaker, func() error {
		return h.DoWithHTTPRetry(ctx, c.retryConfig, func() error {
			req, cancel, err := h.NewRequestWithTimeout(ctx, c.timeout, http.MethodDelete, url, nil)
			if err != nil {
				return err
			}
			defer cancel()

			resp, err := c.client.Do(req)
			if err != nil {
				return err
			}

			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != http.StatusOK {
				return h.NewHTTPStatusError("scrapper", resp.StatusCode, c.retryConfig)
			}

			return nil
		})
	})
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
	return h.DoWithCircuitBreaker(c.circuitBreaker, func() error {
		return h.DoWithHTTPRetry(ctx, c.retryConfig, func() error {
			req, cancel, err := h.NewRequestWithTimeout(
				ctx,
				c.timeout,
				http.MethodPost,
				c.baseURL+"/links",
				bytes.NewBuffer(data),
			)
			if err != nil {
				return err
			}
			defer cancel()

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
				return h.NewHTTPStatusError("scrapper", resp.StatusCode, c.retryConfig)
			}

			return nil
		})
	})
}

func (c *HTTPscrapperClient) RemoveLink(ctx context.Context, chatID int64, urlStr string) error {
	body := removeLinkRequest{
		URL: urlStr,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return h.DoWithCircuitBreaker(c.circuitBreaker, func() error {
		return h.DoWithHTTPRetry(ctx, c.retryConfig, func() error {
			req, cancel, err := h.NewRequestWithTimeout(
				ctx,
				c.timeout,
				http.MethodDelete,
				c.baseURL+"/links",
				bytes.NewBuffer(data),
			)
			if err != nil {
				return err
			}
			defer cancel()

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
				return h.NewHTTPStatusError("scrapper", resp.StatusCode, c.retryConfig)
			}

			return nil
		})
	})
}

func (c *HTTPscrapperClient) ListLinks(ctx context.Context, chatID int64) ([]domain.Link, error) {
	url := c.baseURL + "/list"

	var links []domain.Link

	err := h.DoWithCircuitBreaker(c.circuitBreaker, func() error {
		return h.DoWithHTTPRetry(ctx, c.retryConfig, func() error {
			req, cancel, err := h.NewRequestWithTimeout(ctx, c.timeout, http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			defer cancel()

			setChatIDHeader(req, chatID)

			resp, err := c.client.Do(req)
			if err != nil {
				return err
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != http.StatusOK {
				return h.NewHTTPStatusError("scrapper", resp.StatusCode, c.retryConfig)
			}

			links = nil
			if err := json.NewDecoder(resp.Body).Decode(&links); err != nil {
				return pkg.ErrInvalidAPIResponse
			}

			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return links, nil
}

func (c *HTTPscrapperClient) RemoveLinksByTag(ctx context.Context, chatID int64, tag string) (int64, error) {
	body := removeByTagRequest{
		Tag: tag,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}

	var removedCount int64
	err = h.DoWithCircuitBreaker(c.circuitBreaker, func() error {
		return h.DoWithHTTPRetry(ctx, c.retryConfig, func() error {
			req, cancel, err := h.NewRequestWithTimeout(
				ctx,
				c.timeout,
				http.MethodDelete,
				c.baseURL+"/links/by-tag",
				bytes.NewBuffer(data),
			)
			if err != nil {
				return err
			}
			defer cancel()

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
				return h.NewHTTPStatusError("scrapper", resp.StatusCode, c.retryConfig)
			}

			type removeByTagResponse struct {
				RemovedCount int64 `json:"removedCount"`
			}

			var result removeByTagResponse
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return pkg.ErrInvalidAPIResponse
			}
			removedCount = result.RemovedCount
			return nil
		})
	})
	if err != nil {
		return 0, err
	}
	return removedCount, nil
}

func setChatIDHeader(req *http.Request, chatID int64) {
	req.Header.Set(tgChatIDHeader, strconv.FormatInt(chatID, decimalNum))
}
