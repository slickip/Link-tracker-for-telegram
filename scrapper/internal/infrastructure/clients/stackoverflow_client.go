package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
)

const defaultStackOverflowBaseURL = "https://api.stackexchange.com/2.3"

type StackOverflowClient struct {
	client  *http.Client
	baseURL string
	site    string
	perPage int
}

type StackOverflowClientConfig struct {
	BaseURL string
	Site    string
	Timeout time.Duration
	PerPage int
}

type stackOverflowQuestionResponse struct {
	Items []struct {
		Title string `json:"title"`
	} `json:"items"`
}

type stackOverflowAnswersResponse struct {
	Items []struct {
		CreationDate int64  `json:"creation_date"`
		Body         string `json:"body"`

		Owner struct {
			DisplayName string `json:"display_name"`
		} `json:"owner"`
	} `json:"items"`
}

type stackOverflowCommentsResponse struct {
	Items []struct {
		CreationDate int64  `json:"creation_date"`
		Body         string `json:"body"`

		Owner struct {
			DisplayName string `json:"display_name"`
		} `json:"owner"`
	} `json:"items"`
}

func NewStackOverflowClient(cfg StackOverflowClientConfig) *StackOverflowClient {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultStackOverflowBaseURL
	}

	site := cfg.Site
	if site == "" {
		site = "stackoverflow"
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	perPage := cfg.PerPage
	if perPage <= 0 {
		perPage = 100
	}

	return &StackOverflowClient{
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
		site:    site,
		perPage: perPage,
	}
}

func (c *StackOverflowClient) GetNewAnswersAndComments(
	ctx context.Context,
	questionID int64,
	linkURL string,
	linkID int64,
	since time.Time,
) ([]domain.LinkUpdate, time.Time, error) {
	title, err := c.getQuestionTitle(ctx, questionID)
	if err != nil {
		return nil, time.Time{}, err
	}

	answers, maxAnswerCreatedAt, err := c.getNewAnswers(ctx, questionID, linkURL, linkID, title, since)
	if err != nil {
		return nil, time.Time{}, err
	}

	comments, maxCommentCreatedAt, err := c.getNewComments(ctx, questionID, linkURL, linkID, title, since)
	if err != nil {
		return nil, time.Time{}, err
	}

	updates := make([]domain.LinkUpdate, 0, len(answers)+len(comments))
	updates = append(updates, answers...)
	updates = append(updates, comments...)

	maxCreatedAt := since
	if maxAnswerCreatedAt.After(maxCreatedAt) {
		maxCreatedAt = maxAnswerCreatedAt
	}
	if maxCommentCreatedAt.After(maxCreatedAt) {
		maxCreatedAt = maxCommentCreatedAt
	}

	return updates, maxCreatedAt, nil
}

func (c *StackOverflowClient) getQuestionTitle(
	ctx context.Context,
	questionID int64,
) (string, error) {
	endpoint, err := c.buildURL(
		fmt.Sprintf("/questions/%d", questionID),
		map[string]string{
			"site":   c.site,
			"filter": "default",
		},
	)
	if err != nil {
		return "", err
	}

	var result stackOverflowQuestionResponse
	if err := c.getJSON(ctx, endpoint, &result); err != nil {
		return "", err
	}

	if len(result.Items) == 0 {
		return "", fmt.Errorf("stackoverflow question not found")
	}

	return cleanupText(result.Items[0].Title), nil
}

func (c *StackOverflowClient) getNewAnswers(
	ctx context.Context,
	questionID int64,
	linkURL string,
	linkID int64,
	title string,
	since time.Time,
) ([]domain.LinkUpdate, time.Time, error) {
	endpoint, err := c.buildURL(
		fmt.Sprintf("/questions/%d/answers", questionID),
		map[string]string{
			"site":     c.site,
			"sort":     "creation",
			"order":    "desc",
			"pagesize": strconv.Itoa(c.perPage),
			"filter":   "withbody",
		},
	)
	if err != nil {
		return nil, time.Time{}, err
	}

	var result stackOverflowAnswersResponse
	if err := c.getJSON(ctx, endpoint, &result); err != nil {
		return nil, time.Time{}, err
	}

	updates := make([]domain.LinkUpdate, 0)
	maxCreatedAt := since

	for _, item := range result.Items {
		createdAt := time.Unix(item.CreationDate, 0)

		if !since.IsZero() && !createdAt.After(since) {
			continue
		}

		updates = append(updates, domain.LinkUpdate{
			LinkID:    linkID,
			URL:       linkURL,
			Type:      domain.UpdateTypeStackOverflowAnswer,
			Title:     title,
			Username:  item.Owner.DisplayName,
			CreatedAt: createdAt,
			Preview:   MakePreview(item.Body),
		})

		if createdAt.After(maxCreatedAt) {
			maxCreatedAt = createdAt
		}
	}

	return updates, maxCreatedAt, nil
}

func (c *StackOverflowClient) getNewComments(
	ctx context.Context,
	questionID int64,
	linkURL string,
	linkID int64,
	title string,
	since time.Time,
) ([]domain.LinkUpdate, time.Time, error) {
	endpoint, err := c.buildURL(
		fmt.Sprintf("/questions/%d/comments", questionID),
		map[string]string{
			"site":     c.site,
			"sort":     "creation",
			"order":    "desc",
			"pagesize": strconv.Itoa(c.perPage),
			"filter":   "withbody",
		},
	)
	if err != nil {
		return nil, time.Time{}, err
	}

	var result stackOverflowCommentsResponse
	if err := c.getJSON(ctx, endpoint, &result); err != nil {
		return nil, time.Time{}, err
	}

	updates := make([]domain.LinkUpdate, 0)
	maxCreatedAt := since

	for _, item := range result.Items {
		createdAt := time.Unix(item.CreationDate, 0)

		if !since.IsZero() && !createdAt.After(since) {
			continue
		}

		updates = append(updates, domain.LinkUpdate{
			LinkID:    linkID,
			URL:       linkURL,
			Type:      domain.UpdateTypeStackOverflowComment,
			Title:     title,
			Username:  item.Owner.DisplayName,
			CreatedAt: createdAt,
			Preview:   MakePreview(item.Body),
		})

		if createdAt.After(maxCreatedAt) {
			maxCreatedAt = createdAt
		}
	}

	return updates, maxCreatedAt, nil
}

func (c *StackOverflowClient) buildURL(path string, params map[string]string) (string, error) {
	endpoint, err := url.Parse(c.baseURL + path)
	if err != nil {
		return "", err
	}

	query := endpoint.Query()
	for key, value := range params {
		query.Set(key, value)
	}

	endpoint.RawQuery = query.Encode()

	return endpoint.String(), nil
}

func (c *StackOverflowClient) getJSON(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "link-tracker")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("stackoverflow status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return err
	}

	return nil
}
