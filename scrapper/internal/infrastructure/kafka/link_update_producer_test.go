package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	confluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

const (
	testKafkaImage         = "confluentinc/confluent-local:7.5.0"
	testLinkID             = 42
	messageReadTimeout     = 15 * time.Second
	topicNumPartitions     = 1
	topicReplicationFactor = 1
)

func TestConfluentLinkUpdateProducerProducesJSONMessage(t *testing.T) {
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
	topic := fmt.Sprintf("link-updates-%d", time.Now().UnixNano())

	createTestTopic(t, ctx, bootstrapServers, topic)

	producer, err := NewConfluentLinkUpdateProducer(LinkUpdateProducerConfig{
		BootstrapServers: bootstrapServers,
		Topic:            topic,
		ClientID:         "scrapper-test",
	})
	if err != nil {
		t.Fatalf("create link update producer: %v", err)
	}
	defer producer.Close()

	consumer, err := confluent.NewConsumer(&confluent.ConfigMap{
		"bootstrap.servers": bootstrapServers,
		"group.id":          "bot-test-group",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		t.Fatalf("create kafka consumer: %v", err)
	}

	defer func() {
		if err := consumer.Close(); err != nil {
			t.Logf("close kafka consumer: %v", err)
		}
	}()

	if err := consumer.SubscribeTopics([]string{topic}, nil); err != nil {
		t.Fatalf("subscribe to topic: %v", err)
	}

	expected := api.LinkUpdate{
		ID:          testLinkID,
		URL:         "https://github.com/golang/go",
		TgChatIDs:   []int64{1001, 1002},
		Type:        "github_issue",
		Title:       "Test issue",
		Username:    "tester",
		CreatedAt:   time.Now().UTC(),
		Preview:     "Short preview",
		Description: "Test description",
	}

	if err := producer.Produce(ctx, expected); err != nil {
		t.Fatalf("produce link update: %v", err)
	}

	message, err := consumer.ReadMessage(messageReadTimeout)
	if err != nil {
		t.Fatalf("read kafka message: %v", err)
	}

	if string(message.Key) != "42" {
		t.Fatalf("expected kafka key %q, got %q", "42", string(message.Key))
	}

	assertContentTypeHeader(t, message.Headers)

	var actual api.LinkUpdate
	if err := json.Unmarshal(message.Value, &actual); err != nil {
		t.Fatalf("unmarshal kafka message: %v", err)
	}

	if actual.ID != expected.ID {
		t.Fatalf("expected ID %d, got %d", expected.ID, actual.ID)
	}

	if actual.URL != expected.URL {
		t.Fatalf("expected URL %q, got %q", expected.URL, actual.URL)
	}

	if actual.Description != expected.Description {
		t.Fatalf("expected description %q, got %q", expected.Description, actual.Description)
	}

	if len(actual.TgChatIDs) != len(expected.TgChatIDs) {
		t.Fatalf("expected TgChatIDs %v, got %v", expected.TgChatIDs, actual.TgChatIDs)
	}

	for i := range expected.TgChatIDs {
		if actual.TgChatIDs[i] != expected.TgChatIDs[i] {
			t.Fatalf("expected TgChatIDs %v, got %v", expected.TgChatIDs, actual.TgChatIDs)
		}
	}
}

func createTestTopic(
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

func assertContentTypeHeader(t *testing.T, headers []confluent.Header) {
	t.Helper()

	for _, header := range headers {
		if header.Key == kafkaContentTypeHeaderKey &&
			string(header.Value) == kafkaJSONContentType {
			return
		}
	}

	t.Fatalf("expected kafka header %s=%s", kafkaContentTypeHeaderKey, kafkaJSONContentType)
}
