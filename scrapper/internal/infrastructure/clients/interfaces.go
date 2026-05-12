package clients

import (
	"context"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
)

type GitHubAPI interface {
	GetNewIssuesAndPullRequests(
		ctx context.Context,
		owner, repo string,
		since time.Time,
	) ([]domain.LinkUpdate, error)
}

type StackOverflowAPI interface {
	GetNewAnswersAndComments(
		ctx context.Context,
		questionID int64,
		since time.Time,
	) ([]domain.LinkUpdate, error)
}
