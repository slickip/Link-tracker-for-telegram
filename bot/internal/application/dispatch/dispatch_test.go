package dispatch_test

import (
	"context"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/commands"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/dispatch"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
)

type fakeTrackSessionRepo struct {
	sessions map[int64]domain.TrackSession
	getErr   error
	setErr   error
	resetErr error
}

func newFakeTrackSessionRepo() *fakeTrackSessionRepo {
	return &fakeTrackSessionRepo{
		sessions: make(map[int64]domain.TrackSession),
	}
}

func (r *fakeTrackSessionRepo) Get(ctx context.Context, chatID int64) (domain.TrackSession, bool, error) {
	if r.getErr != nil {
		return domain.TrackSession{}, false, r.getErr
	}

	session, ok := r.sessions[chatID]
	return session, ok, nil
}

func (r *fakeTrackSessionRepo) Set(ctx context.Context, chatID int64, session domain.TrackSession) error {
	if r.setErr != nil {
		return r.setErr
	}

	r.sessions[chatID] = session
	return nil
}

func (r *fakeTrackSessionRepo) Reset(ctx context.Context, chatID int64) error {
	if r.resetErr != nil {
		return r.resetErr
	}

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

func TestStartCommand_Positive(t *testing.T) {
	dispatcher := setupDispatcher()

	response, err := dispatcher.Dispatch(context.Background(), 123, "/start")
	if err != nil {
		t.Fatalf("%v: %v", pkg.ErrUnexpectedError, err)
	}

	if response == "" {
		t.Fatal(pkg.ErrEmptyStartResponse)
	}
}

func TestHelpCommand_Positive(t *testing.T) {
	dispatcher := setupDispatcher()

	response, err := dispatcher.Dispatch(context.Background(), 123, "/help")
	if err != nil {
		t.Fatalf("%v: %v", pkg.ErrUnexpectedError, err)
	}

	if response == "" {
		t.Fatal(pkg.ErrEmptyHelpResponse)
	}
}

func TestUnknownCommand_Negative(t *testing.T) {
	dispatcher := setupDispatcher()

	response, err := dispatcher.Dispatch(context.Background(), 123, "/unknown_command")
	if err != nil {
		t.Fatalf("%v: %v", pkg.ErrUnexpectedError, err)
	}

	if response == "" {
		t.Fatal(pkg.ErrEmptyUnknownResponse)
	}
}
