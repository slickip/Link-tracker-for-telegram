package clients

import (
	"context"

	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
)

type FallbackBotClient struct {
	http BotClient
	grpc BotClient
	log  *logger.Slog
}

func NewFallbackBotClient(httpClient BotClient, grpcClient BotClient, log *logger.Slog) *FallbackBotClient {
	return &FallbackBotClient{
		http: httpClient,
		grpc: grpcClient,
		log:  log,
	}
}

func (c *FallbackBotClient) SendUpdate(ctx context.Context, update api.LinkUpdate) error {
	if err := c.http.SendUpdate(ctx, update); err != nil {
		if c.log != nil {
			c.log.Warn("bot http failed, fallback to grpc", "error", err)
		}
		return c.grpc.SendUpdate(ctx, update)
	}
	return nil
}
