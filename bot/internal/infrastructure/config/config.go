package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken       string
	ScrapperURL         string
	ScrapperHTTPTimeout time.Duration
	ScrapperHTTPRetry   RetryConfig
	BotHTTPAddr         string
	BotGRPCAddr         string
	ScrapperGRPCAddr    string
	DatabaseURL         string
	AccessType          AccessType
	Kafka               KafkaConfig
}

type AccessType string

const (
	AccessTypeSQL AccessType = "SQL"
	AccessTypeORM AccessType = "ORM"
)

type KafkaConfig struct {
	BootstrapServers    string
	LinkUpdatesTopic    string
	DLQTopic            string
	ConsumerGroup       string
	ClientID            string
	MaxRetries          int
	SchemaRegistryURL   string
	LinkUpdatesSubject  string
	SerializationFormat string
}

type RetryConfig struct {
	MaxAttempts           uint
	Delay                 time.Duration
	RetryableHTTPStatuses []int
}

const (
	envTelegramToken       = "TELEGRAM_TOKEN"
	envScrapperURL         = "SCRAPPER_URL"
	envScrapperHTTPTimeout = "SCRAPPER_HTTP_TIMEOUT"
	envBotHTTPAddr         = "BOT_HTTP_ADDR"
	envBotGRPCAddr         = "BOT_GRPC_ADDR"
	envScrapperGRPCAddr    = "SCRAPPER_GRPC_ADDR"
	envDatabaseURL         = "DATABASE_URL"
	envAccessType          = "ACCESS_TYPE"

	envKafkaBootstrapServers = "KAFKA_BOOTSTRAP_SERVERS"
	envKafkaLinkUpdatesTopic = "KAFKA_LINK_UPDATES_TOPIC"
	envKafkaDLQTopic         = "KAFKA_DLQ_TOPIC"
	envKafkaConsumerGroup    = "KAFKA_CONSUMER_GROUP"
	envKafkaClientID         = "KAFKA_CLIENT_ID"
	envKafkaMaxRetries       = "KAFKA_MAX_RETRIES"

	envSchemaRegistryURL        = "SCHEMA_REGISTRY_URL"
	envKafkaLinkUpdatesSubject  = "KAFKA_LINK_UPDATES_SUBJECT"
	envKafkaSerializationFormat = "KAFKA_SERIALIZATION_FORMAT"

	envScrapperHTTPRetryMaxAttempts      = "SCRAPPER_HTTP_RETRY_MAX_ATTEMPTS"
	envScrapperHTTPRetryDelay            = "SCRAPPER_HTTP_RETRY_DELAY"
	envScrapperHTTPRetryableHTTPStatuses = "SCRAPPER_HTTP_RETRYABLE_STATUSES"
)

const (
	defaultBotHTTPAddr         = ":8080"
	defaultBotGRPCAddr         = "localhost:8082"
	defaultScrapperGRPCAddr    = "localhost:8083"
	defaultScrapperHTTPTimeout = 5 * time.Second
	defaultDatabaseURL         = "postgres://postgres:12345@localhost:5432/notesdb?sslmode=disable"
	defaultAccessType          = string(AccessTypeSQL)

	defaultKafkaBootstrapServers = "localhost:19092,localhost:19093,localhost:19094"
	defaultKafkaLinkUpdatesTopic = "link-updates"
	defaultKafkaDLQTopic         = "link-updates-dlq"
	defaultKafkaConsumerGroup    = "bot-link-updates"
	defaultKafkaClientID         = "bot"
	defaultKafkaMaxRetries       = 3
	minKafkaMaxRetries           = 0

	defaultSchemaRegistryURL        = "http://localhost:18085"
	defaultKafkaLinkUpdatesSubject  = "link-updates-value"
	defaultKafkaSerializationFormat = "JSON"

	defaultScrapperHTTPRetryMaxAttempts      uint = 3
	defaultScrapperHTTPRetryDelay                 = 500 * time.Millisecond
	defaultScrapperHTTPRetryableHTTPStatuses      = "429,500,502,503,504"
)

