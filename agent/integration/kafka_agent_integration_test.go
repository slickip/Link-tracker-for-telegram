package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	confluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/require"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/agent/internal/application"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/agent/internal/infrastructure/ai"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	appkafka "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/kafka"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
)

func TestAIAgent_ShouldPublishProcessedMessage_TC_3_1(t *testing.T) {
	ctx := context.Background()

	brokers := startKafka(t, ctx)

	rawTopic := uniqueTopic("link.raw-updates")
	processedTopic := uniqueTopic("link.processed-updates")
	dlqTopic := uniqueTopic("link.raw-updates-dlq")

	createTopics(t, brokers, rawTopic, processedTopic, dlqTopic)

	log := logger.New(slog.LevelInfo)

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
			[]string{"spam"},
			[]string{"bot-user"},
			1,
		),
		ai.NewStubSummarizer(log),
		application.NewPriorityService(
			[]string{"critical", "urgent", "security"},
			[]string{"minor", "typo", "docs"},
		),
		500,
	)

	groupingService := application.NewGroupingService(
		20*time.Millisecond,
		processedProducer,
		log,
	)

	handler := application.NewHandler(
		processor,
		groupingService,
		log,
	)

	consumer, err := appkafka.NewLinkUpdateConsumer(
		appkafka.LinkUpdateConsumerConfig{
			BootstrapServers:    brokers,
			Topic:               rawTopic,
			DLQTopic:            dlqTopic,
			ConsumerGroup:       uniqueTopic("ai-agent-test-group"),
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
		TgChatIDs:   []int64{111},
		Type:        "issue",
		Title:       "Critical issue",
		Username:    "normal-user",
		CreatedAt:   time.Now().UTC(),
		Preview:     "preview",
		Description: "critical bug fix in kafka processing",
	}

	require.NoError(t, rawProducer.Produce(ctx, update))

	processed := consumeLinkUpdate(t, brokers, processedTopic, 20*time.Second)

	require.Equal(t, update.ID, processed.ID)
	require.Equal(t, update.URL, processed.URL)
	require.Equal(t, []int64{111}, processed.TgChatIDs)
	require.Equal(t, update.Description, processed.Description)
	require.Equal(t, string(application.PriorityHigh), processed.Priority)

	cancel()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(3 * time.Second):
		t.Fatal("consumer did not stop")
	}
}

func TestAIAgent_ShouldNotPublishFilteredMessage_TC_3_2(t *testing.T) {
	ctx := context.Background()

	brokers := startKafka(t, ctx)

	rawTopic := uniqueTopic("link.raw-updates")
	processedTopic := uniqueTopic("link.processed-updates")
	dlqTopic := uniqueTopic("link.raw-updates-dlq")

	createTopics(t, brokers, rawTopic, processedTopic, dlqTopic)

	log := logger.New(slog.LevelInfo)

	processedProducer, err := appkafka.NewConfluentLinkUpdateProducer(
		appkafka.LinkUpdateProducerConfig{
			BootstrapServers:    brokers,
			Topic:               processedTopic,
			ClientID:            "ai-agent-filtered-test-producer",
			SerializationFormat: appkafka.SerializationFormatJSON,
		},
	)
	require.NoError(t, err)
	defer processedProducer.Close()

	processor := application.NewProcessor(
		application.NewFilterService(
			[]string{"spam"},
			nil,
			1,
		),
		ai.NewStubSummarizer(log),
		application.NewPriorityService(
			[]string{"critical", "urgent", "security"},
			[]string{"minor", "typo", "docs"},
		),
		500,
	)

	groupingService := application.NewGroupingService(
		20*time.Millisecond,
		processedProducer,
		log,
	)

	handler := application.NewHandler(
		processor,
		groupingService,
		log,
	)

	consumer, err := appkafka.NewLinkUpdateConsumer(
		appkafka.LinkUpdateConsumerConfig{
			BootstrapServers:    brokers,
			Topic:               rawTopic,
			DLQTopic:            dlqTopic,
			ConsumerGroup:       uniqueTopic("ai-agent-filtered-test-group"),
			ClientID:            "ai-agent-filtered-test-consumer",
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
			ClientID:            "scrapper-filtered-test-producer",
			SerializationFormat: appkafka.SerializationFormatJSON,
		},
	)
	require.NoError(t, err)
	defer rawProducer.Close()

	update := api.LinkUpdate{
		ID:          54321,
		URL:         "https://github.com/test/repo",
		TgChatIDs:   []int64{111},
		Type:        "issue",
		Title:       "Spam issue",
		Username:    "normal-user",
		CreatedAt:   time.Now().UTC(),
		Preview:     "preview",
		Description: "spam message should be filtered",
	}

	require.NoError(t, rawProducer.Produce(ctx, update))

	requireNoMessage(t, brokers, processedTopic, 2*time.Second)

	cancel()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(3 * time.Second):
		t.Fatal("consumer did not stop")
	}
}

func startKafka(t *testing.T, ctx context.Context) string {
	t.Helper()

	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("skipping integration test: RUN_INTEGRATION_TESTS is not true")
	}

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

func consumeLinkUpdate(
	t *testing.T,
	brokers string,
	topic string,
	timeout time.Duration,
) api.LinkUpdate {
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

	deadline := time.After(timeout)

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

func requireNoMessage(
	t *testing.T,
	brokers string,
	topic string,
	timeout time.Duration,
) {
	t.Helper()

	consumer, err := confluent.NewConsumer(&confluent.ConfigMap{
		"bootstrap.servers":  brokers,
		"group.id":           uniqueTopic("empty-reader"),
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
	})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, consumer.Close())
	}()

	require.NoError(t, consumer.SubscribeTopics([]string{topic}, nil))

	deadline := time.After(timeout)

	for {
		select {
		case <-deadline:
			return
		default:
			event := consumer.Poll(100)
			if event == nil {
				continue
			}

			if _, ok := event.(*confluent.Message); ok {
				t.Fatal("unexpected message in processed topic")
			}
		}
	}
}
