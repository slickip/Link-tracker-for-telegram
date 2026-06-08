package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type HuggingFaceSummarizer struct {
	client *http.Client
	apiURL string
	token  string
}

func NewHuggingFaceSummarizer(
	apiURL string,
	token string,
	timeout time.Duration,
) *HuggingFaceSummarizer {
	return &HuggingFaceSummarizer{
		client: &http.Client{
			Timeout: timeout,
		},
		apiURL: strings.TrimSpace(apiURL),
		token:  strings.TrimSpace(token),
	}
}

type huggingFaceRequest struct {
	Inputs string `json:"inputs"`
}

type huggingFaceResponseItem struct {
	SummaryText string `json:"summary_text"`
}

func (s *HuggingFaceSummarizer) Summarize(
	ctx context.Context,
	text string,
	threshold int,
) (string, error) {
	if s.apiURL == "" {
		return fallbackCut(text, threshold), nil
	}

	payload, err := json.Marshal(huggingFaceRequest{
		Inputs: "Summarize the following update in 2-3 sentences:\n\n" + text,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.apiURL,
		bytes.NewReader(payload),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fallbackCut(text, threshold), nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fallbackCut(text, threshold), nil
	}

	var response []huggingFaceResponseItem
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fallbackCut(text, threshold), nil
	}

	if len(response) == 0 || strings.TrimSpace(response[0].SummaryText) == "" {
		return fallbackCut(text, threshold), nil
	}

	return strings.TrimSpace(response[0].SummaryText), nil
}

func fallbackCut(text string, threshold int) string {
	if threshold <= 0 {
		return "..."
	}

	if utf8.RuneCountInString(text) <= threshold {
		return text
	}

	runes := []rune(text)
	return fmt.Sprintf("%s...", string(runes[:threshold]))
}
