package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type GitHubClient struct {
	client *http.Client
}

type githubRepoResponse struct {
	UpdatedAt time.Time `json:"updated_at"`
}

func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		client: &http.Client{},
	}
}

func (c *GitHubClient) GetRepoUpdatedAt(
	ctx context.Context,
	owner string,
	repo string,
) (time.Time, error) {

	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s",
		owner,
		repo,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return time.Time{}, err
	}
	req.Header.Set("User-Agent", "link-tracker")

	resp, err := c.client.Do(req)
	if err != nil {
		return time.Time{}, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("github status %d", resp.StatusCode)
	}

	var result githubRepoResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return time.Time{}, err
	}

	return result.UpdatedAt, nil
}
