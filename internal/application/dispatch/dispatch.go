package dispatch

import (
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type Dispatcher struct {
	commands map[string]domain.Command
}

func NewDispatcher(commands []domain.Command) *Dispatcher {
	cmdMap := make(map[string]domain.Command)

	for _, cmd := range commands {
		cmdMap[cmd.Name()] = cmd
	}

	return &Dispatcher{
		commands: cmdMap,
	}
}

func (d *Dispatcher) Dispatch(text string) domain.Command {
	if text == "" {
		return d.commands["unknown"]
	}

	commandName := strings.Split(text, " ")[0]

	cmd, ok := d.commands[commandName]
	if !ok {
		return d.commands["unknown"]
	}

	return cmd
}
