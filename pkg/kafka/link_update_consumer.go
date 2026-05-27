package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	confluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/avrocodec"
)

const (
	kafkaPollTimeoutMS           = 1000
	kafkaAutoOffsetResetEarliest = "earliest"
	kafkaEnableAutoCommit        = false

	defaultRetryDelay = 500 * time.Millisecond

	consumerMaxRetriesFloor = 0
)

const (
	deadLetterReasonDeserialization = "deserialization_error"
	deadLetterReasonValidation      = "validation_error"
	deadLetterReasonProcessing      = "processing_error"
)

var (
	ErrKafkaConsumerGroupEmpty = errors.New("kafka consumer group is empty")
	ErrKafkaHandlerEmpty       = errors.New("kafka link update handler is empty")
	ErrInvalidKafkaLinkUpdate  = errors.New("invalid kafka link update")
)

type LinkUpdateHandler interface {
	HandleLinkUpdate(ctx context.Context, update api.LinkUpdate) error
}

type LinkUpdateConsumerConfig struct {
	BootstrapServers    string
	Topic               string
	DLQTopic            string
	ConsumerGroup       string
	ClientID            string
	MaxRetries          int
	SchemaRegistryURL   string
	LinkUpdatesSubject  string
	SerializationFormat string
}

type LinkUpdateConsumer struct {
	consumer    *confluent.Consumer
	dlqProducer *DeadLetterProducer
	topic       string
	handler     LinkUpdateHandler
	maxRetries  int
	retryDelay  time.Duration
	codec       *avrocodec.LinkUpdateCodec
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
	if cfg.DLQTopic == "" {
		return nil, ErrKafkaDLQTopicEmpty
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
		"enable.auto.commit": kafkaEnableAutoCommit,
	})
	if err != nil {
		return nil, fmt.Errorf("create kafka consumer: %w", err)
	}

	dlqProducer, err := NewDeadLetterProducer(
		cfg.BootstrapServers,
		cfg.DLQTopic,
		cfg.ClientID,
	)
	if err != nil {
		_ = consumer.Close()
		return nil, err
	}

	var codec *avrocodec.LinkUpdateCodec

	if strings.EqualFold(cfg.SerializationFormat, SerializationFormatAvro) {
		subject := cfg.LinkUpdatesSubject
		if subject == "" {
			subject = cfg.Topic + "-value"
		}

		codec, err = avrocodec.NewLinkUpdateCodec(
			context.Background(),
			cfg.SchemaRegistryURL,
			subject,
		)
		if err != nil {
			_ = consumer.Close()
			dlqProducer.Close()
			return nil, err
		}
	}

	maxRetries := cfg.MaxRetries
	if maxRetries < consumerMaxRetriesFloor {
		maxRetries = consumerMaxRetriesFloor
	}

	return &LinkUpdateConsumer{
		consumer:    consumer,
		dlqProducer: dlqProducer,
		topic:       cfg.Topic,
		handler:     handler,
		maxRetries:  maxRetries,
		retryDelay:  defaultRetryDelay,
		codec:       codec,
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
			event := c.consumer.Poll(kafkaPollTimeoutMS)
			if event == nil {
				continue
			}
			switch message := event.(type) {
			case *confluent.Message:
				if err := c.handleMessage(ctx, message); err != nil {
					return err
				}
			case confluent.Error:
				if message.IsFatal() {
					return fmt.Errorf("fatal kafka consumer error: %w", message)
				}
				continue
			}
		}
	}
}

func (c *LinkUpdateConsumer) handleMessage(
	ctx context.Context,
	message *confluent.Message,
) error {
	var (
		update api.LinkUpdate
		err    error
	)

	if c.codec != nil {
		update, err = c.codec.Deserialize(message.Value)
	} else {
		err = json.Unmarshal(message.Value, &update)
	}
	if err != nil {
		return c.sendToDLQAndCommit(ctx, message, deadLetterReasonDeserialization, err)
	}

	if err := validateLinkUpdate(update); err != nil {
		return c.sendToDLQAndCommit(ctx, message, deadLetterReasonValidation, err)
	}

	if err := c.handleWithRetries(ctx, update); err != nil {
		return c.sendToDLQAndCommit(ctx, message, deadLetterReasonProcessing, err)
	}

	return c.commitMessage(message)
}

func (c *LinkUpdateConsumer) handleWithRetries(
	ctx context.Context,
	update api.LinkUpdate,
) error {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if err := c.handler.HandleLinkUpdate(ctx, update); err != nil {
			lastErr = err

			if attempt < c.maxRetries {
				if err := sleepWithContext(ctx, c.retryDelay); err != nil {
					return err
				}
				continue
			}
			return lastErr
		}
		return nil
	}
	return lastErr
}

func (c *LinkUpdateConsumer) sendToDLQAndCommit(
	ctx context.Context,
	message *confluent.Message,
	reason string,
	err error,
) error {
	if dlqErr := c.dlqProducer.Produce(ctx, message, reason, err); dlqErr != nil {
		return dlqErr
	}

	return c.commitMessage(message)
}

func (c *LinkUpdateConsumer) commitMessage(message *confluent.Message) error {
	if _, err := c.consumer.CommitMessage(message); err != nil {
		return fmt.Errorf("commit kafka message: %w", err)
	}

	return nil
}

func validateLinkUpdate(update api.LinkUpdate) error {
	if update.URL == "" || len(update.TgChatIDs) == 0 {
		return ErrInvalidKafkaLinkUpdate
	}

	return nil
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *LinkUpdateConsumer) Close() error {
	if c == nil {
		return nil
	}
	if c.dlqProducer != nil {
		c.dlqProducer.Close()
	}
	if c.consumer == nil {
		return nil
	}
	return c.consumer.Close()
}
