package main

import (
	"database/sql"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	scrapperpb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/config"
	domainrepo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain/repositories"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/clients"
	grpcserver "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/grpc"
	httpserver "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/http"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/http/handlers"
	ormrepo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/persistence/orm/repositories"
	sqlrepo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/persistence/sql/repositories"
)

func main() {
	_ = godotenv.Load()

	log := logger.New(slog.LevelInfo)
	cfg := config.MustLoad()

	var (
		chatRepo     domainrepo.ChatRepository
		linkRepo     domainrepo.LinkRepository
		trackingRepo domainrepo.TrackingRepository
		tagRepo      domainrepo.TagRepository
	)

	switch cfg.AccessType {
	case config.AccessTypeSQL:
		sqlDB, err := sql.Open("postgres", cfg.DatabaseURL)
		if err != nil {
			log.Error("failed to open sql database", "error", err)
			os.Exit(1)
		}

		if err := sqlDB.Ping(); err != nil {
			log.Error("failed to ping sql database", "error", err)
			os.Exit(1)
		}

		chatRepo = sqlrepo.NewSQLChatRepository(sqlDB)
		linkRepo = sqlrepo.NewSQLChatLinkRepository(sqlDB, log)
		trackingRepo = sqlrepo.NewSQLTrackingRepository(sqlDB, log)
		tagRepo = sqlrepo.NewSQLTagRepository(sqlDB, log)

		log.Info("scrapper repositories initialized", "access_type", "SQL")

	case config.AccessTypeORM:
		gormDB, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
		if err != nil {
			log.Error("failed to open orm database", "error", err)
			os.Exit(1)
		}

		chatRepo = ormrepo.NewORMChatRepository(gormDB)
		linkRepo = ormrepo.NewORMChatLinkRepository(gormDB)
		trackingRepo = ormrepo.NewGormTrackingRepository(gormDB)
		tagRepo = ormrepo.NewORMTagRepository(gormDB)

		log.Info("scrapper repositories initialized", "access_type", "ORM")

	default:
		log.Error("unknown access type", "access_type", cfg.AccessType)
		os.Exit(1)
	}

	chatService := services.NewChatService(chatRepo)
	linkService := services.NewLinkService(linkRepo, chatRepo)
	tagService := services.NewTagService(tagRepo, chatRepo)

	httpBotClient := clients.NewHTTPBotClient(cfg.BotHTTPURL)

	var botClient clients.BotClient = httpBotClient
	grpcBotClient, err := clients.NewGRPCBotClient(cfg.BotGRPCAddr)
	if err != nil {
		log.Warn("failed to init bot grpc client, fallback to http only", "error", err)
	} else {
		botClient = clients.NewFallbackBotClient(httpBotClient, grpcBotClient, log)
	}

	githubClient := clients.NewGitHubClient()
	soClient := clients.NewStackOverflowClient()

	scheduler := services.NewScheduler(
		trackingRepo,
		githubClient,
		soClient,
		botClient,
		log,
	)
	scheduler.Start()

	go func() {
		lis, err := net.Listen("tcp", cfg.ScrapperGRPCAddr)
		if err != nil {
			log.Error("scrapper gRPC listen failed", "error", err)
			return
		}

		grpcSrv := grpc.NewServer()
		scrapperpb.RegisterScrapperServiceServer(
			grpcSrv,
			grpcserver.NewScrapperGRPCServer(chatService, linkService, log),
		)

		log.Info("scrapper gRPC server starting", "addr", cfg.ScrapperGRPCAddr)

		if err := grpcSrv.Serve(lis); err != nil {
			log.Error("scrapper gRPC server failed", "error", err)
		}
	}()

	chatHandler := handlers.NewChatHandler(chatService)
	linkHandler := handlers.NewLinkHandler(linkService)
	tagHandler := handlers.NewTagHandler(tagService)

	router := httpserver.NewRouter(
		chatHandler,
		linkHandler,
		tagHandler,
	)

	log.Info("scrapper HTTP server starting", "addr", cfg.ScrapperHTTPAddr)

	if err := http.ListenAndServe(cfg.ScrapperHTTPAddr, router); err != nil {
		log.Error("scrapper HTTP server failed", "error", err)
	}
}
