package main

import (
	"log/slog"
	"os"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/commands"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/logger"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/adapters"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(slog.LevelInfo)

	bot, err := adapters.NewBot(cfg.TelegramToken)
	if err != nil {
		slog.Error("failed to create bot", "error", err)
		os.Exit(1)
	}

	if err := bot.SetCommands(); err != nil {
		log.Warn("failed to set bot commands", "error", err)
		os.Exit(1)
	}

	dispatcher := commands.NewDefaultDispatcher()

	log.Info("bot started successfully")

	updates := bot.ListenUpdates()

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		text := update.Message.Text

		log.Info("received message", "chat_id", chatID, "text", text)

		cmd := dispatcher.Dispatch(text)

		response, err := cmd.Execute(chatID)
		if err != nil {
			log.Warn("command execution error", "error", err)
			continue
		}

		if err := bot.SendMessage(chatID, response); err != nil {
			log.Warn("failed to send message", "error", err)
		}
	}

}
