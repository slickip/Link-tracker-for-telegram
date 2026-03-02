package dispatch_test

import (
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/commands"
)

func TestStartCommand_Positive(t *testing.T) {
	dispatcher := commands.NewDefaultDispatcher()

	cmd := dispatcher.Dispatch("/start")

	response, err := cmd.Execute(123)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response == "" {
		t.Fatal("expected non-empty response for /start")
	}
}

func TestHelpCommand_Positive(t *testing.T) {
	dispatcher := commands.NewDefaultDispatcher()

	cmd := dispatcher.Dispatch("/help")

	response, err := cmd.Execute(123)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response == "" {
		t.Fatal("expected non-empty response for /help")
	}
}

func TestUnknownCommand_Negative(t *testing.T) {
	dispatcher := commands.NewDefaultDispatcher()

	cmd := dispatcher.Dispatch("/unknown_command")

	response, err := cmd.Execute(123)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response == "" {
		t.Fatal("expected error message for unknown command")
	}
}
