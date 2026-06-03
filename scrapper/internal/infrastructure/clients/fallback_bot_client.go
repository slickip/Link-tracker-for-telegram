package clients

import (
	"context"
	"errors"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
)

type FallbackBotClient struct {
	primary  BotClient
	fallback BotClient
	log      *logger.Slog
}

var _ BotClient = (*FallbackBotClient)(nil)

func NewFallbackBotClient(primary BotClient, fallback BotClient, log *logger.Slog) *FallbackBotClient {
	return &FallbackBotClient{
		primary:  primary,
		fallback: fallback,
		log:      log,
	}
}

func (c *FallbackBotClient) SendUpdate(ctx context.Context, update api.LinkUpdate) error {
	primaryErr := c.primary.SendUpdate(ctx, update)
	if primaryErr == nil {
		return nil
	}

	if c.log != nil {
		c.log.Warn(
			"primary bot notification transport failed, using fallback transport",
			"error",
			primaryErr,
		)
	}

	fallbackErr := c.fallback.SendUpdate(ctx, update)
	if fallbackErr != nil {
		return errors.Join(primaryErr, fallbackErr)
	}

	return nil
}
