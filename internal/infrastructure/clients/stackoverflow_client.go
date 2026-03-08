package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type StackOverflowClient struct {
	client *http.Client
}

type soResponse struct {
	Items []struct {
		LastActivityDate int64 `json:"last_activity_date"`
	} `json:"items"`
}

func NewStackOverflowClient() *StackOverflowClient {
	return &StackOverflowClient{
		client: &http.Client{},
	}
}

func (c *StackOverflowClient) GetQuestionUpdatedAt(
	ctx context.Context,
	questionID int64,
) (time.Time, error) {

	url := fmt.Sprintf(
		"https://api.stackexchange.com/2.3/questions/%d?site=stackoverflow",
		questionID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return time.Time{}, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return time.Time{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("stackoverflow status %d", resp.StatusCode)
	}

	var result soResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return time.Time{}, err
	}

	if len(result.Items) == 0 {
		return time.Time{}, fmt.Errorf("empty response")
	}

	return time.Unix(result.Items[0].LastActivityDate, 0), nil
}
