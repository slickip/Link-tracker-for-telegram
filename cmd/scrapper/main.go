package main

import (
	"log/slog"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/logger"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/clients"
	httpserver "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/http"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/http/handlers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/repositories"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/services"
)

func main() {

	log := logger.New(slog.LevelInfo)

	chatRepo := repositories.NewInMemoryChatRepository()
	linkRepo := repositories.NewInMemoryLinkRepository()

	chatService := services.NewChatService(chatRepo)
	linkService := services.NewLinkService(linkRepo, chatRepo)

	botClient := clients.NewHTTPBotClient("http://localhost:8080")

	githubClient := clients.NewGitHubClient()
	soClient := clients.NewStackOverflowClient()

	scheduler := services.NewScheduler(
		linkRepo,
		githubClient,
		soClient,
		botClient,
		log,
	)

	scheduler.Start()

	chatHandler := handlers.NewChatHandler(chatService)
	linkHandler := handlers.NewLinkHandler(linkService)

	router := httpserver.NewRouter(
		chatHandler,
		linkHandler,
	)

	log.Info("scrapper HTTP server starting", "addr", ":8081")

	if err := http.ListenAndServe(":8081", router); err != nil {
		log.Error("scrapper HTTP server failed", "error", err)
	}
}
