package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	BotHTTPURL       string
	BotGRPCAddr      string
	ScrapperHTTPAddr string
	ScrapperGRPCAddr string
	DatabaseURL      string
	AccessType       AccessType

	NotificationTransport NotificationTransport
	Kafka                 KafkaConfig
	Outbox                OutboxConfig

	CheckInterval time.Duration
	LinkBatchSize int
	WorkerCount   int

	GitHubBaseURL        string
	GitHubToken          string
	StackOverflowBaseURL string
	StackOverflowSite    string
	ExternalAPITimeout   time.Duration
	ExternalAPIPerPage   int
}

type AccessType string

const (
	AccessTypeSQL AccessType = "SQL"
	AccessTypeORM AccessType = "ORM"
)

const (
	envBotHTTPURL       = "BOT_HTTP_URL"
	envBotGRPCAddr      = "BOT_GRPC_ADDR"
	envScrapperHTTPAddr = "SCRAPPER_HTTP_ADDR"
	envScrapperGRPCAddr = "SCRAPPER_GRPC_ADDR"
	envDatabaseURL      = "DATABASE_URL"
	envAccessType       = "ACCESS_TYPE"

	envCheckInterval = "CHECK_INTERVAL"
	envLinkBatchSize = "LINK_BATCH_SIZE"
	envWorkerCount   = "WORKER_COUNT"

	envGitHubBaseURL        = "GITHUB_BASE_URL"
	envGitHubToken          = "GITHUB_TOKEN"
	envStackOverflowBaseURL = "STACKOVERFLOW_BASE_URL"
	envStackOverflowSite    = "STACKOVERFLOW_SITE"
	envExternalAPITimeout   = "EXTERNAL_API_TIMEOUT"
	envExternalAPIPerPage   = "EXTERNAL_API_PER_PAGE"

	envNotificationTransport = "NOTIFICATION_TRANSPORT"

	envKafkaBootstrapServers = "KAFKA_BOOTSTRAP_SERVERS"
	envKafkaLinkUpdatesTopic = "KAFKA_LINK_UPDATES_TOPIC"
	envKafkaClientID         = "KAFKA_CLIENT_ID"

	envOutboxEnabled         = "OUTBOX_ENABLED"
	envOutboxPublishInterval = "OUTBOX_PUBLISH_INTERVAL"
	envOutboxBatchSize       = "OUTBOX_BATCH_SIZE"
)

const (
	defaultBotHTTPURL       = "http://localhost:8080"
	defaultBotGRPCAddr      = "localhost:8082"
	defaultScrapperHTTPAddr = ":8081"
	defaultScrapperGRPCAddr = ":8083"
	defaultDatabaseURL      = "postgres://postgres:12345@localhost:5432/notesdb?sslmode=disable"
	defaultAccessType       = string(AccessTypeSQL)

	defaultCheckInterval = 30 * time.Second
	defaultLinkBatchSize = 100
	defaultWorkerCount   = 4

	defaultGitHubBaseURL        = "https://api.github.com"
	defaultGitHubToken          = ""
	defaultStackOverflowBaseURL = "https://api.stackexchange.com/2.3"
	defaultStackOverflowSite    = "stackoverflow"
	defaultExternalAPITimeout   = 10 * time.Second
	defaultExternalAPIPerPage   = 100

	defaultNotificationTransport = string(NotificationTransportKafka)

	defaultKafkaBootstrapServers = "localhost:19092,localhost:19093,localhost:19094"
	defaultKafkaLinkUpdatesTopic = "link-updates"
	defaultKafkaClientID         = "scrapper"

	defaultOutboxEnabled         = true
	defaultOutboxPublishInterval = 5 * time.Second
	defaultOutboxBatchSize       = 100
)

const (
	minLinkBatchSize      = 50
	maxLinkBatchSize      = 500
	minWorkerCount        = 1
	minExternalAPIPerPage = 1
)

type NotificationTransport string

const (
	NotificationTransportKafka NotificationTransport = "KAFKA"
	NotificationTransportHTTP  NotificationTransport = "HTTP"
	NotificationTransportGRPC  NotificationTransport = "GRPC"
)

type KafkaConfig struct {
	BootstrapServers string
	LinkUpdatesTopic string
	ClientID         string
}

type OutboxConfig struct {
	Enabled         bool
	PublishInterval time.Duration
	BatchSize       int
}

