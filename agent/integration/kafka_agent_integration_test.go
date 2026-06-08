package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	confluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/require"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/agent/internal/application"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/agent/internal/infrastructure/ai"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	appkafka "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/kafka"
)

type testAgentHandler struct {
	processor *application.Processor
	producer  appkafka.LinkUpdateProducer
}

func (h *testAgentHandler) HandleLinkUpdate(
	ctx context.Context,
	update api.LinkUpdate,
) error {
	processed, ok, err := h.processor.Process(ctx, update)
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	return h.producer.Produce(ctx, processed)
}

func TestAIAgent_ShouldReceiveAndProcessCorrectKafkaMessage_TC_1_1(t *testing.T) {
	ctx := context.Background()

	brokers := startKafka(t, ctx)

	rawTopic := uniqueTopic("link.raw-updates")
	processedTopic := uniqueTopic("link.processed-updates")
	dlqTopic := uniqueTopic("link.raw-updates-dlq")

	createTopics(t, brokers, rawTopic, processedTopic, dlqTopic)

	processedProducer, err := appkafka.NewConfluentLinkUpdateProducer(
		appkafka.LinkUpdateProducerConfig{
			BootstrapServers:    brokers,
			Topic:               processedTopic,
			ClientID:            "ai-agent-test-producer",
			SerializationFormat: appkafka.SerializationFormatJSON,
		},
	)
	require.NoError(t, err)
	defer processedProducer.Close()

	processor := application.NewProcessor(
		application.NewFilterService(
			[]string{"spam", "promo"},
			[]string{"bot-user"},
			20,
		),
		ai.NewStubSummarizer(),
		500,
	)

	handler := &testAgentHandler{
		processor: processor,
		producer:  processedProducer,
	}

	consumer, err := appkafka.NewLinkUpdateConsumer(
		appkafka.LinkUpdateConsumerConfig{
			BootstrapServers:    brokers,
			Topic:               rawTopic,
			DLQTopic:            dlqTopic,
			ConsumerGroup:       "ai-agent-test-group",
			ClientID:            "ai-agent-test-consumer",
			MaxRetries:          0,
			SerializationFormat: appkafka.SerializationFormatJSON,
		},
		handler,
	)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, consumer.Close())
	}()

	consumerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.Start(consumerCtx)
	}()

	rawProducer, err := appkafka.NewConfluentLinkUpdateProducer(
		appkafka.LinkUpdateProducerConfig{
			BootstrapServers:    brokers,
			Topic:               rawTopic,
			ClientID:            "scrapper-test-producer",
			SerializationFormat: appkafka.SerializationFormatJSON,
		},
	)
	require.NoError(t, err)
	defer rawProducer.Close()

	update := api.LinkUpdate{
		ID:          12345,
		URL:         "https://github.com/test/repo",
		TgChatIDs:   []int64{111, 222},
		Type:        "issue",
		Title:       "Test issue",
		Username:    "normal-user",
		CreatedAt:   time.Now().UTC(),
		Preview:     "preview",
		Description: "This is a valid update that should pass filtering and be published to processed topic",
	}

	require.NoError(t, rawProducer.Produce(ctx, update))

	processed := consumeLinkUpdate(t, brokers, processedTopic)

	require.Equal(t, update.ID, processed.ID)
	require.Equal(t, update.URL, processed.URL)
	require.Equal(t, update.TgChatIDs, processed.TgChatIDs)
	require.Equal(t, update.Description, processed.Description)
	require.Equal(t, application.DefaultPriority, processed.Priority)

	cancel()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(3 * time.Second):
		t.Fatal("consumer did not stop")
	}
}

func TestAIAgent_ShouldHandleInvalidKafkaMessageWithoutCrash_TC_1_2(t *testing.T) {
	ctx := context.Background()

	brokers := startKafka(t, ctx)

	rawTopic := uniqueTopic("link.raw-updates")
	processedTopic := uniqueTopic("link.processed-updates")
	dlqTopic := uniqueTopic("link.raw-updates-dlq")

	createTopics(t, brokers, rawTopic, processedTopic, dlqTopic)

	processedProducer, err := appkafka.NewConfluentLinkUpdateProducer(
		appkafka.LinkUpdateProducerConfig{
			BootstrapServers:    brokers,
			Topic:               processedTopic,
			ClientID:            "ai-agent-invalid-test-producer",
			SerializationFormat: appkafka.SerializationFormatJSON,
		},
	)
	require.NoError(t, err)
	defer processedProducer.Close()

	processor := application.NewProcessor(
		application.NewFilterService(nil, nil, 1),
		ai.NewStubSummarizer(),
		500,
	)

	handler := &testAgentHandler{
		processor: processor,
		producer:  processedProducer,
	}

	consumer, err := appkafka.NewLinkUpdateConsumer(
		appkafka.LinkUpdateConsumerConfig{
			BootstrapServers:    brokers,
			Topic:               rawTopic,
			DLQTopic:            dlqTopic,
			ConsumerGroup:       "ai-agent-invalid-test-group",
			ClientID:            "ai-agent-invalid-test-consumer",
			MaxRetries:          0,
			SerializationFormat: appkafka.SerializationFormatJSON,
		},
		handler,
	)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, consumer.Close())
	}()

	consumerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.Start(consumerCtx)
	}()

	produceRawBytes(t, brokers, rawTopic, []byte(`{"id": "invalid-id", "tgChatIds": "not-array"}`))

	dlqMessage := consumeDLQMessage(t, brokers, dlqTopic)

	require.Equal(t, rawTopic, dlqMessage.OriginalTopic)
	require.Equal(t, appkafka.SerializationFormatJSON, appkafka.SerializationFormatJSON)
	require.Equal(t, "deserialization_error", dlqMessage.Reason)
	require.NotEmpty(t, dlqMessage.Error)

	cancel()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(3 * time.Second):
		t.Fatal("consumer did not stop after invalid message")
	}
}

