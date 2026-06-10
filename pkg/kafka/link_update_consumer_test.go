//go:build integration
// +build integration

package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	confluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/slickip/link-tracker/pkg/api"
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
		messages: make(chan sentMessage, integrationTestSentMessageChanBuffer),
	}
}

func (f *fakeMessageSender) HandleLinkUpdate(_ context.Context, update api.LinkUpdate) error {
	text := update.SubscriberNotificationBody()

	for _, chatID := range update.TgChatIDs {
		f.messages <- sentMessage{
			chatID: chatID,
			text:   text,
		}
	}

	return nil
}

type failingHandler struct {
	mu    sync.Mutex
	calls int
	err   error
}

func newFailingHandler(err error) *failingHandler {
	return &failingHandler{
		err: err,
	}
}

func (h *failingHandler) HandleLinkUpdate(_ context.Context, _ api.LinkUpdate) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.calls++

	return h.err
}

func (h *failingHandler) Calls() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.calls
}

func TestLinkUpdateConsumerHandlesKafkaMessage(t *testing.T) {
	ctx := context.Background()

	mainTopic, dlqTopic := uniqueTopicPair(t)
	bootstrapServers := prepareKafkaTopics(t, ctx, mainTopic, dlqTopic)

	fakeBot := newFakeMessageSender()

	consumer := newTestConsumer(
		t,
		bootstrapServers,
		mainTopic,
		dlqTopic,
		fmt.Sprintf("bot-test-group-happy-path-%d", time.Now().UnixNano()),
		integrationTestDefaultMaxRetries,
		fakeBot,
	)
	defer closeConsumer(t, consumer)

	consumerCtx, cancelConsumer := context.WithCancel(ctx)
	defer cancelConsumer()

	errCh := startConsumer(consumerCtx, consumer)

	expected := api.LinkUpdate{
		ID:          1,
		URL:         "https://github.com/test/repo",
		TgChatIDs:   []int64{123},
		Type:        "github_issue",
		Title:       "Test issue",
		Username:    "tester",
		CreatedAt:   time.Now().UTC(),
		Preview:     "Preview",
		Description: "Test description",
	}

	produceRawMessage(t, bootstrapServers, mainTopic, []byte("1"), mustMarshal(t, expected))

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

	case <-time.After(integrationTestWaitConsumeMessage):
		t.Fatal("timeout waiting for consumed message")
	}
}

func TestLinkUpdateConsumerSendsInvalidJSONToDLQ(t *testing.T) {
	ctx := context.Background()

	mainTopic, dlqTopic := uniqueTopicPair(t)
	bootstrapServers := prepareKafkaTopics(t, ctx, mainTopic, dlqTopic)

	handler := newFailingHandler(errors.New("handler must not be called"))

	consumer := newTestConsumer(
		t,
		bootstrapServers,
		mainTopic,
		dlqTopic,
		fmt.Sprintf("bot-test-group-invalid-json-%d", time.Now().UnixNano()),
		integrationTestDefaultMaxRetries,
		handler,
	)
	defer closeConsumer(t, consumer)

	consumerCtx, cancelConsumer := context.WithCancel(ctx)
	defer cancelConsumer()

	errCh := startConsumer(consumerCtx, consumer)

	invalidPayload := []byte(`{"id":`)
	produceRawMessage(t, bootstrapServers, mainTopic, []byte("bad-json"), invalidPayload)

	dlq := readDLQMessage(t, bootstrapServers, dlqTopic)

	if dlq.Reason != deadLetterReasonDeserialization {
		t.Fatalf("expected DLQ reason %q, got %q", deadLetterReasonDeserialization, dlq.Reason)
	}

	if dlq.OriginalPayload != string(invalidPayload) {
		t.Fatalf("expected original payload %q, got %q", string(invalidPayload), dlq.OriginalPayload)
	}

	if handler.Calls() != integrationTestExpectZeroHandlerCalls {
		t.Fatalf("expected handler calls %d, got %d", integrationTestExpectZeroHandlerCalls, handler.Calls())
	}

	select {
	case err := <-errCh:
		t.Fatalf("consumer failed: %v", err)
	default:
	}
}

func TestLinkUpdateConsumerRetriesAndSendsProcessingErrorToDLQ(t *testing.T) {
	ctx := context.Background()

	mainTopic, dlqTopic := uniqueTopicPair(t)
	bootstrapServers := prepareKafkaTopics(t, ctx, mainTopic, dlqTopic)

	processingErr := errors.New("telegram is temporarily unavailable")
	handler := newFailingHandler(processingErr)

	maxRetries := integrationTestProcessingMaxRetries

	consumer := newTestConsumer(
		t,
		bootstrapServers,
		mainTopic,
		dlqTopic,
		fmt.Sprintf("bot-test-group-processing-error-%d", time.Now().UnixNano()),
		maxRetries,
		handler,
	)
	defer closeConsumer(t, consumer)

	consumerCtx, cancelConsumer := context.WithCancel(ctx)
	defer cancelConsumer()

	errCh := startConsumer(consumerCtx, consumer)

	update := api.LinkUpdate{
		ID:          10,
		URL:         "https://github.com/test/repo",
		TgChatIDs:   []int64{123},
		Description: "Test description",
	}

	produceRawMessage(t, bootstrapServers, mainTopic, []byte("10"), mustMarshal(t, update))

	dlq := readDLQMessage(t, bootstrapServers, dlqTopic)

	if dlq.Reason != deadLetterReasonProcessing {
		t.Fatalf("expected DLQ reason %q, got %q", deadLetterReasonProcessing, dlq.Reason)
	}

	if !strings.Contains(dlq.Error, processingErr.Error()) {
		t.Fatalf("expected DLQ error to contain %q, got %q", processingErr.Error(), dlq.Error)
	}

	expectedCalls := maxRetries + 1
	if handler.Calls() != expectedCalls {
		t.Fatalf("expected handler calls %d, got %d", expectedCalls, handler.Calls())
	}

	select {
	case err := <-errCh:
		t.Fatalf("consumer failed: %v", err)
	default:
	}
}

