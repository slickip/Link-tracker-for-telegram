package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type fakeSummarizer struct {
	called bool
	result string
}

func (s *fakeSummarizer) Summarize(
	ctx context.Context,
	text string,
	threshold int,
) (string, error) {
	s.called = true

	if s.result != "" {
		return s.result, nil
	}

	return "summary", nil
}

func TestProcessor_ShouldSummarizeLongText_TC_3_1(t *testing.T) {
	t.Parallel()

	summarizer := &fakeSummarizer{
		result: "short summary",
	}

	processor := NewProcessor(
		NewFilterService(nil, nil, 1),
		summarizer,
		NewPriorityService([]string{"critical"}, []string{"typo"}),
		20,
	)

	originalText := "This is a very long update description that definitely exceeds the summarization threshold"

	result, ok, err := processor.Process(context.Background(), api.LinkUpdate{
		ID:          1,
		URL:         "https://github.com/test/repo",
		TgChatIDs:   []int64{111},
		Username:    "normal-user",
		Description: originalText,
	})

	require.NoError(t, err)
	require.True(t, ok)
	require.True(t, summarizer.called)
	require.Equal(t, "short summary", result.Description)
	require.NotEqual(t, originalText, result.Description)
	require.Equal(t, string(PriorityMedium), result.Priority)
}

func TestProcessor_ShouldNotSummarizeShortText_TC_3_2(t *testing.T) {
	t.Parallel()

	summarizer := &fakeSummarizer{}

	processor := NewProcessor(
		NewFilterService(nil, nil, 1),
		summarizer,
		NewPriorityService([]string{"critical"}, []string{"typo"}),
		100,
	)

	originalText := "Short update"

	result, ok, err := processor.Process(context.Background(), api.LinkUpdate{
		ID:          1,
		URL:         "https://github.com/test/repo",
		TgChatIDs:   []int64{111},
		Username:    "normal-user",
		Description: originalText,
	})

	require.NoError(t, err)
	require.True(t, ok)
	require.False(t, summarizer.called)
	require.Equal(t, originalText, result.Description)
	require.Equal(t, string(PriorityMedium), result.Priority)
}
