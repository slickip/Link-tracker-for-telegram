package dispatch_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/clients"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/commands"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/dispatch"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"
)

type trackAndListMockScrapperClient struct {
	registerChatFn func(ctx context.Context, chatID int64) error
	listLinksFn    func(ctx context.Context, chatID int64) ([]domain.Link, error)
	addLinkFn      func(ctx context.Context, chatID int64, url string, tags []string) error
	removeLinkFn   func(ctx context.Context, chatID int64, url string) error
	removeByTagFn  func(ctx context.Context, chatID int64, tag string) (int64, error)
}

func (m *trackAndListMockScrapperClient) RegisterChat(ctx context.Context, chatID int64) error {
	if m.registerChatFn != nil {
		return m.registerChatFn(ctx, chatID)
	}
	return nil
}

func (m *trackAndListMockScrapperClient) DeleteChat(ctx context.Context, chatID int64) error {
	return nil
}

func (m *trackAndListMockScrapperClient) AddLink(ctx context.Context, chatID int64, url string, tags []string) error {
	if m.addLinkFn == nil {
		return nil
	}
	return m.addLinkFn(ctx, chatID, url, tags)
}

func (m *trackAndListMockScrapperClient) RemoveLink(ctx context.Context, chatID int64, url string) error {
	if m.removeLinkFn != nil {
		return m.removeLinkFn(ctx, chatID, url)
	}
	return nil
}

func (m *trackAndListMockScrapperClient) RemoveLinksByTag(ctx context.Context, chatID int64, tag string) (int64, error) {
	if m.removeByTagFn != nil {
		return m.removeByTagFn(ctx, chatID, tag)
	}
	return 0, nil
}

func (m *trackAndListMockScrapperClient) ListLinks(ctx context.Context, chatID int64) ([]domain.Link, error) {
	if m.listLinksFn == nil {
		return nil, nil
	}
	return m.listLinksFn(ctx, chatID)
}

func setupTrackAndListDispatcher(
	repo *fakeTrackSessionRepo,
	client clients.ScrapperClient,
) *dispatch.Dispatcher {
	trackService := services.NewTrackService(client, repo)

	cmds := []domain.Command{
		commands.NewStartCommand(client),
		commands.NewHelpCommand(),
		commands.NewTrackCommand(trackService),
		commands.NewListCommand(client),
		commands.NewUntrackCommand(client),
		commands.NewCancelCommand(),
		commands.NewUnknownCommand(),
	}

	return dispatch.NewDispatcher(cmds, trackService, repo)
}

func TestTrackFlow_Positive(t *testing.T) {
	ctx := context.Background()
	chatID := int64(1)
	repo := newFakeTrackSessionRepo()

	var gotChatID int64
	var gotURL string
	var gotTags []string

	mock := &trackAndListMockScrapperClient{
		addLinkFn: func(ctx context.Context, cID int64, url string, tags []string) error {
			gotChatID = cID
			gotURL = url
			gotTags = tags
			return nil
		},
	}

	d := setupTrackAndListDispatcher(repo, mock)

	resp, err := d.Dispatch(ctx, chatID, "/track")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Отправь мне ссылку, которую хочешь отслеживать" {
		t.Fatalf("unexpected response for /track: %q", resp)
	}

	resp, err = d.Dispatch(ctx, chatID, "https://github.com/user/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(resp, "Отправь мне теги") {
		t.Fatalf("unexpected response for URL step: %q", resp)
	}

	resp, err = d.Dispatch(ctx, chatID, "work, bug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Ссылка добавлена успешно!" {
		t.Fatalf("unexpected response for tags step: %q", resp)
	}

	if gotChatID != chatID {
		t.Fatalf("expected chatID %d, got %d", chatID, gotChatID)
	}
	if gotURL != "https://github.com/user/repo" {
		t.Fatalf("expected url %q, got %q", "https://github.com/user/repo", gotURL)
	}
	if len(gotTags) != 2 || gotTags[0] != "work" || gotTags[1] != "bug" {
		t.Fatalf("expected tags [work bug], got %#v", gotTags)
	}

	_, active, err := repo.Get(ctx, chatID)
	if err != nil {
		t.Fatalf("unexpected repo error: %v", err)
	}
	if active {
		t.Fatalf("expected track session to be reset after successful add")
	}
}

