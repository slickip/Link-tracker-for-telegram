package ai

import (
	"context"
	"unicode/utf8"
)

type StubSummarizer struct{}

func NewStubSummarizer() *StubSummarizer {
	return &StubSummarizer{}
}

func (s *StubSummarizer) Summarize(
	ctx context.Context,
	text string,
	threshold int,
) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	if threshold <= 0 {
		return "...", nil
	}

	if utf8.RuneCountInString(text) <= threshold {
		return text, nil
	}

	runes := []rune(text)
	return string(runes[:threshold]) + "...", nil
}
