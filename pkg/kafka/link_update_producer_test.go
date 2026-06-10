//go:build integration
// +build integration

package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	confluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/slickip/link-tracker/pkg/api"
)

const testLinkID = 42

func TestConfluentLinkUpdateProducerProducesJSONMessage(t *testing.T) {
	ctx := context.Background()

	if sharedKafkaBootstrap == "" {
		t.Fatal("shared kafka bootstrap is empty")
	}

	topic := fmt.Sprintf("%s%d", integrationTestTopicPrefixMain, time.Now().UnixNano())

	createTestTopic(t, ctx, sharedKafkaBootstrap, topic)

	producer, err := NewConfluentLinkUpdateProducer(LinkUpdateProducerConfig{
		BootstrapServers: sharedKafkaBootstrap,
		Topic:            topic,
		ClientID:         integrationTestClientIDScrapper,
	})
	if err != nil {
		t.Fatalf("create link update producer: %v", err)
	}
	defer producer.Close()

	consumer, err := confluent.NewConsumer(&confluent.ConfigMap{
		"bootstrap.servers": sharedKafkaBootstrap,
		"group.id":          fmt.Sprintf("bot-test-group-%d", time.Now().UnixNano()),
		"auto.offset.reset": kafkaAutoOffsetResetEarliest,
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

	message, err := consumer.ReadMessage(integrationTestWaitConsumeMessage)
	if err != nil {
		t.Fatalf("read kafka message: %v", err)
	}

	wantKey := strconv.FormatInt(testLinkID, decimalBase)
	if string(message.Key) != wantKey {
		t.Fatalf("expected kafka key %q, got %q", wantKey, string(message.Key))
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
			NumPartitions:     integrationTestNumPartitions,
			ReplicationFactor: integrationTestReplicationFactor,
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