func uniqueTopicPair(t *testing.T) (main string, dlq string) {
	t.Helper()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	return integrationTestTopicPrefixMain + suffix, integrationTestTopicPrefixDLQ + suffix
}

func prepareKafkaTopics(t *testing.T, ctx context.Context, topics ...string) string {
	t.Helper()

	if sharedKafkaBootstrap == "" {
		t.Fatal("shared kafka bootstrap is empty")
	}

	createKafkaTopics(t, ctx, sharedKafkaBootstrap, topics...)

	return sharedKafkaBootstrap
}

func createKafkaTopics(
	t *testing.T,
	ctx context.Context,
	bootstrapServers string,
	topics ...string,
) {
	t.Helper()

	adminClient, err := confluent.NewAdminClient(&confluent.ConfigMap{
		"bootstrap.servers": bootstrapServers,
	})
	if err != nil {
		t.Fatalf("create kafka admin client: %v", err)
	}

	defer adminClient.Close()

	specs := make([]confluent.TopicSpecification, 0, len(topics))
	for _, topic := range topics {
		specs = append(specs, confluent.TopicSpecification{
			Topic:             topic,
			NumPartitions:     integrationTestNumPartitions,
			ReplicationFactor: integrationTestReplicationFactor,
		})
	}

	results, err := adminClient.CreateTopics(ctx, specs)
	if err != nil {
		t.Fatalf("create kafka topics: %v", err)
	}

	for _, result := range results {
		if result.Error.Code() != confluent.ErrNoError {
			t.Fatalf("create topic result error: %v", result.Error)
		}
	}
}

func newTestConsumer(
	t *testing.T,
	bootstrapServers string,
	mainTopic string,
	dlqTopic string,
	consumerGroup string,
	maxRetries int,
	handler LinkUpdateHandler,
) *LinkUpdateConsumer {
	t.Helper()

	consumer, err := NewLinkUpdateConsumer(
		LinkUpdateConsumerConfig{
			BootstrapServers: bootstrapServers,
			Topic:            mainTopic,
			DLQTopic:         dlqTopic,
			ConsumerGroup:    consumerGroup,
			ClientID:         integrationTestClientIDBot,
			MaxRetries:       maxRetries,
		},
		handler,
	)
	if err != nil {
		t.Fatalf("create kafka consumer: %v", err)
	}

	return consumer
}

func startConsumer(
	ctx context.Context,
	consumer *LinkUpdateConsumer,
) <-chan error {
	errCh := make(chan error, integrationTestErrChanBuffer)

	go func() {
		err := consumer.Start(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
	}()

	return errCh
}

func closeConsumer(t *testing.T, consumer *LinkUpdateConsumer) {
	t.Helper()

	if err := consumer.Close(); err != nil {
		t.Logf("close kafka consumer: %v", err)
	}
}

func produceRawMessage(
	t *testing.T,
	bootstrapServers string,
	topic string,
	key []byte,
	payload []byte,
) {
	t.Helper()

	producer, err := confluent.NewProducer(&confluent.ConfigMap{
		"bootstrap.servers": bootstrapServers,
		"client.id":         integrationTestClientIDScrapper,
	})
	if err != nil {
		t.Fatalf("create kafka producer: %v", err)
	}

	defer producer.Close()

	deliveryChan := make(chan confluent.Event, integrationTestDeliveryChanBuffer)

	if err := producer.Produce(&confluent.Message{
		TopicPartition: confluent.TopicPartition{
			Topic:     &topic,
			Partition: confluent.PartitionAny,
		},
		Key:   key,
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

	case <-time.After(integrationTestWaitProduceDelivery):
		t.Fatal("timeout waiting for message delivery")
	}
}

func readDLQMessage(
	t *testing.T,
	bootstrapServers string,
	dlqTopic string,
) DeadLetterMessage {
	t.Helper()

	consumer, err := confluent.NewConsumer(&confluent.ConfigMap{
		"bootstrap.servers": bootstrapServers,
		"group.id":          fmt.Sprintf("dlq-reader-%d", time.Now().UnixNano()),
		"auto.offset.reset": kafkaAutoOffsetResetEarliest,
	})
	if err != nil {
		t.Fatalf("create dlq consumer: %v", err)
	}

	defer func() {
		if err := consumer.Close(); err != nil {
			t.Logf("close dlq consumer: %v", err)
		}
	}()

	if err := consumer.SubscribeTopics([]string{dlqTopic}, nil); err != nil {
		t.Fatalf("subscribe to dlq topic: %v", err)
	}

	message, err := consumer.ReadMessage(integrationTestWaitReadDLQMessage)
	if err != nil {
		t.Fatalf("read dlq message: %v", err)
	}

	var dlq DeadLetterMessage
	if err := json.Unmarshal(message.Value, &dlq); err != nil {
		t.Fatalf("unmarshal dlq message: %v", err)
	}

	return dlq
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()

	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal value: %v", err)
	}

	return payload
}
