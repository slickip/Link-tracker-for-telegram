package commands

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"

type HelpCommand struct{}

var _ domain.Command = (*HelpCommand)(nil)

func NewHelpCommand() domain.Command {
	return &HelpCommand{}
}

func (c *HelpCommand) Name() string {
	return "/help"
}

func (c *HelpCommand) Execute(chatID int64, _ string) (string, error) {
	return "Доступные команды:\n" +
		"/start - начало работы с ботом\n" +
		"/help - показать это сообщение\n" +
		"/track - начать диалог для добавления ссылки\n" +
		"    Шаг 1: отправь /track\n" +
		"    Шаг 2: отправь ссылку (GitHub или StackOverflow)\n" +
		"    Шаг 3: отправь теги через запятую (например: work, bug) или '-' чтобы пропустить\n" +
		"/untrack <url> - перестать отслеживать указанную ссылку\n" +
		"/list [tag] - показать список отслеживаемых ссылок; если указать тег, список будет отфильтрован\n" +
		"/cancel - отменить текущую операцию (/track)", nil
}

