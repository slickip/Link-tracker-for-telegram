package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	confluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

const (
	deadLetterProducerAcksAll          = "all"
	deadLetterProducerMessageTimeoutMS = 10000
	deadLetterProducerFlushTimeoutMS   = 10000
)

var ErrKafkaDLQTopicEmpty = errors.New("kafka dlq topic is empty")

type DeadLetterMessage struct {
	OriginalTopic     string    `json:"originalTopic"`
	OriginalPartition int32     `json:"originalPartition"`
	OriginalOffset    int64     `json:"originalOffset"`
	OriginalKey       string    `json:"originalKey"`
	OriginalPayload   string    `json:"originalPayload"`
	Reason            string    `json:"reason"`
	Error             string    `json:"error"`
	FailedAt          time.Time `json:"failedAt"`
}

type DeadLetterProducer struct {
	producer *confluent.Producer
	topic    string
}

func NewDeadLetterProducer(
	bootstrapServers string,
	topic string,
	clientID string,
) (*DeadLetterProducer, error) {
	if bootstrapServers == "" {
		return nil, ErrKafkaBootstrapServersEmpty
	}
	if topic == "" {
		return nil, ErrKafkaDLQTopicEmpty
	}

	producer, err := confluent.NewProducer(&confluent.ConfigMap{
		"bootstrap.servers":  bootstrapServers,
		"client.id":          clientID + "-dlq",
		"acks":               deadLetterProducerAcksAll,
		"message.timeout.ms": deadLetterProducerMessageTimeoutMS,
	})
	if err != nil {
		return nil, fmt.Errorf("create dlq producer: %w", err)
	}

	return &DeadLetterProducer{
		producer: producer,
		topic:    topic,
	}, nil
}

func (p *DeadLetterProducer) Produce(
	ctx context.Context,
	originalMessage *confluent.Message,
	reason string,
	processingErr error,
) error {
	dlqMessage := DeadLetterMessage{
		OriginalPartition: originalMessage.TopicPartition.Partition,
		OriginalOffset:    int64(originalMessage.TopicPartition.Offset),
		OriginalKey:       string(originalMessage.Key),
		OriginalPayload:   string(originalMessage.Value),
		Reason:            reason,
		FailedAt:          time.Now().UTC(),
	}

	if originalMessage.TopicPartition.Topic != nil {
		dlqMessage.OriginalTopic = *originalMessage.TopicPartition.Topic
	}

	if processingErr != nil {
		dlqMessage.Error = processingErr.Error()
	}

	payload, err := json.Marshal(dlqMessage)
	if err != nil {
		return fmt.Errorf("marshal dlq message: %w", err)
	}

	deliveryChan := make(chan confluent.Event, deliveryChannelBufferSize)

	if err := p.producer.Produce(&confluent.Message{
		TopicPartition: confluent.TopicPartition{
			Topic:     &p.topic,
			Partition: confluent.PartitionAny,
		},
		Key:   originalMessage.Key,
		Value: payload,
	}, deliveryChan); err != nil {
		return fmt.Errorf("produce dlq message: %w", err)
	}

	select {
	case event := <-deliveryChan:
		message, ok := event.(*confluent.Message)
		if !ok {
			return fmt.Errorf("unexpected kafka event type: %T", event)
		}

		if message.TopicPartition.Error != nil {
			return fmt.Errorf("deliver dlq message: %w", message.TopicPartition.Error)
		}

		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *DeadLetterProducer) Close() {
	if p == nil || p.producer == nil {
		return
	}

	p.producer.Flush(deadLetterProducerFlushTimeoutMS)
	p.producer.Close()
}