func MustLoad() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found")
	}

	botHTTPURL := getEnv(envBotHTTPURL, defaultBotHTTPURL)
	botGRPCAddr := getEnv(envBotGRPCAddr, defaultBotGRPCAddr)
	scrapperHTTPAddr := getEnv(envScrapperHTTPAddr, defaultScrapperHTTPAddr)
	scrapperGRPCAddr := getEnv(envScrapperGRPCAddr, defaultScrapperGRPCAddr)
	databaseURL := getEnv(envDatabaseURL, defaultDatabaseURL)

	accessTypeStr := getEnv(envAccessType, defaultAccessType)
	accessType := AccessType(accessTypeStr)

	if accessType != AccessTypeSQL && accessType != AccessTypeORM {
		log.Fatalf("invalid %s: %s", envAccessType, accessTypeStr)
	}

	notificationTransportStr := getEnv(envNotificationTransport, defaultNotificationTransport)
	notificationTransport := NotificationTransport(notificationTransportStr)

	if notificationTransport != NotificationTransportKafka &&
		notificationTransport != NotificationTransportHTTP &&
		notificationTransport != NotificationTransportGRPC {
		log.Fatalf("invalid %s: %s", envNotificationTransport, notificationTransportStr)
	}

	kafkaConfig := KafkaConfig{
		BootstrapServers: getEnv(envKafkaBootstrapServers, defaultKafkaBootstrapServers),
		LinkUpdatesTopic: getEnv(envKafkaLinkUpdatesTopic, defaultKafkaLinkUpdatesTopic),
		ClientID:         getEnv(envKafkaClientID, defaultKafkaClientID),
	}

	linkBatchSize := getEnvInt(envLinkBatchSize, defaultLinkBatchSize)
	if linkBatchSize < minLinkBatchSize || linkBatchSize > maxLinkBatchSize {
		log.Fatalf(
			"%s must be between %d and %d",
			envLinkBatchSize,
			minLinkBatchSize,
			maxLinkBatchSize,
		)
	}

	workerCount := getEnvInt(envWorkerCount, defaultWorkerCount)
	if workerCount < minWorkerCount {
		log.Fatalf("%s must be at least %d", envWorkerCount, minWorkerCount)
	}

	externalAPIPerPage := getEnvInt(envExternalAPIPerPage, defaultExternalAPIPerPage)
	if externalAPIPerPage < minExternalAPIPerPage {
		log.Fatalf("%s must be at least %d", envExternalAPIPerPage, minExternalAPIPerPage)
	}

	return &Config{
		BotHTTPURL:       botHTTPURL,
		BotGRPCAddr:      botGRPCAddr,
		ScrapperHTTPAddr: scrapperHTTPAddr,
		ScrapperGRPCAddr: scrapperGRPCAddr,
		DatabaseURL:      databaseURL,
		AccessType:       accessType,

		NotificationTransport: notificationTransport,
		Kafka:                 kafkaConfig,

		CheckInterval: getEnvDuration(envCheckInterval, defaultCheckInterval),
		LinkBatchSize: linkBatchSize,
		WorkerCount:   workerCount,

		GitHubBaseURL:        getEnv(envGitHubBaseURL, defaultGitHubBaseURL),
		GitHubToken:          getEnv(envGitHubToken, defaultGitHubToken),
		StackOverflowBaseURL: getEnv(envStackOverflowBaseURL, defaultStackOverflowBaseURL),
		StackOverflowSite:    getEnv(envStackOverflowSite, defaultStackOverflowSite),
		ExternalAPITimeout:   getEnvDuration(envExternalAPITimeout, defaultExternalAPITimeout),
		ExternalAPIPerPage:   externalAPIPerPage,
		Outbox: OutboxConfig{
			Enabled:         getEnvBool(envOutboxEnabled, defaultOutboxEnabled),
			PublishInterval: getEnvDuration(envOutboxPublishInterval, defaultOutboxPublishInterval),
			BatchSize:       getEnvInt(envOutboxBatchSize, defaultOutboxBatchSize),
		},
	}
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("invalid int value for %s: %s", key, value)
	}

	return parsed
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Fatalf("invalid duration value for %s: %s", key, value)
	}

	return parsed
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Fatalf("invalid bool value for %s: %s", key, value)
	}

	return parsed
}
