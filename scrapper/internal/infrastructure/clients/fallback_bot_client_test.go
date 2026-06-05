package clients

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type fakeBotClient struct {
	err        error
	calls      int
	lastUpdate api.LinkUpdate
}

func (c *fakeBotClient) SendUpdate(_ context.Context, update api.LinkUpdate) error {
	c.calls++
	c.lastUpdate = update

	return c.err
}

func TestFallbackBotClientPrimaryHTTPFailedUsesKafkaFallback(t *testing.T) {
	httpErr := errors.New("http transport unavailable")

	httpClient := &fakeBotClient{
		err: httpErr,
	}
	kafkaClient := &fakeBotClient{}

	client := NewFallbackBotClient(httpClient, kafkaClient, nil)

	update := api.LinkUpdate{
		ID:        1,
		URL:       "https://github.com/owner/repo",
		TgChatIDs: []int64{123},
	}

	err := client.SendUpdate(context.Background(), update)
	if err != nil {
		t.Fatalf("expected nil error because kafka fallback succeeded, got: %v", err)
	}

	if httpClient.calls != 1 {
		t.Fatalf("expected http client to be called once, got: %d", httpClient.calls)
	}

	if kafkaClient.calls != 1 {
		t.Fatalf("expected kafka fallback client to be called once, got: %d", kafkaClient.calls)
	}

	if !reflect.DeepEqual(kafkaClient.lastUpdate, update) {
		t.Fatalf("expected kafka fallback to receive original update")
	}
}

func TestFallbackBotClientReturnsBothErrorsWhenHTTPAndKafkaFailed(t *testing.T) {
	httpErr := errors.New("http transport unavailable")
	kafkaErr := errors.New("kafka transport unavailable")

	httpClient := &fakeBotClient{
		err: httpErr,
	}
	kafkaClient := &fakeBotClient{
		err: kafkaErr,
	}

	client := NewFallbackBotClient(httpClient, kafkaClient, nil)

	update := api.LinkUpdate{
		ID:        1,
		URL:       "https://github.com/owner/repo",
		TgChatIDs: []int64{123},
	}

	err := client.SendUpdate(context.Background(), update)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, httpErr) {
		t.Fatalf("expected error to contain http error, got: %v", err)
	}

	if !errors.Is(err, kafkaErr) {
		t.Fatalf("expected error to contain kafka error, got: %v", err)
	}

	if httpClient.calls != 1 {
		t.Fatalf("expected http client to be called once, got: %d", httpClient.calls)
	}

	if kafkaClient.calls != 1 {
		t.Fatalf("expected kafka fallback client to be called once, got: %d", kafkaClient.calls)
	}
}

func TestFallbackBotClientDoesNotUseKafkaWhenHTTPSucceeded(t *testing.T) {
	httpClient := &fakeBotClient{}
	kafkaClient := &fakeBotClient{}

	client := NewFallbackBotClient(httpClient, kafkaClient, nil)

	update := api.LinkUpdate{
		ID:        1,
		URL:       "https://github.com/owner/repo",
		TgChatIDs: []int64{123},
	}

	err := client.SendUpdate(context.Background(), update)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	if httpClient.calls != 1 {
		t.Fatalf("expected http client to be called once, got: %d", httpClient.calls)
	}

	if kafkaClient.calls != 0 {
		t.Fatalf("expected kafka fallback not to be called, got: %d", kafkaClient.calls)
	}
}
