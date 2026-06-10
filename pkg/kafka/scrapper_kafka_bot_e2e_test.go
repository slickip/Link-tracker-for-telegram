//go:build integration
// +build integration

package kafka

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/slickip/link-tracker/pkg/api"
)

func TestScraperKafkaProducerToBotConsumerEndToEnd(t *testing.T) {
	ctx := context.Background()

	mainTopic, dlqTopic := uniqueTopicPair(t)
	bootstrap := prepareKafkaTopics(t, ctx, mainTopic, dlqTopic)

	producer, err := NewConfluentLinkUpdateProducer(LinkUpdateProducerConfig{
		BootstrapServers:    bootstrap,
		Topic:               mainTopic,
		ClientID:            integrationTestClientIDE2EScrapper,
		SerializationFormat: SerializationFormatJSON,
	})
	if err != nil {
		t.Fatalf("create producer: %v", err)
	}
	defer producer.Close()

	fakeBot := newFakeMessageSender()

	consumer, err := NewLinkUpdateConsumer(
		LinkUpdateConsumerConfig{
			BootstrapServers:    bootstrap,
			Topic:               mainTopic,
			DLQTopic:            dlqTopic,
			ConsumerGroup:       fmt.Sprintf("e2e-bot-%d", time.Now().UnixNano()),
			ClientID:            integrationTestClientIDE2EBot,
			MaxRetries:          integrationTestDefaultMaxRetries,
			SerializationFormat: SerializationFormatJSON,
		},
		fakeBot,
	)
	if err != nil {
		t.Fatalf("create consumer: %v", err)
	}
	defer closeConsumer(t, consumer)

	consumerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := startConsumer(consumerCtx, consumer)

	expected := api.LinkUpdate{
		ID:          777,
		URL:         "https://example.com/page",
		TgChatIDs:   []int64{4242},
		Type:        "github_issue",
		Title:       "t",
		Username:    "u",
		CreatedAt:   time.Now().UTC(),
		Preview:     "p",
		Description: "Страница изменилась",
	}

	if err := producer.Produce(ctx, expected); err != nil {
		t.Fatalf("produce: %v", err)
	}

	select {
	case message := <-fakeBot.messages:
		if message.chatID != expected.TgChatIDs[0] {
			t.Fatalf("chatID: want %d got %d", expected.TgChatIDs[0], message.chatID)
		}
		if message.text != expected.SubscriberNotificationBody() {
			t.Fatalf("text: want %q got %q", expected.SubscriberNotificationBody(), message.text)
		}
	case err := <-errCh:
		t.Fatalf("consumer: %v", err)
	case <-time.After(integrationTestWaitE2ENotification):
		t.Fatal("timeout waiting for notification")
	}
}
