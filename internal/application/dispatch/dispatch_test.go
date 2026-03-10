package dispatch_test

import (
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/commands"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
)

func TestStartCommand_Positive(t *testing.T) {
	dispatcher := commands.NewDefaultDispatcher()

	cmd := dispatcher.Dispatch("/start")

	response, err := cmd.Execute(123)

	if err != nil {
		t.Fatalf("%v: %v", pkg.ErrUnexpectedError, err)
	}

	if response == "" {
		t.Fatal(pkg.ErrEmptyStartResponse)
	}
}

func TestHelpCommand_Positive(t *testing.T) {
	dispatcher := commands.NewDefaultDispatcher()

	cmd := dispatcher.Dispatch("/help")

	response, err := cmd.Execute(123)

	if err != nil {
		t.Fatalf("%v: %v", pkg.ErrUnexpectedError, err)
	}

	if response == "" {
		t.Fatal(pkg.ErrEmptyHelpResponse)
	}
}

func TestUnknownCommand_Negative(t *testing.T) {
	dispatcher := commands.NewDefaultDispatcher()

	cmd := dispatcher.Dispatch("/unknown_command")

	response, err := cmd.Execute(123)

	if err != nil {
		t.Fatalf("%v: %v", pkg.ErrUnexpectedError, err)
	}

	if response == "" {
		t.Fatal(pkg.ErrEmptyUnknownResponse)
	}
}
