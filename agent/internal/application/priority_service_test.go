package application

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPriorityService_ShouldReturnHigh_TC_1_1(t *testing.T) {
	t.Parallel()

	service := NewPriorityService(
		[]string{"critical", "urgent", "breaking", "security"},
		[]string{"minor", "typo", "chore", "docs"},
	)

	priority := service.DeterminePriority("critical bug fix")

	require.Equal(t, PriorityHigh, priority)
}

func TestPriorityService_ShouldReturnMedium_TC_1_2(t *testing.T) {
	t.Parallel()

	service := NewPriorityService(
		[]string{"critical", "urgent", "breaking", "security"},
		[]string{"minor", "typo", "chore", "docs"},
	)

	priority := service.DeterminePriority("regular feature update")

	require.Equal(t, PriorityMedium, priority)
}

func TestPriorityService_ShouldReturnLow_TC_1_3(t *testing.T) {
	t.Parallel()

	service := NewPriorityService(
		[]string{"critical", "urgent", "breaking", "security"},
		[]string{"minor", "typo", "chore", "docs"},
	)

	priority := service.DeterminePriority("fix typo in readme")

	require.Equal(t, PriorityLow, priority)
}

func TestPriorityService_HighHasPriorityOverLow(t *testing.T) {
	t.Parallel()

	service := NewPriorityService(
		[]string{"critical"},
		[]string{"typo"},
	)

	priority := service.DeterminePriority("critical typo fix")

	require.Equal(t, PriorityHigh, priority)
}
