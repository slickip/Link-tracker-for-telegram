package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/clients"
	scrappermetrics "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/metrics"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/parsers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/repositories"
)

const (
	defaultSchedulerInterval    = 30 * time.Second
	defaultSchedulerBatchSize   = 100
	defaultSchedulerWorkerCount = 1

	updateDescriptionTimeFormat = "2006-01-02 15:04:05"

	processingErrorType     = "processing_error"
	processingErrorTitle    = "Ошибка обработки ссылки"
	processingErrorUsername = "scrapper"
	decimalBase             = 10
)

type Scheduler struct {
	repo repositories.TrackingRepository

	githubClient *clients.GitHubClient
	soClient     *clients.StackOverflowClient
	botClient    clients.BotClient
	log          *logger.Slog

	interval         time.Duration
	batchSize        int
	workerCount      int
	failureMu        sync.Mutex
	reportedFailures map[int64]string

	outboxRepo    repositories.OutboxRepository
	outboxTopic   string
	outboxEnabled bool
}

func NewScheduler(
	repo repositories.TrackingRepository,
	githubClient *clients.GitHubClient,
	soClient *clients.StackOverflowClient,
	botClient clients.BotClient,
	log *logger.Slog,
	interval time.Duration,
	batchSize int,
	workerCount int,
) *Scheduler {
	if interval <= 0 {
		interval = defaultSchedulerInterval
	}
	if batchSize <= 0 {
		batchSize = defaultSchedulerBatchSize
	}
	if workerCount <= 0 {
		workerCount = defaultSchedulerWorkerCount
	}

	return &Scheduler{
		repo:             repo,
		githubClient:     githubClient,
		soClient:         soClient,
		botClient:        botClient,
		log:              log,
		interval:         interval,
		batchSize:        batchSize,
		workerCount:      workerCount,
		reportedFailures: make(map[int64]string),
	}
}

func (s *Scheduler) Start() {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		s.log.Error("scheduler init error", "error", err)
		return
	}

	_, err = scheduler.NewJob(
		gocron.DurationJob(s.interval),
		gocron.NewTask(s.CheckLinks),
	)
	if err != nil {
		s.log.Error("scheduler job error", "error", err)
		return
	}

	s.log.Info("scheduler started", "interval", s.interval.String())

	scheduler.Start()
}

func (s *Scheduler) CheckLinks() {
	ctx := context.Background()

	s.log.Info("checking links")

	var (
		offset     = 0
		totalLinks = 0
	)

	for {
		links, err := s.repo.GetTrackedLinksBatch(ctx, s.batchSize, offset)
		if err != nil {
			s.log.Error("failed to get tracked links batch", "error", err, "offset", offset, "limit", s.batchSize)
			return
		}
		if len(links) == 0 {
			break
		}

		totalLinks += len(links)
		s.processBatch(ctx, links)

		offset += len(links)
	}

	s.log.Info("links check finished", "count", totalLinks)
}

func (s *Scheduler) processBatch(ctx context.Context, links []domain.Link) {
	if len(links) == 0 {
		return
	}

	workers := s.workerCount
	if workers > len(links) {
		workers = len(links)
	}
	if workers <= 0 {
		workers = defaultSchedulerWorkerCount
	}

	chunkSize := (len(links) + workers - 1) / workers

	type failedLink struct {
		link domain.Link
		err  error
	}

	var (
		wg     sync.WaitGroup
		failMu sync.Mutex
		failed = make([]failedLink, 0)
	)

	for i := 0; i < workers; i++ {
		start := i * chunkSize
		if start >= len(links) {
			break
		}
		end := start + chunkSize
		if end > len(links) {
			end = len(links)
		}

		part := links[start:end]
		wg.Add(1)
		go func(part []domain.Link) {
			defer wg.Done()

			for _, link := range part {
				if err := s.processLink(ctx, link); err != nil {
					s.log.Warn("link check failed", "id", link.ID, "url", link.URL, "error", err)

					failMu.Lock()
					failed = append(failed, failedLink{link: link, err: err})
					failMu.Unlock()

					continue
				}

				s.clearReportedFailure(link.ID)
			}
		}(part)
	}

	wg.Wait()
	for _, f := range failed {
		s.reportLinkFailure(ctx, f.link, f.err)
	}
}

func (s *Scheduler) reportLinkFailure(ctx context.Context, link domain.Link, err error) {
	preview := MakeErrorPreview(err)

	if !s.shouldReportFailure(link.ID, preview) {
		return
	}

	subscribers, subErr := s.repo.FindSubscribers(ctx, link.ID)
	if subErr != nil {
		s.log.Warn("failed to find subscribers for failed link", "id", link.ID, "error", subErr)
		s.clearReportedFailure(link.ID)
		return
	}

	if len(subscribers) == 0 {
		return
	}

	dto := api.LinkUpdate{
		ID:        link.ID,
		URL:       link.URL,
		TgChatIDs: subscribers,
		Type:      processingErrorType,
		Title:     processingErrorTitle,
		Username:  processingErrorUsername,
		CreatedAt: time.Now().UTC(),
		Preview:   preview,
		Description: fmt.Sprintf(
			"Не удалось обработать ссылку: %s\nПричина: %s",
			link.URL,
			preview,
		),
	}

	if sendErr := s.botClient.SendUpdate(ctx, dto); sendErr != nil {
		s.log.Warn("failed to send link failure update", "id", link.ID, "error", sendErr)
		s.clearReportedFailure(link.ID)
	}
}