func TestTrackFlow_InvalidLink(t *testing.T) {
	ctx := context.Background()
	chatID := int64(2)
	repo := newFakeTrackSessionRepo()

	addCalled := false
	mock := &trackAndListMockScrapperClient{
		addLinkFn: func(ctx context.Context, cID int64, url string, tags []string) error {
			addCalled = true
			return nil
		},
	}

	d := setupTrackAndListDispatcher(repo, mock)

	if _, err := d.Dispatch(ctx, chatID, "/track"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, err := d.Dispatch(ctx, chatID, "tbank://github.com/user/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Некорректная ссылка. Поддерживаются только ссылки GitHub и StackOverflow" {
		t.Fatalf("unexpected invalid-link response: %q", resp)
	}
	if addCalled {
		t.Fatalf("expected AddLink not to be called on invalid link")
	}

	session, ok, err := repo.Get(ctx, chatID)
	if err != nil {
		t.Fatalf("unexpected repo error: %v", err)
	}
	if !ok {
		t.Fatalf("expected session to still exist after invalid link")
	}
	if session.State != domain.StateWaitingForURL {
		t.Fatalf("expected session state waiting_for_url, got %q", session.State)
	}
	if session.URL != "" {
		t.Fatalf("expected session URL to stay empty, got %q", session.URL)
	}
}

func TestTrackFlow_AlreadySubscribed(t *testing.T) {
	ctx := context.Background()
	chatID := int64(3)
	repo := newFakeTrackSessionRepo()

	mock := &trackAndListMockScrapperClient{
		addLinkFn: func(ctx context.Context, cID int64, url string, tags []string) error {
			return fmt.Errorf("rpc error: code = AlreadyExists desc = already tracked: %w", errors.New("boom"))
		},
	}

	d := setupTrackAndListDispatcher(repo, mock)

	if _, err := d.Dispatch(ctx, chatID, "/track"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := d.Dispatch(ctx, chatID, "https://github.com/user/repo"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, err := d.Dispatch(ctx, chatID, "work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Ссылка уже отслеживается" {
		t.Fatalf("unexpected already-subscribed response: %q", resp)
	}

	_, active, err := repo.Get(ctx, chatID)
	if err != nil {
		t.Fatalf("unexpected repo error: %v", err)
	}
	if active {
		t.Fatalf("expected track session to be reset after already-subscribed")
	}
}

func TestListFlow_ActiveSubscriptions(t *testing.T) {
	ctx := context.Background()
	chatID := int64(4)
	repo := newFakeTrackSessionRepo()

	mock := &trackAndListMockScrapperClient{
		listLinksFn: func(ctx context.Context, cID int64) ([]domain.Link, error) {
			if cID != chatID {
				t.Fatalf("unexpected chatID: %d", cID)
			}
			return []domain.Link{
				{URL: "https://github.com/user/repo", Tags: []string{"go", "work"}},
				{URL: "https://stackoverflow.com/questions/1", Tags: []string{"go"}},
			}, nil
		},
	}

	d := setupTrackAndListDispatcher(repo, mock)

	resp, err := d.Dispatch(ctx, chatID, "/list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(resp, "Отслеживаемые ссылки") {
		t.Fatalf("unexpected list response header: %q", resp)
	}
	if !strings.Contains(resp, "- https://github.com/user/repo") {
		t.Fatalf("expected github url in response: %q", resp)
	}
	if !strings.Contains(resp, "(go, work)") {
		t.Fatalf("expected tags in response: %q", resp)
	}
}

func TestListFlow_NoActiveSubscriptions(t *testing.T) {
	ctx := context.Background()
	chatID := int64(5)
	repo := newFakeTrackSessionRepo()

	mock := &trackAndListMockScrapperClient{
		listLinksFn: func(ctx context.Context, cID int64) ([]domain.Link, error) {
			return []domain.Link{}, nil
		},
	}

	d := setupTrackAndListDispatcher(repo, mock)

	resp, err := d.Dispatch(ctx, chatID, "/list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Список отслеживаемых ссылок пуст" {
		t.Fatalf("unexpected no-subscriptions response: %q", resp)
	}
}

func TestListFlow_FilterByTag(t *testing.T) {
	ctx := context.Background()
	chatID := int64(6)
	repo := newFakeTrackSessionRepo()

	mock := &trackAndListMockScrapperClient{
		listLinksFn: func(ctx context.Context, cID int64) ([]domain.Link, error) {
			return []domain.Link{
				{URL: "https://github.com/user/repo1", Tags: []string{"go"}},
				{URL: "https://github.com/user/repo2", Tags: []string{"work"}},
			}, nil
		},
	}

	d := setupTrackAndListDispatcher(repo, mock)

	resp, err := d.Dispatch(ctx, chatID, "/list go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(resp, "- https://github.com/user/repo1") {
		t.Fatalf("expected filtered url in response: %q", resp)
	}
	if strings.Contains(resp, "- https://github.com/user/repo2") {
		t.Fatalf("did not expect non-matching url in response: %q", resp)
	}
}
