package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	confluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/slickip/link-tracker/pkg/api"
	"github.com/slickip/link-tracker/pkg/avrocodec"
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

type RawMessageProducer interface {
	ProduceRaw(ctx context.Context, topic string, key string, payload []byte) error
}

type LinkUpdateProducerConfig struct {
	BootstrapServers    string
	Topic               string
	ClientID            string
	SchemaRegistryURL   string
	LinkUpdatesSubject  string
	SerializationFormat string
}

type ConfluentLinkUpdateProducer struct {
	producer *confluent.Producer
	topic    string
	codec    *avrocodec.LinkUpdateCodec
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
			producer.Close()
			return nil, err
		}
	}

	return &ConfluentLinkUpdateProducer{
		producer: producer,
		topic:    topic,
		codec:    codec,
	}, nil
}

func (p *ConfluentLinkUpdateProducer) Produce(
	ctx context.Context,
	update api.LinkUpdate,
) error {
	var (
		body []byte
		err  error
	)

	if p.codec != nil {
		body, err = p.codec.Serialize(update)
	} else {
		body, err = json.Marshal(update)
	}
	if err != nil {
		return err
	}

	key := strconv.FormatInt(update.ID, decimalBase)

	return p.produceBytes(ctx, p.topic, key, body)
}

func (p *ConfluentLinkUpdateProducer) ProduceRaw(
	ctx context.Context,
	topic string,
	key string,
	payload []byte,
) error {
	if p.codec != nil {
		var update api.LinkUpdate
		if err := json.Unmarshal(payload, &update); err != nil {
			return err
		}

		avroPayload, err := p.codec.Serialize(update)
		if err != nil {
			return err
		}

		return p.produceBytes(ctx, topic, key, avroPayload)
	}

	return p.produceBytes(ctx, topic, key, payload)
}

func (p *ConfluentLinkUpdateProducer) produceBytes(
	ctx context.Context,
	topic string,
	key string,
	payload []byte,
) error {
	deliveryChan := make(chan confluent.Event, deliveryChannelBufferSize)

	if err := p.producer.Produce(&confluent.Message{
		TopicPartition: confluent.TopicPartition{
			Topic:     &topic,
			Partition: confluent.PartitionAny,
		},
		Key:   []byte(key),
		Value: payload,
		Headers: []confluent.Header{
			{
				Key:   kafkaContentTypeHeaderKey,
				Value: []byte(kafkaJSONContentType),
			},
		},
	}, deliveryChan); err != nil {
		return err
	}

	select {
	case event := <-deliveryChan:
		message, ok := event.(*confluent.Message)
		if !ok {
			return fmt.Errorf("unexpected kafka event type: %T", event)
		}

		if message.TopicPartition.Error != nil {
			return message.TopicPartition.Error
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
