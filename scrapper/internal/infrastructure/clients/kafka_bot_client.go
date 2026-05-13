package clients

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/kafka"
)

type KafkaBotClient struct {
	producer kafka.LinkUpdateProducer
}

var _ BotClient = (*KafkaBotClient)(nil)

func NewKafkaBotClient(producer kafka.LinkUpdateProducer) *KafkaBotClient {
	return &KafkaBotClient{
		producer: producer,
	}
}

func (c *KafkaBotClient) SendUpdate(ctx context.Context, update api.LinkUpdate) error {
	return c.producer.Produce(ctx, update)
}
