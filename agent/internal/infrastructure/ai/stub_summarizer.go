package ai

import (
	"context"

	"github.com/slickip/link-tracker/pkg/logger"
)

type StubSummarizer struct {
	log *logger.Slog
}

func NewStubSummarizer(log *logger.Slog) *StubSummarizer {
	return &StubSummarizer{
		log: log,
	}
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
		s.log.Error(
			logInvalidThreshold,
			"threshold", threshold,
		)
		return text, nil
	}

	return cutWithEllipsis(text, threshold), nil
}
