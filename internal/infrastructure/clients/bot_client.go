package clients

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type BotClient interface {
	SendUpdate(ctx context.Context, update domain.LinkUpdate) error
}
