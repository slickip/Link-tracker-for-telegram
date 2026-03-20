package dispatch_test

import (
	"context"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/commands"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/dispatch"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/repositories"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
)

type mockScrapperClient struct {
	links []domain.Link
}

func (m *mockScrapperClient) RegisterChat(ctx context.Context, chatID int64) error {
	return nil
}

func (m *mockScrapperClient) DeleteChat(ctx context.Context, chatID int64) error {
	return nil
}

func (m *mockScrapperClient) AddLink(ctx context.Context, chatID int64, url string, tags []string) error {
	m.links = append(m.links, domain.Link{
		URL:  url,
		Tags: tags,
	})
	return nil
}

func (m *mockScrapperClient) RemoveLink(ctx context.Context, chatID int64, url string) error {
	return nil
}

func (m *mockScrapperClient) ListLinks(ctx context.Context, chatID int64) ([]domain.Link, error) {
	return m.links, nil
}

func setupDispatcher() *dispatch.Dispatcher {
	repo := repositories.NewInMemoryTrackSessionRepository()

	mockClient := &mockScrapperClient{}

	trackService := services.NewTrackService(mockClient, repo)

	cmds := []domain.Command{
		commands.NewStartCommand(mockClient),
		commands.NewHelpCommand(),
		commands.NewTrackCommand(trackService),
		commands.NewListCommand(mockClient),
		commands.NewUntrackCommand(mockClient),
		commands.NewCancelCommand(),
		commands.NewUnknownCommand(),
	}

	return dispatch.NewDispatcher(cmds, trackService, repo)
}

func TestStartCommand_Positive(t *testing.T) {
	dispatcher := setupDispatcher()

	response, err := dispatcher.Dispatch(123, "/start")

	if err != nil {
		t.Fatalf("%v: %v", pkg.ErrUnexpectedError, err)
	}

	if response == "" {
		t.Fatal(pkg.ErrEmptyStartResponse)
	}
}

func TestHelpCommand_Positive(t *testing.T) {
	dispatcher := setupDispatcher()

	response, err := dispatcher.Dispatch(123, "/help")

	if err != nil {
		t.Fatalf("%v: %v", pkg.ErrUnexpectedError, err)
	}

	if response == "" {
		t.Fatal(pkg.ErrEmptyHelpResponse)
	}
}

func TestUnknownCommand_Negative(t *testing.T) {
	dispatcher := setupDispatcher()

	response, err := dispatcher.Dispatch(123, "/unknown_command")

	if err != nil {
		t.Fatalf("%v: %v", pkg.ErrUnexpectedError, err)
	}

	if response == "" {
		t.Fatal(pkg.ErrEmptyUnknownResponse)
	}
}
