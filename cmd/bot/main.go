package main

import (
	"log/slog"
	"net/http"
	"os"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/adapters"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/clients"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/commands"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/config"
	httpserver "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/http"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/http/handlers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/repositories"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/logger"
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

	trackService := services.NewTrackService(
		scrapperClient,
		trackRepo,
	)

	dispatcher := commands.NewDefaultDispatcher(
		scrapperClient,
		trackService,
		trackRepo,
	)

	updatesHandler := handlers.NewUpdatesHandler(bot)
	router := httpserver.NewBotRouter(updatesHandler)

	go func() {
		addr := ":8080"
		log.Info("bot HTTP server starting", "addr", addr)

		if err := http.ListenAndServe(addr, router); err != nil {
			log.Error("bot HTTP server failed", "error", err)
		}
	}()

	log.Info("bot started successfully")

	bot.Run(dispatcher, log)
}
