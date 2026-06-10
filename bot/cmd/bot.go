package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/slickip/link-tracker/bot/internal/application/clients"
	"github.com/slickip/link-tracker/bot/internal/application/commands"
	"github.com/slickip/link-tracker/bot/internal/application/services"
	"github.com/slickip/link-tracker/bot/internal/domain/repositories"
	"github.com/slickip/link-tracker/bot/internal/infrastructure/adapters"
	"github.com/slickip/link-tracker/bot/internal/infrastructure/config"
	grpcserver "github.com/slickip/link-tracker/bot/internal/infrastructure/grpc"
	httpserver "github.com/slickip/link-tracker/bot/internal/infrastructure/http"
	"github.com/slickip/link-tracker/bot/internal/infrastructure/http/handlers"
	botmetrics "github.com/slickip/link-tracker/bot/internal/infrastructure/metrics"
	dbpkgorm "github.com/slickip/link-tracker/bot/internal/infrastructure/persistence/orm/database"
	ormrepo "github.com/slickip/link-tracker/bot/internal/infrastructure/persistence/orm/repositories"
	dbpkgsql "github.com/slickip/link-tracker/bot/internal/infrastructure/persistence/sql/database"
	sqlrepo "github.com/slickip/link-tracker/bot/internal/infrastructure/persistence/sql/repositories"
	botpb "github.com/slickip/link-tracker/pkg/api/bot"
	h "github.com/slickip/link-tracker/pkg/helpers"
	"github.com/slickip/link-tracker/pkg/kafka"
	"github.com/slickip/link-tracker/pkg/logger"
	commonmetrics "github.com/slickip/link-tracker/pkg/metrics"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

func main() {
	var (
		cfg = config.MustLoad()
		log = logger.New(slog.LevelInfo)
	)

	botmetrics.Register()

	metricsServer := commonmetrics.NewServer(":8011", slog.Default())
	metricsServer.Start()
	defer func() {
		_ = metricsServer.Shutdown(context.Background())
	}()

	bot, err := adapters.NewBot(cfg.TelegramToken)
	if err != nil {
		log.Error("failed to create bot", "error", err)
		os.Exit(1)
	}

	if err := bot.SetCommands(); err != nil {
		log.Warn("failed to set bot commands", "error", err)
	}

	httpScrapperClient := clients.NewScrapperClient(
		cfg.ScrapperURL,
		cfg.ScrapperHTTPTimeout,
		h.HTTPRetryConfig{
			MaxAttempts:           cfg.ScrapperHTTPRetry.MaxAttempts,
			Delay:                 cfg.ScrapperHTTPRetry.Delay,
			RetryableHTTPStatuses: cfg.ScrapperHTTPRetry.RetryableHTTPStatuses,
		},
		h.CircuitBreakerConfig{
			Enabled:                       cfg.ScrapperHTTPCircuitBreaker.Enabled,
			FailureRateThreshold:          cfg.ScrapperHTTPCircuitBreaker.FailureRateThreshold,
			MinimumRequests:               cfg.ScrapperHTTPCircuitBreaker.MinimumRequests,
			SlidingWindowInterval:         cfg.ScrapperHTTPCircuitBreaker.SlidingWindowInterval,
			SlidingWindowBucketPeriod:     cfg.ScrapperHTTPCircuitBreaker.SlidingWindowBucketPeriod,
			WaitDurationInOpenState:       cfg.ScrapperHTTPCircuitBreaker.WaitDurationInOpenState,
			PermittedCallsInHalfOpenState: cfg.ScrapperHTTPCircuitBreaker.PermittedCallsInHalfOpenState,
		},
	)

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

	updateService := services.NewUpdateService(bot)

	updatesHandler := handlers.NewUpdatesHandler(updateService)
	router := httpserver.NewBotRouter(updatesHandler)

	router = h.RateLimitMiddleware(h.RateLimiterConfig{
		Enabled:           cfg.RateLimit.Enabled,
		RequestsPerSecond: cfg.RateLimit.RequestsPerSecond,
		Burst:             cfg.RateLimit.Burst,
		CleanupInterval:   cfg.RateLimit.CleanupInterval,
		TTL:               cfg.RateLimit.TTL,
	})(router)

	kafkaConsumer, err := kafka.NewLinkUpdateConsumer(
		kafka.LinkUpdateConsumerConfig{
			BootstrapServers:    cfg.Kafka.BootstrapServers,
			Topic:               cfg.Kafka.LinkUpdatesTopic,
			DLQTopic:            cfg.Kafka.DLQTopic,
			ConsumerGroup:       cfg.Kafka.ConsumerGroup,
			ClientID:            cfg.Kafka.ClientID,
			MaxRetries:          cfg.Kafka.MaxRetries,
			SchemaRegistryURL:   cfg.Kafka.SchemaRegistryURL,
			LinkUpdatesSubject:  cfg.Kafka.LinkUpdatesSubject,
			SerializationFormat: cfg.Kafka.SerializationFormat,
		},
		updateService,
	)
	if err != nil {
		log.Error("failed to create kafka link update consumer", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := kafkaConsumer.Close(); err != nil {
			log.Warn("failed to close kafka consumer", "error", err)
		}
	}()

	g, ctx := errgroup.WithContext(context.Background())

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
		log.Info(
			"kafka link update consumer starting",
			"topic",
			cfg.Kafka.LinkUpdatesTopic,
			"group",
			cfg.Kafka.ConsumerGroup,
		)

		err := kafkaConsumer.Start(ctx)
		if errors.Is(err, context.Canceled) {
			return nil
		}

		return err
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
