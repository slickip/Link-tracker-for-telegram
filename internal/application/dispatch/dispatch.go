package dispatch

import (
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type Dispatcher struct {
	commands map[string]domain.Command
	unknown  domain.Command
}

func NewDispatcher(commands []domain.Command, unknown domain.Command) *Dispatcher {
	cmdMap := make(map[string]domain.Command)

	for _, cmd := range commands {
		cmdMap[cmd.Name()] = cmd
	}

	return &Dispatcher{
		commands: cmdMap,
		unknown:  unknown,
	}
}

func (d *Dispatcher) Dispatch(text string) domain.Command {
	if text == "" {
		return d.unknown
	}

	commandName := strings.Split(text, " ")[0]

	cmd, ok := d.commands[commandName]
	if !ok {
		return d.unknown
	}

	return cmd
}
