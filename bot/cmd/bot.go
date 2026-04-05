package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/clients"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/commands"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain/repositories"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/adapters"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/config"
	grpcserver "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/grpc"
	httpserver "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/http"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/http/handlers"
	dbpkgorm "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/persistence/orm/database"
	ormrepo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/persistence/orm/repositories"
	dbpkgsql "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/persistence/sql/database"
	sqlrepo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/persistence/sql/repositories"
	botpb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

func main() {
	//hw-4
	var (
		cfg = config.MustLoad()
		log = logger.New(slog.LevelInfo)
	)

	bot, err := adapters.NewBot(cfg.TelegramToken)
	if err != nil {
		log.Error("failed to create bot", "error", err)
		os.Exit(1)
	}

	if err := bot.SetCommands(); err != nil {
		log.Warn("failed to set bot commands", "error", err)
	}

	httpScrapperClient := clients.NewScrapperClient(cfg.ScrapperURL)

	var scrapperClient clients.ScrapperClient = httpScrapperClient

	grpcScrapperClient, err := clients.NewGRPCScrapperClient(cfg.ScrapperGRPCAddr)
	if err != nil {
		log.Warn("gRPC unavailable, using HTTP only", "error", err)
	} else {
		log.Info("gRPC available, enabling fallback (HTTP → gRPC)")
		scrapperClient = clients.NewFallbackScrapperClient(httpScrapperClient, grpcScrapperClient, log)
	}

	var trackRepo repositories.TrackSessionRepository

	switch cfg.AccessType {
	case "SQL":
		sqlDB, err := dbpkgsql.NewSQLDB(cfg.DatabaseURL)
		if err != nil {
			log.Error("failed to connect sql db", "error", err)
			os.Exit(1)
		}

		if err := sqlDB.Ping(); err != nil {
			log.Error("failed to ping sql db", "error", err)
			os.Exit(1)
		}

		trackRepo = sqlrepo.NewSQLTrackSessionRepository(sqlDB)
		log.Info("bot repository initialized", "access_type", "SQL")

	case "ORM":
		gormDB, err := dbpkgorm.NewGormDB(cfg.DatabaseURL)
		if err != nil {
			log.Error("failed to connect gorm db", "error", err)
			os.Exit(1)
		}

		trackRepo = ormrepo.NewORMTrackSessionRepository(gormDB)
		log.Info("bot repository initialized", "access_type", "ORM")

	default:
		log.Error("unknown access type", "access_type", cfg.AccessType)
		os.Exit(1)
	}

	trackService := services.NewTrackService(scrapperClient, trackRepo)

	dispatcher := commands.NewDefaultDispatcher(
		scrapperClient,
		trackService,
		trackRepo,
	)

	updatesHandler := handlers.NewUpdatesHandler(bot)
	router := httpserver.NewBotRouter(updatesHandler)

	g, _ := errgroup.WithContext(context.Background())

	g.Go(func() error {
		addr := cfg.BotHTTPAddr
		log.Info("bot HTTP server starting", "addr", addr)
		return http.ListenAndServe(addr, router)
	})

	g.Go(func() error {
		lis, err := net.Listen("tcp", cfg.BotGRPCAddr)
		if err != nil {
			return err
		}

		grpcSrv := grpc.NewServer()
		botpb.RegisterBotServiceServer(
			grpcSrv,
			grpcserver.NewBotGRPCServer(bot, log),
		)

		log.Info("bot gRPC server starting", "addr", cfg.BotGRPCAddr)

		return grpcSrv.Serve(lis)
	})

	g.Go(func() error {
		log.Info("telegram bot loop starting")
		bot.Run(dispatcher, log)
		return nil
	})

	log.Info("bot started successfully")

	if err := g.Wait(); err != nil {
		log.Error("service failed", "error", err)
	}
}
