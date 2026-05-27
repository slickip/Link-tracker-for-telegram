package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	h "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/helpers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
)

const defaultGitHubBaseURL = "https://api.github.com"

type GitHubClient struct {
	client      *http.Client
	baseURL     string
	token       string
	timeout     time.Duration
	perPage     int
	retryConfig h.HTTPRetryConfig
}

type GitHubClientConfig struct {
	BaseURL string
	Token   string
	Timeout time.Duration
	PerPage int
	Retry   h.HTTPRetryConfig
}

type githubIssueResponse struct {
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	HTMLURL   string    `json:"html_url"`

	User struct {
		Login string `json:"login"`
	} `json:"user"`

	PullRequest *struct{} `json:"pull_request,omitempty"`
}

func NewGitHubClient(cfg GitHubClientConfig) *GitHubClient {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultGitHubBaseURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	perPage := cfg.PerPage
	if perPage <= 0 {
		perPage = 100
	}

	return &GitHubClient{
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL:     baseURL,
		token:       cfg.Token,
		timeout:     timeout,
		perPage:     perPage,
		retryConfig: h.NormalizeRetryConfig(cfg.Retry),
	}
}

func (c *GitHubClient) GetNewIssuesAndPullRequests(
	ctx context.Context,
	owner string,
	repo string,
	linkURL string,
	linkID int64,
	since time.Time,
) ([]domain.LinkUpdate, time.Time, error) {
	endpoint, err := url.Parse(fmt.Sprintf(
		"%s/repos/%s/%s/issues",
		c.baseURL,
		url.PathEscape(owner),
		url.PathEscape(repo),
	))
	if err != nil {
		return nil, time.Time{}, err
	}

	query := endpoint.Query()
	query.Set("state", "all")
	query.Set("sort", "created")
	query.Set("direction", "desc")
	query.Set("per_page", fmt.Sprintf("%d", c.perPage))

	if !since.IsZero() {
		query.Set("since", since.UTC().Format(time.RFC3339))
	}

	endpoint.RawQuery = query.Encode()

	var items []githubIssueResponse

	err = h.DoWithHTTPRetry(ctx, c.retryConfig, func() error {
		req, cancel, err := h.NewRequestWithTimeout(ctx, c.timeout, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return err
		}
		defer cancel()

		c.setHeaders(req)

		resp, err := c.client.Do(req)
		if err != nil {
			return err
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			return h.NewHTTPStatusError("github", resp.StatusCode, c.retryConfig)
		}

		items = nil
		if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, time.Time{}, err
	}

	updates := make([]domain.LinkUpdate, 0)
	maxCreatedAt := since

	for _, item := range items {
		if !since.IsZero() && !item.CreatedAt.After(since) {
			continue
		}

		updateType := domain.UpdateTypeGitHubIssue
		if item.PullRequest != nil {
			updateType = domain.UpdateTypeGitHubPullRequest
		}

		updates = append(updates, domain.LinkUpdate{
			LinkID:    linkID,
			URL:       linkURL,
			Type:      updateType,
			Title:     item.Title,
			Username:  item.User.Login,
			CreatedAt: item.CreatedAt,
			Preview:   MakePreview(item.Body),
		})

		if item.CreatedAt.After(maxCreatedAt) {
			maxCreatedAt = item.CreatedAt
		}
	}

	return updates, maxCreatedAt, nil
}

func (c *GitHubClient) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "link-tracker")

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}
