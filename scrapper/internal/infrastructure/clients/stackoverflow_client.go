package clients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
)

const (
	defaultStackOverflowBaseURL = "https://api.stackexchange.com/2.3"
	defaultStackOverflowSite    = "stackoverflow"
	defaultStackOverflowTimeout = 10 * time.Second
	defaultStackOverflowPerPage = 100
)

const (
	stackOverflowQuestionPathTemplate = "/questions/%d"
	stackOverflowAnswersPathTemplate  = "/questions/%d/answers"
	stackOverflowCommentsPathTemplate = "/questions/%d/comments"
)

const (
	queryParamSite     = "site"
	queryParamFilter   = "filter"
	queryParamSort     = "sort"
	queryParamOrder    = "order"
	queryParamPageSize = "pagesize"
)

const (
	stackOverflowFilterDefault  = "default"
	stackOverflowFilterWithBody = "withbody"
	stackOverflowSortCreation   = "creation"
	stackOverflowOrderDesc      = "desc"
)

const (
	userAgentHeader = "User-Agent"
	userAgentValue  = "link-tracker"
)

const (
	stackOverflowQuestionNotFoundError = "stackoverflow question not found"
	stackOverflowStatusErrorFormat     = "stackoverflow status %d"
)

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
		site = defaultStackOverflowSite
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultStackOverflowTimeout
	}

	perPage := cfg.PerPage
	if perPage <= 0 {
		perPage = defaultStackOverflowPerPage
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
		fmt.Sprintf(stackOverflowQuestionPathTemplate, questionID),
		map[string]string{
			queryParamSite:   c.site,
			queryParamFilter: stackOverflowFilterDefault,
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
		return "", errors.New(stackOverflowQuestionNotFoundError)
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
		fmt.Sprintf(stackOverflowAnswersPathTemplate, questionID),
		map[string]string{
			queryParamSite:     c.site,
			queryParamSort:     stackOverflowSortCreation,
			queryParamOrder:    stackOverflowOrderDesc,
			queryParamPageSize: strconv.Itoa(c.perPage),
			queryParamFilter:   stackOverflowFilterWithBody,
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
		fmt.Sprintf(stackOverflowCommentsPathTemplate, questionID),
		map[string]string{
			queryParamSite:     c.site,
			queryParamSort:     stackOverflowSortCreation,
			queryParamOrder:    stackOverflowOrderDesc,
			queryParamPageSize: strconv.Itoa(c.perPage),
			queryParamFilter:   stackOverflowFilterWithBody,
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

	req.Header.Set(userAgentHeader, userAgentValue)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(stackOverflowStatusErrorFormat, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return err
	}

	return nil
}