func MustLoad() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found")
	}

	token := getRequiredEnv(envTelegramToken)
	scrapperURL := getRequiredEnv(envScrapperURL)
	scrapperHTTPTimeout := getEnvDuration(envScrapperHTTPTimeout, defaultScrapperHTTPTimeout)

	botHTTPAddr := getEnv(envBotHTTPAddr, defaultBotHTTPAddr)
	botGRPCAddr := getEnv(envBotGRPCAddr, defaultBotGRPCAddr)
	scrapperGRPCAddr := getEnv(envScrapperGRPCAddr, defaultScrapperGRPCAddr)
	databaseURL := getEnv(envDatabaseURL, defaultDatabaseURL)

	accessTypeStr := getEnv(envAccessType, defaultAccessType)
	accessType := AccessType(accessTypeStr)

	if accessType != AccessTypeSQL && accessType != AccessTypeORM {
		log.Fatalf("invalid %s: %s", envAccessType, accessTypeStr)
	}

	kafkaMaxRetries := getEnvInt(envKafkaMaxRetries, defaultKafkaMaxRetries)
	if kafkaMaxRetries < minKafkaMaxRetries {
		log.Fatalf("%s must be at least %d", envKafkaMaxRetries, minKafkaMaxRetries)
	}

	return &Config{
		TelegramToken:       token,
		ScrapperURL:         scrapperURL,
		ScrapperHTTPTimeout: scrapperHTTPTimeout,
		BotHTTPAddr:         botHTTPAddr,
		BotGRPCAddr:         botGRPCAddr,
		ScrapperGRPCAddr:    scrapperGRPCAddr,
		DatabaseURL:         databaseURL,
		AccessType:          accessType,
		Kafka: KafkaConfig{
			BootstrapServers:    getEnv(envKafkaBootstrapServers, defaultKafkaBootstrapServers),
			LinkUpdatesTopic:    getEnv(envKafkaLinkUpdatesTopic, defaultKafkaLinkUpdatesTopic),
			DLQTopic:            getEnv(envKafkaDLQTopic, defaultKafkaDLQTopic),
			ConsumerGroup:       getEnv(envKafkaConsumerGroup, defaultKafkaConsumerGroup),
			ClientID:            getEnv(envKafkaClientID, defaultKafkaClientID),
			MaxRetries:          kafkaMaxRetries,
			SchemaRegistryURL:   getEnv(envSchemaRegistryURL, defaultSchemaRegistryURL),
			LinkUpdatesSubject:  getEnv(envKafkaLinkUpdatesSubject, defaultKafkaLinkUpdatesSubject),
			SerializationFormat: getEnv(envKafkaSerializationFormat, defaultKafkaSerializationFormat),
		},
		ScrapperHTTPRetry: RetryConfig{
			MaxAttempts: getEnvUint(
				envScrapperHTTPRetryMaxAttempts,
				defaultScrapperHTTPRetryMaxAttempts,
			),
			Delay: getEnvDuration(
				envScrapperHTTPRetryDelay,
				defaultScrapperHTTPRetryDelay,
			),
			RetryableHTTPStatuses: getEnvIntSlice(
				envScrapperHTTPRetryableHTTPStatuses,
				defaultScrapperHTTPRetryableHTTPStatuses,
			),
		},
	}
}

func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s is not set", key)
	}

	return value
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

	if parsed <= 0 {
		log.Fatalf("%s must be positive", key)
	}

	return parsed
}

func getEnvUint(key string, defaultValue uint) uint {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseUint(value, 10, 0)
	if err != nil {
		log.Fatalf("invalid uint value for %s: %s", key, value)
	}

	if parsed == 0 {
		log.Fatalf("%s must be positive", key)
	}

	return uint(parsed)
}

func getEnvIntSlice(key string, defaultValue string) []int {
	value := getEnv(key, defaultValue)

	parts := strings.Split(value, ",")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		parsed, err := strconv.Atoi(part)
		if err != nil {
			log.Fatalf("invalid int value in %s: %s", key, part)
		}

		result = append(result, parsed)
	}

	if len(result) == 0 {
		log.Fatalf("%s must not be empty", key)
	}

	return result
}
