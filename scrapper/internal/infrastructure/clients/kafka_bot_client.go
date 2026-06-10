package clients

import (
	"context"

	"github.com/slickip/link-tracker/pkg/api"
	"github.com/slickip/link-tracker/pkg/kafka"
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