func (s *Scheduler) processLink(ctx context.Context, link domain.Link) error {
	parsed, err := parsers.ParseLink(link.URL)
	if err != nil {
		return err
	}

	started := time.Now()
	source := string(parsed.Source)

	defer func() {
		scrappermetrics.RequestDurationMs.
			WithLabelValues("external_source", source).
			Observe(float64(time.Since(started).Milliseconds()))
	}()

	updates, newUpdatedAt, err := s.fetchUpdates(ctx, link, parsed)
	if err != nil {
		return err
	}

	if link.LastUpdatedAt.IsZero() {
		if newUpdatedAt.IsZero() {
			newUpdatedAt = time.Now().UTC()
		}

		return s.repo.UpdateLastUpdated(ctx, link.ID, newUpdatedAt)
	}

	if len(updates) == 0 {
		return nil
	}

	if !s.outboxEnabled && newUpdatedAt.After(link.LastUpdatedAt) {
		if err := s.repo.UpdateLastUpdated(ctx, link.ID, newUpdatedAt); err != nil {
			return err
		}
	}

	subscribers, err := s.repo.FindSubscribers(ctx, link.ID)
	if err != nil {
		return err
	}

	if len(subscribers) == 0 {
		if s.outboxEnabled && newUpdatedAt.After(link.LastUpdatedAt) {
			return s.repo.UpdateLastUpdated(ctx, link.ID, newUpdatedAt)
		}

		return nil
	}

	outboxMessages := make([]domain.OutboxMessage, 0, len(updates))

	for _, update := range updates {
		dto := api.LinkUpdate{
			ID:        update.LinkID,
			URL:       update.URL,
			TgChatIDs: subscribers,

			Type:      string(update.Type),
			Title:     update.Title,
			Username:  update.Username,
			CreatedAt: update.CreatedAt,
			Preview:   update.Preview,

			Description: formatUpdateDescription(update),
		}

		if s.outboxEnabled {
			message, err := makeOutboxMessage(s.outboxTopic, dto)
			if err != nil {
				return err
			}

			outboxMessages = append(outboxMessages, message)
			continue
		}

		if err := s.botClient.SendUpdate(ctx, dto); err != nil {
			return err
		}
	}

	if s.outboxEnabled {
		if newUpdatedAt.After(link.LastUpdatedAt) {
			return s.outboxRepo.SaveLinkUpdateOutbox(
				ctx,
				link.ID,
				newUpdatedAt,
				outboxMessages,
			)
		}

		return s.outboxRepo.SaveOutboxMessages(ctx, outboxMessages)
	}

	return nil
}

func (s *Scheduler) fetchUpdates(
	ctx context.Context,
	link domain.Link,
	parsed parsers.ParsedLink,
) ([]domain.LinkUpdate, time.Time, error) {
	switch parsed.Source {
	case parsers.SourceGitHub:
		return s.githubClient.GetNewIssuesAndPullRequests(
			ctx,
			parsed.GithubOwner,
			parsed.GithubRepo,
			link.URL,
			link.ID,
			link.LastUpdatedAt,
		)

	case parsers.SourceStackOverflow:
		return s.soClient.GetNewAnswersAndComments(
			ctx,
			parsed.StackOverflowQuestionID,
			link.URL,
			link.ID,
			link.LastUpdatedAt,
		)

	default:
		return nil, time.Time{}, nil
	}
}

func formatUpdateDescription(update domain.LinkUpdate) string {
	return fmt.Sprintf(
		"Обнаружено обновление по ссылке: %s\n\nТип: %s\nТема: %s\nАвтор: %s\nСоздано: %s\n\n%s",
		update.URL,
		update.Type,
		update.Title,
		update.Username,
		update.CreatedAt.Format(updateDescriptionTimeFormat),
		update.Preview,
	)
}

func MakeErrorPreview(err error) string {
	if err == nil {
		return ""
	}
	return clients.MakePreview(err.Error())
}

func (s *Scheduler) shouldReportFailure(linkID int64, preview string) bool {
	s.failureMu.Lock()
	defer s.failureMu.Unlock()

	oldPreview, exists := s.reportedFailures[linkID]
	if exists && oldPreview == preview {
		return false
	}

	s.reportedFailures[linkID] = preview
	return true
}

func (s *Scheduler) clearReportedFailure(linkID int64) {
	s.failureMu.Lock()
	defer s.failureMu.Unlock()

	delete(s.reportedFailures, linkID)
}

func (s *Scheduler) EnableOutbox(
	outboxRepo repositories.OutboxRepository,
	topic string,
) {
	s.outboxRepo = outboxRepo
	s.outboxTopic = topic
	s.outboxEnabled = outboxRepo != nil && topic != ""
}

func makeOutboxMessage(
	topic string,
	update api.LinkUpdate,
) (domain.OutboxMessage, error) {
	payload, err := json.Marshal(update)
	if err != nil {
		return domain.OutboxMessage{}, err
	}

	now := time.Now().UTC()

	return domain.OutboxMessage{
		Topic:      topic,
		MessageKey: strconv.FormatInt(update.ID, decimalBase),
		Payload:    payload,
		Status:     domain.OutboxStatusPending,
		Attempts:   0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}
