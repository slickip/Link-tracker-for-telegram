package services

import (
	"context"
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/clients"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/parsers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/repositories"
)

type Scheduler struct {
	repo repositories.LinkRepository

	githubClient *clients.GitHubClient
	soClient     *clients.StackOverflowClient

	botClient clients.BotClient
}

func (s *Scheduler) Start() {

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Println("scheduler init error:", err)
		return
	}

	_, err = scheduler.NewJob(
		gocron.DurationJob(1*time.Minute),
		gocron.NewTask(s.CheckLinks),
	)

	if err != nil {
		log.Println("scheduler job error:", err)
		return
	}

	scheduler.Start()
}

func (s *Scheduler) CheckLinks() {

	links, err := s.repo.GetAllTrackedLinks()
	if err != nil {
		log.Println("failed to get tracked links:", err)
		return
	}

	for _, link := range links {

		err := s.processLink(link)
		if err != nil {
			log.Println("check failed:", err)
		}
	}
}

func (s *Scheduler) processLink(link domain.Link) error {

	parsed, err := parsers.ParseLink(link.URL)
	if err != nil {
		return err
	}

	var updatedAt time.Time

	switch parsed.Source {

	case "github":

		updatedAt, err = s.githubClient.GetRepoUpdatedAt(
			context.Background(),
			parsed.GithubOwner,
			parsed.GithubRepo,
		)

	case "stackoverflow":

		updatedAt, err = s.soClient.GetQuestionUpdatedAt(
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
