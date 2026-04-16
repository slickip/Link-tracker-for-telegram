package services

import (
	"context"
	"time"

	"github.com/go-co-op/gocron/v2"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain/repositories"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/clients"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/parsers"
)

const seconds = 30

type Scheduler struct {
	repo repositories.TrackingRepository

	githubClient *clients.GitHubClient
	soClient     *clients.StackOverflowClient
	botClient    clients.BotClient
	log          *logger.Slog
}

func NewScheduler(
	repo repositories.TrackingRepository,
	githubClient *clients.GitHubClient,
	soClient *clients.StackOverflowClient,
	botClient clients.BotClient,
	log *logger.Slog,
) *Scheduler {
	return &Scheduler{
		repo:         repo,
		githubClient: githubClient,
		soClient:     soClient,
		botClient:    botClient,
		log:          log,
	}
}

func (s *Scheduler) Start() {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		s.log.Error("scheduler init error", "error", err)
		return
	}

	_, err = scheduler.NewJob(
		gocron.DurationJob(seconds*time.Second),
		gocron.NewTask(s.CheckLinks),
	)
	if err != nil {
		s.log.Error("scheduler job error", "error", err)
		return
	}

	s.log.Info("scheduler started", "interval", "30s")
	scheduler.Start()
}

func (s *Scheduler) CheckLinks() {
	ctx := context.Background()

	s.log.Info("checking links")

	links, err := s.repo.GetAllTrackedLinks(ctx)
	if err != nil {
		s.log.Error("failed to get tracked links", "error", err)
		return
	}

	s.log.Info("links found", "count", len(links))

	for _, link := range links {
		if err := s.processLink(ctx, link); err != nil {
			s.log.Warn("link check failed", "url", link.URL, "error", err)
		}
	}
}

func (s *Scheduler) processLink(ctx context.Context, link domain.Link) error {
	parsed, err := parsers.ParseLink(link.URL)
	if err != nil {
		return err
	}

	var newUpdatedAt time.Time

	switch parsed.Source {
	case "github":
		newUpdatedAt, err = s.githubClient.GetRepoUpdatedAt(
			ctx,
			parsed.GithubOwner,
			parsed.GithubRepo,
		)

	case "stackoverflow":
		newUpdatedAt, err = s.soClient.GetQuestionUpdatedAt(
			ctx,
			parsed.StackOverflowQuestionID,
		)

	default:
		return nil
	}

	if err != nil {
		return err
	}

	if link.LastUpdatedAt.IsZero() {
		return s.repo.UpdateLastUpdated(ctx, link.URL, newUpdatedAt)
	}

	if !newUpdatedAt.After(link.LastUpdatedAt) {
		return nil
	}

	if err = s.repo.UpdateLastUpdated(ctx, link.URL, newUpdatedAt); err != nil {
		return err
	}

	subscribers, err := s.repo.FindSubscribers(ctx, link.URL)
	if err != nil {
		return err
	}

	if len(subscribers) == 0 {
		return nil
	}

	update := domain.LinkUpdate{
		URL:       link.URL,
		TgChatIDs: subscribers,
	}

	return s.botClient.SendUpdate(ctx, update)
}
