package domain

import "time"

type UpdateType string

const (
	UpdateTypeGitHubIssue          UpdateType = "github_issue"
	UpdateTypeGitHubPullRequest    UpdateType = "github_pull_request"
	UpdateTypeStackOverflowAnswer  UpdateType = "stackoverflow_answer"
	UpdateTypeStackOverflowComment UpdateType = "stackoverflow_comment"
)

type LinkUpdate struct {
	LinkID    int64
	URL       string
	Type      UpdateType
	Title     string
	Username  string
	CreatedAt time.Time
	Preview   string
}
