package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	confluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

const (
	kafkaPollTimeoutMs           = 1000
	kafkaAutoOffsetResetEarliest = "earliest"
	kafkaEnableAutoCommitFalse   = false
)

var (
	ErrKafkaBootstrapServersEmpty = errors.New("kafka bootstrap servers are empty")
	ErrKafkaTopicEmpty            = errors.New("kafka topic is empty")
	ErrKafkaConsumerGroupEmpty    = errors.New("kafka consumer group is empty")
	ErrKafkaHandlerEmpty          = errors.New("kafka link update handler is empty")
)

type LinkUpdateHandler interface {
	HandleLinkUpdate(ctx context.Context, update api.LinkUpdate) error
}

type LinkUpdateConsumerConfig struct {
	BootstrapServers string
	Topic            string
	ConsumerGroup    string
	ClientID         string
}

type LinkUpdateConsumer struct {
	consumer *confluent.Consumer
	topic    string
	handler  LinkUpdateHandler
}

func NewLinkUpdateConsumer(
	cfg LinkUpdateConsumerConfig,
	handler LinkUpdateHandler,
) (*LinkUpdateConsumer, error) {
	if cfg.BootstrapServers == "" {
		return nil, ErrKafkaBootstrapServersEmpty
	}
	if cfg.Topic == "" {
		return nil, ErrKafkaTopicEmpty
	}
	if cfg.ConsumerGroup == "" {
		return nil, ErrKafkaConsumerGroupEmpty
	}
	if handler == nil {
		return nil, ErrKafkaHandlerEmpty
	}

	consumer, err := confluent.NewConsumer(&confluent.ConfigMap{
		"bootstrap.servers":  cfg.BootstrapServers,
		"group.id":           cfg.ConsumerGroup,
		"client.id":          cfg.ClientID,
		"auto.offset.reset":  kafkaAutoOffsetResetEarliest,
		"enable.auto.commit": kafkaEnableAutoCommitFalse,
	})
	if err != nil {
		return nil, fmt.Errorf("create kafka consumer: %w", err)
	}

	return &LinkUpdateConsumer{
		consumer: consumer,
		topic:    cfg.Topic,
		handler:  handler,
	}, nil
}

func (c *LinkUpdateConsumer) Start(ctx context.Context) error {
	if err := c.consumer.SubscribeTopics([]string{c.topic}, nil); err != nil {
		return fmt.Errorf("subscribe to kafka topic: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
			event := c.consumer.Poll(kafkaPollTimeoutMs)
			if event == nil {
				continue
			}

			switch message := event.(type) {
			case *confluent.Message:
				if err := c.handleMessage(ctx, message); err != nil {
					return err
				}

			case confluent.Error:
				return fmt.Errorf("kafka consumer error: %w", message)
			}
		}
	}
}

func (c *LinkUpdateConsumer) handleMessage(
	ctx context.Context,
	message *confluent.Message,
) error {
	var update api.LinkUpdate
	if err := json.Unmarshal(message.Value, &update); err != nil {
		return fmt.Errorf("unmarshal link update: %w", err)
	}

	if err := c.handler.HandleLinkUpdate(ctx, update); err != nil {
		return fmt.Errorf("handle link update: %w", err)
	}

	if _, err := c.consumer.CommitMessage(message); err != nil {
		return fmt.Errorf("commit kafka message: %w", err)
	}

	return nil
}

func (c *LinkUpdateConsumer) Close() error {
	if c == nil || c.consumer == nil {
		return nil
	}

	return c.consumer.Close()
}
