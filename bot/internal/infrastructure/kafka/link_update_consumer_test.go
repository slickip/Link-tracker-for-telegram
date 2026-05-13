package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	confluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

const (
	testKafkaImage         = "confluentinc/confluent-local:7.5.0"
	testTopic              = "link-updates"
	topicNumPartitions     = 1
	topicReplicationFactor = 1
	messageConsumeTimeout  = 15 * time.Second
	messageDeliveryTimeout = 10 * time.Second
	fakeSenderBufferSize   = 1
	testUpdateID           = 1
	testChatID             = 123
)

type sentMessage struct {
	chatID int64
	text   string
}

type fakeMessageSender struct {
	messages chan sentMessage
}

func newFakeMessageSender() *fakeMessageSender {
	return &fakeMessageSender{
		messages: make(chan sentMessage, fakeSenderBufferSize),
	}
}

func (f *fakeMessageSender) SendMessage(chatID int64, text string) error {
	f.messages <- sentMessage{
		chatID: chatID,
		text:   text,
	}

	return nil
}

func TestLinkUpdateConsumerHandlesKafkaMessage(t *testing.T) {
	ctx := context.Background()

	kafkaContainer, err := tckafka.Run(
		ctx,
		testKafkaImage,
		tckafka.WithClusterID("test-cluster"),
	)
	if err != nil {
		t.Fatalf("start kafka container: %v", err)
	}

	defer func() {
		if err := kafkaContainer.Terminate(ctx); err != nil {
			t.Logf("terminate kafka container: %v", err)
		}
	}()

	brokers, err := kafkaContainer.Brokers(ctx)
	if err != nil {
		t.Fatalf("get kafka brokers: %v", err)
	}

	bootstrapServers := strings.Join(brokers, ",")

	createKafkaTopic(t, ctx, bootstrapServers, testTopic)

	fakeBot := newFakeMessageSender()
	updateService := services.NewUpdateService(fakeBot)

	consumer, err := NewLinkUpdateConsumer(
		LinkUpdateConsumerConfig{
			BootstrapServers: bootstrapServers,
			Topic:            testTopic,
			ConsumerGroup:    "bot-test-group",
			ClientID:         "bot-test",
		},
		updateService,
	)
	if err != nil {
		t.Fatalf("create kafka consumer: %v", err)
	}

	defer func() {
		if err := consumer.Close(); err != nil {
			t.Logf("close kafka consumer: %v", err)
		}
	}()

	consumerCtx, cancelConsumer := context.WithCancel(ctx)
	defer cancelConsumer()

	errCh := make(chan error, 1)

	go func() {
		err := consumer.Start(consumerCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
	}()

	expected := api.LinkUpdate{
		ID:          testUpdateID,
		URL:         "https://github.com/test/repo",
		TgChatIDs:   []int64{testChatID},
		Type:        "github_issue",
		Title:       "Test issue",
		Username:    "tester",
		CreatedAt:   time.Now().UTC(),
		Preview:     "Preview",
		Description: "Test description",
	}

	produceLinkUpdate(t, bootstrapServers, testTopic, expected)

	select {
	case message := <-fakeBot.messages:
		if message.chatID != expected.TgChatIDs[0] {
			t.Fatalf("expected chatID %d, got %d", expected.TgChatIDs[0], message.chatID)
		}

		if message.text != expected.Description {
			t.Fatalf("expected text %q, got %q", expected.Description, message.text)
		}

	case err := <-errCh:
		t.Fatalf("consumer failed: %v", err)

	case <-time.After(messageConsumeTimeout):
		t.Fatal("timeout waiting for consumed message")
	}
}

func createKafkaTopic(
	t *testing.T,
	ctx context.Context,
	bootstrapServers string,
	topic string,
) {
	t.Helper()

	adminClient, err := confluent.NewAdminClient(&confluent.ConfigMap{
		"bootstrap.servers": bootstrapServers,
	})
	if err != nil {
		t.Fatalf("create kafka admin client: %v", err)
	}

	defer adminClient.Close()

	results, err := adminClient.CreateTopics(ctx, []confluent.TopicSpecification{
		{
			Topic:             topic,
			NumPartitions:     topicNumPartitions,
			ReplicationFactor: topicReplicationFactor,
		},
	})
	if err != nil {
		t.Fatalf("create kafka topic: %v", err)
	}

	for _, result := range results {
		if result.Error.Code() != confluent.ErrNoError {
			t.Fatalf("create topic result error: %v", result.Error)
		}
	}
}

func produceLinkUpdate(
	t *testing.T,
	bootstrapServers string,
	topic string,
	update api.LinkUpdate,
) {
	t.Helper()

	producer, err := confluent.NewProducer(&confluent.ConfigMap{
		"bootstrap.servers": bootstrapServers,
		"client.id":         "scrapper-test",
	})
	if err != nil {
		t.Fatalf("create kafka producer: %v", err)
	}

	defer producer.Close()

	payload, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("marshal update: %v", err)
	}

	deliveryChan := make(chan confluent.Event, 1)

	if err := producer.Produce(&confluent.Message{
		TopicPartition: confluent.TopicPartition{
			Topic:     &topic,
			Partition: confluent.PartitionAny,
		},
		Key:   []byte("1"),
		Value: payload,
	}, deliveryChan); err != nil {
		t.Fatalf("produce kafka message: %v", err)
	}

	select {
	case event := <-deliveryChan:
		message, ok := event.(*confluent.Message)
		if !ok {
			t.Fatalf("unexpected kafka event type: %T", event)
		}

		if message.TopicPartition.Error != nil {
			t.Fatalf("deliver kafka message: %v", message.TopicPartition.Error)
		}

	case <-time.After(messageDeliveryTimeout):
		t.Fatal("timeout waiting for message delivery")
	}
}
