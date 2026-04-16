package commands_test

import (
	"context"
	"strings"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/commands"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/dispatch"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"
)

type fakeTrackSessionRepo struct {
	sessions map[int64]domain.TrackSession
}

func newFakeTrackSessionRepo() *fakeTrackSessionRepo {
	return &fakeTrackSessionRepo{
		sessions: make(map[int64]domain.TrackSession),
	}
}

func (r *fakeTrackSessionRepo) Get(ctx context.Context, chatID int64) (domain.TrackSession, bool, error) {
	session, ok := r.sessions[chatID]
	return session, ok, nil
}

func (r *fakeTrackSessionRepo) Set(ctx context.Context, chatID int64, session domain.TrackSession) error {
	r.sessions[chatID] = session
	return nil
}

func (r *fakeTrackSessionRepo) Reset(ctx context.Context, chatID int64) error {
	delete(r.sessions, chatID)
	return nil
}

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
	repo := newFakeTrackSessionRepo()
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

func TestStartCommand(t *testing.T) {
	d := setupDispatcher()

	resp, err := d.Dispatch(context.Background(), 1, "/start")
	if err != nil {
		t.Fatal(err)
	}

	if resp == "" {
		t.Fatal("empty response")
	}
}

func TestHelpCommand(t *testing.T) {
	d := setupDispatcher()

	resp, err := d.Dispatch(context.Background(), 1, "/help")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(resp, "/track") {
		t.Fatal("help text not correct")
	}
}

func TestUnknownCommand(t *testing.T) {
	d := setupDispatcher()

	resp, err := d.Dispatch(context.Background(), 1, "/abracadabra")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(resp, "Неизвестная команда") {
		t.Fatal("wrong unknown response")
	}
}

func TestCancelCommand(t *testing.T) {
	d := setupDispatcher()

	resp, err := d.Dispatch(context.Background(), 1, "/cancel")
	if err != nil {
		t.Fatal(err)
	}

	if resp != "Операция отменена" && resp != "Operation cancelled" {
		t.Fatal("cancel failed")
	}
}

func TestTrackCommand(t *testing.T) {
	d := setupDispatcher()

	resp, err := d.Dispatch(context.Background(), 1, "/track")
	if err != nil {
		t.Fatal(err)
	}

	if resp == "" {
		t.Fatal("track should start flow")
	}
}

func TestListCommand_Empty(t *testing.T) {
	d := setupDispatcher()

	resp, err := d.Dispatch(context.Background(), 1, "/list")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(resp, "пуст") {
		t.Fatal("expected empty list message")
	}
}

func TestUntrackCommand_Invalid(t *testing.T) {
	d := setupDispatcher()

	resp, err := d.Dispatch(context.Background(), 1, "/untrack")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(resp, "Использование") {
		t.Fatal("expected usage message")
	}
}
