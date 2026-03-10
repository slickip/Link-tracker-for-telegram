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
		log.Error("failed to create bot", "error", err)
		os.Exit(1)
	}

	if err := bot.SetCommands(); err != nil {
		log.Warn("failed to set bot commands", "error", err)
	}

	dispatcher := commands.NewDefaultDispatcher()

	log.Info("bot started successfully")

	bot.Run(dispatcher, log)
}
