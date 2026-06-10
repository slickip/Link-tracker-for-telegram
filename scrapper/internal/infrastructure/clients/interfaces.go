package clients

import (
	"context"
	"time"

	"github.com/slickip/link-tracker/scrapper/internal/domain"
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
