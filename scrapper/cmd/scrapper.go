package main

import (
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	scrapperpb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/clients"
	grpcserver "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/grpc"
	httpserver "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/http"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/http/handlers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/repositories"
	"google.golang.org/grpc"
)

func main() {
	_ = godotenv.Load()
	log := logger.New(slog.LevelInfo)

	chatRepo := repositories.NewInMemoryChatRepository()
	linkRepo := repositories.NewInMemoryLinkRepository()

	chatService := services.NewChatService(chatRepo)
	linkService := services.NewLinkService(linkRepo, chatRepo)

	botHTTPURL := os.Getenv("BOT_HTTP_URL")
	if botHTTPURL == "" {
		botHTTPURL = "http://localhost:8080"
	}

	botGRPCAddr := os.Getenv("BOT_GRPC_ADDR")
	if botGRPCAddr == "" {
		botGRPCAddr = "localhost:8082"
	}

	scrapperGRPCAddr := os.Getenv("SCRAPPER_GRPC_ADDR")
	if scrapperGRPCAddr == "" {
		scrapperGRPCAddr = "localhost:8083"
	}

	httpBotClient := clients.NewHTTPBotClient(botHTTPURL)
	var botClient clients.BotClient = httpBotClient
	grpcBotClient, err := clients.NewGRPCBotClient(botGRPCAddr)
	if err != nil {
		log.Warn("failed to init bot grpc client, fallback to http only", "error", err)
	} else {
		botClient = clients.NewFallbackBotClient(httpBotClient, grpcBotClient, log)
	}

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

	go func() {
		lis, err := net.Listen("tcp", scrapperGRPCAddr)
		if err != nil {
			log.Error("scrapper gRPC listen failed", "error", err)
			return
		}

		grpcSrv := grpc.NewServer()
		scrapperpb.RegisterScrapperServiceServer(
			grpcSrv,
			grpcserver.NewScrapperGRPCServer(chatService, linkService, log),
		)

		log.Info("scrapper gRPC server starting", "addr", scrapperGRPCAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Error("scrapper gRPC server failed", "error", err)
		}
	}()

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
