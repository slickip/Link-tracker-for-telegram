package commands

import (
	"context"
	"fmt"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/clients"
)

const (
	minArgsForURL = 2 // /untrack <url>
	minArgsForTag = 3 // /untrack tag <tag>
)

type UntrackCommand struct {
	client clients.ScrapperClient
}

func NewUntrackCommand(client clients.ScrapperClient) *UntrackCommand {
	return &UntrackCommand{client: client}
}

func (c *UntrackCommand) Name() string {
	return "/untrack"
}

func (c *UntrackCommand) Execute(ctx context.Context, chatID int64, text string) (string, error) {
	args := strings.Fields(text)

	if len(args) < minArgsForURL {
		return "Использование:\n/untrack <url> — удалить по ссылке\n/untrack tag <tag> — удалить все ссылки с тегом", nil
	}

	if err := c.client.RegisterChat(ctx, chatID); err != nil {
		return "Не удалось зарегистрировать чат. Попробуй позже", err
	}

	commandArg := args[1]

	if commandArg == "tag" || !isURL(commandArg) {
		if len(args) < minArgsForTag {
			if commandArg != "tag" {
				removedCount, err := c.client.RemoveLinksByTag(ctx, chatID, commandArg)
				if err != nil {
					return "Не получилось удалить ссылки по тегу", err
				}
				if removedCount == 0 {
					return "Ссылки с указанным тегом не найдены", nil
				}
				return fmt.Sprintf("Удалено ссылок с тегом %s: %d", commandArg, removedCount), nil
			}
			return "Использование: /untrack tag <tag>", nil
		}

		tagToRemove := strings.Join(args[2:], " ")

		removedCount, err := c.client.RemoveLinksByTag(ctx, chatID, tagToRemove)
		if err != nil {
			return "Не получилось удалить ссылки по тегу", err
		}

		if removedCount == 0 {
			return "Ссылки с указанным тегом не найдены", nil
		}

		return fmt.Sprintf("Удалено ссылок с тегом %s: %d", tagToRemove, removedCount), nil
	}

	urlToRemove := commandArg

	if err := c.client.RemoveLink(ctx, chatID, urlToRemove); err != nil {
		return "Не получилось удалить ссылку", err
	}

	return "Ссылка удалена", nil
}

func isURL(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
