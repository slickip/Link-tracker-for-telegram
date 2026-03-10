package domain

import "time"

type LinkSource string

const (
	SourceGitHub        LinkSource = "github"
	SourceStackOverflow LinkSource = "stackoverflow"
)

type Link struct {
	URL string

	Tags []string

	Source LinkSource

	LastUpdatedAt time.Time
	LastCheckedAt time.Time
}

type Subscription struct {
	ChatID  int64
	LinkURL string
}

type ParsedLink struct {
	RawURL string

	Source LinkSource

	GithubOwner string
	GithubRepo  string

	StackOverflowQuestionID int64
}