func startKafka(t *testing.T, ctx context.Context) string {
	t.Helper()

	kafkaContainer, err := tckafka.Run(
		ctx,
		"confluentinc/confluent-local:7.5.0",
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, kafkaContainer.Terminate(ctx))
	})

	brokers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, brokers)

	return brokers[0]
}

func uniqueTopic(prefix string) string {
	return fmt.Sprintf("%s.%d", prefix, time.Now().UnixNano())
}

func createTopics(t *testing.T, brokers string, topics ...string) {
	t.Helper()

	admin, err := confluent.NewAdminClient(&confluent.ConfigMap{
		"bootstrap.servers": brokers,
	})
	require.NoError(t, err)
	defer admin.Close()

	specs := make([]confluent.TopicSpecification, 0, len(topics))
	for _, topic := range topics {
		specs = append(specs, confluent.TopicSpecification{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
			Config: map[string]string{
				"min.insync.replicas": "1",
			},
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := admin.CreateTopics(ctx, specs)
	require.NoError(t, err)

	for _, result := range results {
		if result.Error.Code() != confluent.ErrNoError &&
			result.Error.Code() != confluent.ErrTopicAlreadyExists {
			require.NoError(t, result.Error)
		}
	}
}

func consumeLinkUpdate(t *testing.T, brokers string, topic string) api.LinkUpdate {
	t.Helper()

	consumer, err := confluent.NewConsumer(&confluent.ConfigMap{
		"bootstrap.servers":  brokers,
		"group.id":           uniqueTopic("processed-reader"),
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
	})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, consumer.Close())
	}()

	require.NoError(t, consumer.SubscribeTopics([]string{topic}, nil))

	deadline := time.After(20 * time.Second)

	for {
		select {
		case <-deadline:
			t.Fatal("processed update was not received")
		default:
			event := consumer.Poll(100)
			if event == nil {
				continue
			}

			message, ok := event.(*confluent.Message)
			if !ok {
				continue
			}

			var update api.LinkUpdate
			require.NoError(t, json.Unmarshal(message.Value, &update))

			return update
		}
	}
}

func consumeDLQMessage(t *testing.T, brokers string, topic string) appkafka.DeadLetterMessage {
	t.Helper()

	consumer, err := confluent.NewConsumer(&confluent.ConfigMap{
		"bootstrap.servers":  brokers,
		"group.id":           uniqueTopic("dlq-reader"),
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
	})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, consumer.Close())
	}()

	require.NoError(t, consumer.SubscribeTopics([]string{topic}, nil))

	deadline := time.After(20 * time.Second)

	for {
		select {
		case <-deadline:
			t.Fatal("DLQ message was not received")
		default:
			event := consumer.Poll(100)
			if event == nil {
				continue
			}

			message, ok := event.(*confluent.Message)
			if !ok {
				continue
			}

			var dlqMessage appkafka.DeadLetterMessage
			require.NoError(t, json.Unmarshal(message.Value, &dlqMessage))

			return dlqMessage
		}
	}
}

func produceRawBytes(t *testing.T, brokers string, topic string, payload []byte) {
	t.Helper()

	producer, err := confluent.NewProducer(&confluent.ConfigMap{
		"bootstrap.servers": brokers,
		"acks":              "all",
	})
	require.NoError(t, err)
	defer producer.Close()

	deliveryCh := make(chan confluent.Event, 1)

	err = producer.Produce(&confluent.Message{
		TopicPartition: confluent.TopicPartition{
			Topic:     &topic,
			Partition: confluent.PartitionAny,
		},
		Value: payload,
	}, deliveryCh)
	require.NoError(t, err)

	select {
	case event := <-deliveryCh:
		message, ok := event.(*confluent.Message)
		require.True(t, ok)
		require.NoError(t, message.TopicPartition.Error)
	case <-time.After(10 * time.Second):
		t.Fatal("raw invalid kafka message was not delivered")
	}
}
