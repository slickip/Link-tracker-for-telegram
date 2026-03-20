package clients

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
)

type FallbackScrapperClient struct {
	http ScrapperClient
	grpc ScrapperClient
	log  *logger.Slog
}

func NewFallbackScrapperClient(httpClient ScrapperClient, grpcClient ScrapperClient, log *logger.Slog) *FallbackScrapperClient {
	return &FallbackScrapperClient{
		http: httpClient,
		grpc: grpcClient,
		log:  log,
	}
}

func (c *FallbackScrapperClient) RegisterChat(ctx context.Context, chatID int64) error {
	if err := c.http.RegisterChat(ctx, chatID); err != nil {
		if c.log != nil {
			c.log.Warn("scrapper http failed, fallback to grpc", "error", err)
		}
		return c.grpc.RegisterChat(ctx, chatID)
	}
	return nil
}

func (c *FallbackScrapperClient) DeleteChat(ctx context.Context, chatID int64) error {
	if err := c.http.DeleteChat(ctx, chatID); err != nil {
		if c.log != nil {
			c.log.Warn("scrapper http failed, fallback to grpc", "error", err)
		}
		return c.grpc.DeleteChat(ctx, chatID)
	}
	return nil
}

func (c *FallbackScrapperClient) AddLink(ctx context.Context, chatID int64, url string, tags []string) error {
	if err := c.http.AddLink(ctx, chatID, url, tags); err != nil {
		if c.log != nil {
			c.log.Warn("scrapper http failed, fallback to grpc", "error", err)
		}
		return c.grpc.AddLink(ctx, chatID, url, tags)
	}
	return nil
}

func (c *FallbackScrapperClient) RemoveLink(ctx context.Context, chatID int64, url string) error {
	if err := c.http.RemoveLink(ctx, chatID, url); err != nil {
		if c.log != nil {
			c.log.Warn("scrapper http failed, fallback to grpc", "error", err)
		}
		return c.grpc.RemoveLink(ctx, chatID, url)
	}
	return nil
}

func (c *FallbackScrapperClient) ListLinks(ctx context.Context, chatID int64) ([]domain.Link, error) {
	links, err := c.http.ListLinks(ctx, chatID)
	if err != nil {
		if c.log != nil {
			c.log.Warn("scrapper http failed, fallback to grpc", "error", err)
		}
		return c.grpc.ListLinks(ctx, chatID)
	}
	return links, nil
}
