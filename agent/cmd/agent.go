package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/agent/internal/application"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/agent/internal/config"
	infraai "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/agent/internal/infrastructure/ai"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/kafka"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(slog.LevelInfo)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	filterService := application.NewFilterService(
		cfg.Filtering.StopWords,
		cfg.Filtering.ExcludedAuthors,
		cfg.Filtering.MinLength,
	)

	var summarizer application.Summarizer
	if cfg.AI.Enabled {
		summarizer = infraai.NewHuggingFaceSummarizer(
			cfg.AI.APIURL,
			cfg.AI.Token,
			cfg.AI.Timeout,
		)
		log.Info("AI summarizer enabled")
	} else {
		summarizer = infraai.NewStubSummarizer()
		log.Info("stub summarizer enabled")
	}

	processor := application.NewProcessor(
		filterService,
		summarizer,
		cfg.Summarization.Threshold,
	)

	producer, err := kafka.NewConfluentLinkUpdateProducer(
		kafka.LinkUpdateProducerConfig{
			BootstrapServers:    cfg.Kafka.BootstrapServers,
			Topic:               cfg.Kafka.ProcessedTopic,
			ClientID:            cfg.Kafka.ClientID + "-producer",
			SchemaRegistryURL:   cfg.Kafka.SchemaRegistryURL,
			LinkUpdatesSubject:  cfg.Kafka.ProcessedSubject,
			SerializationFormat: cfg.Kafka.SerializationFormat,
		},
	)
	if err != nil {
		log.Error("failed to create processed update producer", "error", err)
		os.Exit(1)
	}
	defer producer.Close()

	handler := application.NewHandler(processor, producer, log)

	consumer, err := kafka.NewLinkUpdateConsumer(
		kafka.LinkUpdateConsumerConfig{
			BootstrapServers:    cfg.Kafka.BootstrapServers,
			Topic:               cfg.Kafka.RawUpdatesTopic,
			DLQTopic:            cfg.Kafka.DLQTopic,
			ConsumerGroup:       cfg.Kafka.ConsumerGroup,
			ClientID:            cfg.Kafka.ClientID + "-consumer",
			MaxRetries:          cfg.Kafka.MaxRetries,
			SchemaRegistryURL:   cfg.Kafka.SchemaRegistryURL,
			LinkUpdatesSubject:  cfg.Kafka.RawUpdatesSubject,
			SerializationFormat: cfg.Kafka.SerializationFormat,
		},
		handler,
	)
	if err != nil {
		log.Error("failed to create raw update consumer", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Warn("failed to close kafka consumer", "error", err)
		}
	}()

	log.Info(
		"AI Agent started",
		"raw_topic", cfg.Kafka.RawUpdatesTopic,
		"processed_topic", cfg.Kafka.ProcessedTopic,
	)

	if err := consumer.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("AI Agent stopped with error", "error", err)
		os.Exit(1)
	}
}
