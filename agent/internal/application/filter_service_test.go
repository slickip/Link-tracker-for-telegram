package application

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

func TestFilterService_ShouldFilterByStopWord_TC_2_1(t *testing.T) {
	t.Parallel()

	filter := NewFilterService(
		[]string{"spam", "ads", "promo"},
		nil,
		20,
	)
	update := api.LinkUpdate{
		Username:    "normal-user",
		Description: "This update contains spam and should be ignored",
	}
	require.False(t, filter.ShouldProcess(update))
}

func TestFilterService_ShouldFilterByExcludedAuthor_TC_2_2(t *testing.T) {
	t.Parallel()

	filter := NewFilterService(
		nil,
		[]string{"bot-user"},
		20,
	)
	update := api.LinkUpdate{
		Username:    "bot-user",
		Description: "This update is long enough but author is excluded",
	}
	require.False(t, filter.ShouldProcess(update))
}

func TestFilterService_ShouldFilterByMinLength_TC_2_3(t *testing.T) {
	t.Parallel()

	filter := NewFilterService(
		nil,
		nil,
		20,
	)
	update := api.LinkUpdate{
		Username:    "normal-user",
		Description: "too short",
	}
	require.False(t, filter.ShouldProcess(update))
}

func TestFilterService_ShouldPassValidUpdate_T_2_4(t *testing.T) {
	t.Parallel()

	filter := NewFilterService(
		[]string{"spam", "ads", "promo"},
		[]string{"bot-user"},
		20,
	)
	update := api.LinkUpdate{
		Username:    "normal-user",
		Description: "This update is valid and should pass all filtering rules",
	}
	require.True(t, filter.ShouldProcess(update))
}

func TestFilterService_StopWordShouldMatchOnlyWholeWord(t *testing.T) {
	t.Parallel()

	filter := NewFilterService(
		[]string{"ads"},
		nil,
		1,
	)

	update := api.LinkUpdate{
		Username:    "normal-user",
		Description: "This update should not be filtered",
	}

	require.True(t, filter.ShouldProcess(update))
}
