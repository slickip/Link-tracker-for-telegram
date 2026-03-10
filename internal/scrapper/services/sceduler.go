package services

import (
	"context"
	"time"

	"github.com/go-co-op/gocron/v2"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/clients"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/parsers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/repositories"
)

type Scheduler struct {
	repo repositories.LinkRepository

	githubClient *clients.GitHubClient
	soClient     *clients.StackOverflowClient

	botClient clients.BotClient

	log domain.Logger
}

func (s *Scheduler) Start() {

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		s.log.Error("scheduler init error", "error", err)
		return
	}

	_, err = scheduler.NewJob(
		gocron.DurationJob(1*time.Minute),
		gocron.NewTask(s.CheckLinks),
	)

	if err != nil {
		s.log.Error("scheduler job error", "error", err)
		return
	}

	s.log.Info("scheduler started", "interval", "1m")

	scheduler.Start()
}

func (s *Scheduler) CheckLinks() {

	s.log.Debug("checking links")

	links, err := s.repo.GetAllTrackedLinks()
	if err != nil {
		s.log.Error("failed to get tracked links", "error", err)
		return
	}

	for _, link := range links {

		err := s.processLink(link)
		if err != nil {
			s.log.Warn("link check failed", "url", link.URL, "error", err)
		}
	}
}

func (s *Scheduler) processLink(link domain.Link) error {

	parsed, err := parsers.ParseLink(link.URL)
	if err != nil {
		return err
	}

	switch parsed.Source {

	case "github":

		_, err = s.githubClient.GetRepoUpdatedAt(
			context.Background(),
			parsed.GithubOwner,
			parsed.GithubRepo,
		)

	case "stackoverflow":

		_, err = s.soClient.GetQuestionUpdatedAt(
			context.Background(),
			parsed.StackOverflowQuestionID,
		)

	default:
		return nil
	}

	if err != nil {
		return err
	}

	subscribers, err := s.repo.FindSubscribers(link.URL)
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

	return s.botClient.SendUpdate(context.Background(), update)
}

func NewScheduler(
	repo repositories.LinkRepository,
	githubClient *clients.GitHubClient,
	soClient *clients.StackOverflowClient,
	botClient clients.BotClient,
	log domain.Logger,
) *Scheduler {

	return &Scheduler{
		repo:         repo,
		githubClient: githubClient,
		soClient:     soClient,
		botClient:    botClient,
		log:          log,
	}
}

