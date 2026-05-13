package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	confluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

const (
	producerAcksAll           = "all"
	producerEnableIdempotence = true
	producerMessageTimeoutMS  = 10000
	producerFlushTimeoutMS    = 10000
	kafkaContentTypeHeaderKey = "content-type"
	kafkaJSONContentType      = "application/json"
	decimalBase               = 10
	deliveryChannelBufferSize = 1
)

var (
	ErrKafkaBootstrapServersEmpty = errors.New("kafka bootstrap servers are empty")
	ErrKafkaTopicEmpty            = errors.New("kafka topic is empty")
)

type LinkUpdateProducer interface {
	Produce(ctx context.Context, update api.LinkUpdate) error
}

type LinkUpdateProducerConfig struct {
	BootstrapServers string
	Topic            string
	ClientID         string
}

type ConfluentLinkUpdateProducer struct {
	producer *confluent.Producer
	topic    string
}

var _ LinkUpdateProducer = (*ConfluentLinkUpdateProducer)(nil)

func NewConfluentLinkUpdateProducer(
	cfg LinkUpdateProducerConfig,
) (*ConfluentLinkUpdateProducer, error) {
	bootstrapServers := strings.TrimSpace(cfg.BootstrapServers)
	if bootstrapServers == "" {
		return nil, ErrKafkaBootstrapServersEmpty
	}

	topic := strings.TrimSpace(cfg.Topic)
	if topic == "" {
		return nil, ErrKafkaTopicEmpty
	}

	producer, err := confluent.NewProducer(&confluent.ConfigMap{
		"bootstrap.servers":  bootstrapServers,
		"client.id":          cfg.ClientID,
		"acks":               producerAcksAll,
		"enable.idempotence": producerEnableIdempotence,
		"message.timeout.ms": producerMessageTimeoutMS,
	})
	if err != nil {
		return nil, fmt.Errorf("create kafka producer: %w", err)
	}

	return &ConfluentLinkUpdateProducer{
		producer: producer,
		topic:    topic,
	}, nil
}

func (p *ConfluentLinkUpdateProducer) Produce(
	ctx context.Context,
	update api.LinkUpdate,
) error {
	body, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("marshal link update: %w", err)
	}

	key := strconv.FormatInt(update.ID, decimalBase)
	deliveryChan := make(chan confluent.Event, deliveryChannelBufferSize)

	if err := p.producer.Produce(&confluent.Message{
		TopicPartition: confluent.TopicPartition{
			Topic:     &p.topic,
			Partition: confluent.PartitionAny,
		},
		Key:   []byte(key),
		Value: body,
		Headers: []confluent.Header{
			{
				Key:   kafkaContentTypeHeaderKey,
				Value: []byte(kafkaJSONContentType),
			},
		},
	}, deliveryChan); err != nil {
		return fmt.Errorf("produce link update: %w", err)
	}

	select {
	case event := <-deliveryChan:
		message, ok := event.(*confluent.Message)
		if !ok {
			return fmt.Errorf("unexpected kafka event type: %T", event)
		}

		if message.TopicPartition.Error != nil {
			return fmt.Errorf("deliver link update: %w", message.TopicPartition.Error)
		}

		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *ConfluentLinkUpdateProducer) Close() {
	if p == nil || p.producer == nil {
		return
	}

	p.producer.Flush(producerFlushTimeoutMS)
	p.producer.Close()
}
