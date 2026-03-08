package main

import (
	"log/slog"
	"os"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/commands"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/logger"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/adapters"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/clients"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/repositories"
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
	scrapperClient := clients.NewScrapperClient(cfg.ScrapperURL)
	trackRepo := repositories.NewInMemoryTrackSessionRepository()
	trackService := services.NewTrackService(scrapperClient, trackRepo)
	dispatcher := commands.NewDefaultDispatcher(scrapperClient, trackService)

	updatesHandler := http.NewUpdatesHandler(bot)

	router := http.NewRouter(updatesHandler)

	go func() {
		log.Info("http server started on :8080")
		http.ListenAndServe(":8080", router)
	}()
	log.Info("bot started successfully")

	bot.Run(dispatcher, log)
}
