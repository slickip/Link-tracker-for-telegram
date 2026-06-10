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
	BotHTTPURL          string
	BotGRPCAddr         string
	ScrapperHTTPAddr    string
	ScrapperGRPCAddr    string
	ScrapperHTTPTimeout time.Duration
	DatabaseURL         string
	AccessType          AccessType

	NotificationTransport NotificationTransport
	Kafka                 KafkaConfig
	Outbox                OutboxConfig

	Valkey ValkeyConfig

	CheckInterval time.Duration
	LinkBatchSize int
	WorkerCount   int

	GitHubBaseURL             string
	GitHubToken               string
	StackOverflowBaseURL      string
	StackOverflowSite         string
	ExternalAPITimeout        time.Duration
	ExternalAPIPerPage        int
	ExternalAPIRetry          RetryConfig
	ExternalAPICircuitBreaker CircuitBreakerConfig
	RateLimit                 RateLimitConfig
}

type AccessType string

const (
	AccessTypeSQL AccessType = "SQL"
	AccessTypeORM AccessType = "ORM"
)

const (
	envBotHTTPURL          = "BOT_HTTP_URL"
	envBotGRPCAddr         = "BOT_GRPC_ADDR"
	envScrapperHTTPAddr    = "SCRAPPER_HTTP_ADDR"
	envScrapperGRPCAddr    = "SCRAPPER_GRPC_ADDR"
	envScrapperHTTPTimeout = "SCRAPPER_HTTP_TIMEOUT"
	envDatabaseURL         = "DATABASE_URL"
	envAccessType          = "ACCESS_TYPE"

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

	envSchemaRegistryURL        = "SCHEMA_REGISTRY_URL"
	envKafkaLinkUpdatesSubject  = "KAFKA_LINK_UPDATES_SUBJECT"
	envKafkaSerializationFormat = "KAFKA_SERIALIZATION_FORMAT"

	envValkeyEnabled                = "VALKEY_ENABLED"
	envValkeyAddresses              = "VALKEY_ADDRESSES"
	envValkeyUsername               = "VALKEY_USERNAME"
	envValkeyPassword               = "VALKEY_PASSWORD"
	envValkeyTTL                    = "VALKEY_TTL"
	envValkeyClientSideCacheEnabled = "VALKEY_CLIENT_SIDE_CACHE_ENABLED"
	envValkeyClientSideCacheTTL     = "VALKEY_CLIENT_SIDE_CACHE_TTL"

	envExternalAPIRetryMaxAttempts      = "EXTERNAL_API_RETRY_MAX_ATTEMPTS"
	envExternalAPIRetryDelay            = "EXTERNAL_API_RETRY_DELAY"
	envExternalAPIRetryableHTTPStatuses = "EXTERNAL_API_RETRYABLE_STATUSES"

	envExternalAPICircuitBreakerEnabled                       = "EXTERNAL_API_CB_ENABLED"
	envExternalAPICircuitBreakerFailureRateThreshold          = "EXTERNAL_API_CB_FAILURE_RATE_THRESHOLD"
	envExternalAPICircuitBreakerMinimumRequests               = "EXTERNAL_API_CB_MINIMUM_REQUESTS"
	envExternalAPICircuitBreakerSlidingWindowInterval         = "EXTERNAL_API_CB_SLIDING_WINDOW_INTERVAL"
	envExternalAPICircuitBreakerSlidingWindowBucketPeriod     = "EXTERNAL_API_CB_SLIDING_WINDOW_BUCKET_PERIOD"
	envExternalAPICircuitBreakerWaitDurationInOpenState       = "EXTERNAL_API_CB_WAIT_DURATION_IN_OPEN_STATE"
	envExternalAPICircuitBreakerPermittedCallsInHalfOpenState = "EXTERNAL_API_CB_PERMITTED_CALLS_IN_HALF_OPEN_STATE"

	envRateLimitEnabled           = "RATE_LIMIT_ENABLED"
	envRateLimitRequestsPerSecond = "RATE_LIMIT_REQUESTS_PER_SECOND"
	envRateLimitBurst             = "RATE_LIMIT_BURST"
	envRateLimitCleanupInterval   = "RATE_LIMIT_CLEANUP_INTERVAL"
	envRateLimitTTL               = "RATE_LIMIT_TTL"
)

const (
	defaultBotHTTPURL          = "http://localhost:8080"
	defaultBotGRPCAddr         = "localhost:8082"
	defaultScrapperHTTPAddr    = ":8081"
	defaultScrapperGRPCAddr    = ":8083"
	defaultScrapperHTTPTimeout = 5 * time.Second
	defaultDatabaseURL         = "postgres://postgres:12345@localhost:5432/notesdb?sslmode=disable"
	defaultAccessType          = string(AccessTypeSQL)

	defaultCheckInterval = 30 * time.Second
	defaultLinkBatchSize = 100
	defaultWorkerCount   = 4

	defaultGitHubBaseURL        = "https://api.github.com"
	defaultGitHubToken          = ""
	defaultStackOverflowBaseURL = "https://api.stackexchange.com/2.3"
	defaultStackOverflowSite    = "stackoverflow"
	defaultExternalAPITimeout   = 10 * time.Second
	defaultExternalAPIPerPage   = 100

	defaultNotificationTransport = string(NotificationTransportHTTP)

	defaultKafkaBootstrapServers = "localhost:19092,localhost:19093,localhost:19094"
	defaultKafkaLinkUpdatesTopic = "link.raw-updates"
	defaultKafkaClientID         = "scrapper"

	defaultOutboxEnabled         = true
	defaultOutboxPublishInterval = 5 * time.Second
	defaultOutboxBatchSize       = 100

	defaultSchemaRegistryURL        = "http://localhost:18085"
	defaultKafkaLinkUpdatesSubject  = "link.raw-updates-value"
	defaultKafkaSerializationFormat = "JSON"

	defaultValkeyEnabled                = true
	defaultValkeyAddresses              = "localhost:6379,localhost:6380,localhost:6381"
	defaultValkeyUsername               = ""
	defaultValkeyPassword               = ""
	defaultValkeyTTL                    = 5 * time.Minute
	defaultValkeyClientSideCacheEnabled = true
	defaultValkeyClientSideCacheTTL     = 30 * time.Second

	defaultExternalAPIRetryMaxAttempts      uint = 3
	defaultExternalAPIRetryDelay                 = 500 * time.Millisecond
	defaultExternalAPIRetryableHTTPStatuses      = "429,500,502,503,504"

	defaultExternalAPICircuitBreakerEnabled                            = true
	defaultExternalAPICircuitBreakerFailureRateThreshold               = 50.0
	defaultExternalAPICircuitBreakerMinimumRequests               uint = 2
	defaultExternalAPICircuitBreakerSlidingWindowInterval              = 10 * time.Second
	defaultExternalAPICircuitBreakerSlidingWindowBucketPeriod          = time.Second
	defaultExternalAPICircuitBreakerWaitDurationInOpenState            = time.Second
	defaultExternalAPICircuitBreakerPermittedCallsInHalfOpenState uint = 2

	defaultRateLimitEnabled           = true
	defaultRateLimitRequestsPerSecond = 10.0
	defaultRateLimitBurst             = 20
	defaultRateLimitCleanupInterval   = time.Minute
	defaultRateLimitTTL               = 5 * time.Minute
)

const (
	minLinkBatchSize      = 50
	maxLinkBatchSize      = 500
	minWorkerCount        = 1
	minExternalAPIPerPage = 1
	float64BitSize        = 64
)

type NotificationTransport string

const (
	NotificationTransportKafka NotificationTransport = "KAFKA"
	NotificationTransportHTTP  NotificationTransport = "HTTP"
	NotificationTransportGRPC  NotificationTransport = "GRPC"
)

type KafkaConfig struct {
	BootstrapServers    string
	LinkUpdatesTopic    string
	ClientID            string
	SchemaRegistryURL   string
	LinkUpdatesSubject  string
	SerializationFormat string
}

type OutboxConfig struct {
	Enabled         bool
	PublishInterval time.Duration
	BatchSize       int
}

type ValkeyConfig struct {
	Enabled                bool
	Addresses              []string
	Username               string
	Password               string
	TTL                    time.Duration
	ClientSideCacheEnabled bool
	ClientSideCacheTTL     time.Duration
}

type RetryConfig struct {
	MaxAttempts           uint
	Delay                 time.Duration
	RetryableHTTPStatuses []int
}

type CircuitBreakerConfig struct {
	Enabled                       bool
	FailureRateThreshold          float64
	MinimumRequests               uint32
	SlidingWindowInterval         time.Duration
	SlidingWindowBucketPeriod     time.Duration
	WaitDurationInOpenState       time.Duration
	PermittedCallsInHalfOpenState uint32
}

type RateLimitConfig struct {
	Enabled           bool
	RequestsPerSecond float64
	Burst             int
	CleanupInterval   time.Duration
	TTL               time.Duration
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
		BootstrapServers:   getEnv(envKafkaBootstrapServers, defaultKafkaBootstrapServers),
		LinkUpdatesTopic:   getEnv(envKafkaLinkUpdatesTopic, defaultKafkaLinkUpdatesTopic),
		ClientID:           getEnv(envKafkaClientID, defaultKafkaClientID),
		SchemaRegistryURL:  getEnv(envSchemaRegistryURL, defaultSchemaRegistryURL),
		LinkUpdatesSubject: getEnv(envKafkaLinkUpdatesSubject, defaultKafkaLinkUpdatesSubject),
		SerializationFormat: getEnv(
			envKafkaSerializationFormat,
			defaultKafkaSerializationFormat,
		),
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

	valkeyConfig := ValkeyConfig{
		Enabled:                getEnvBool(envValkeyEnabled, defaultValkeyEnabled),
		Addresses:              getEnvStringSlice(envValkeyAddresses, defaultValkeyAddresses),
		Username:               getEnv(envValkeyUsername, defaultValkeyUsername),
		Password:               getEnv(envValkeyPassword, defaultValkeyPassword),
		TTL:                    getEnvDuration(envValkeyTTL, defaultValkeyTTL),
		ClientSideCacheEnabled: getEnvBool(envValkeyClientSideCacheEnabled, defaultValkeyClientSideCacheEnabled),
		ClientSideCacheTTL:     getEnvDuration(envValkeyClientSideCacheTTL, defaultValkeyClientSideCacheTTL),
	}

	if valkeyConfig.Enabled && len(valkeyConfig.Addresses) == 0 {
		log.Fatalf("%s must not be empty when Valkey is enabled", envValkeyAddresses)
	}

	if valkeyConfig.Enabled && valkeyConfig.TTL <= 0 {
		log.Fatalf("%s must be positive when Valkey is enabled", envValkeyTTL)
	}

	return &Config{
		BotHTTPURL:          botHTTPURL,
		BotGRPCAddr:         botGRPCAddr,
		ScrapperHTTPAddr:    scrapperHTTPAddr,
		ScrapperGRPCAddr:    scrapperGRPCAddr,
		ScrapperHTTPTimeout: getEnvDuration(envScrapperHTTPTimeout, defaultScrapperHTTPTimeout),
		DatabaseURL:         databaseURL,
		AccessType:          accessType,

		NotificationTransport: notificationTransport,
		Kafka:                 kafkaConfig,

		Valkey: valkeyConfig,

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
		ExternalAPIRetry: RetryConfig{
			MaxAttempts: getEnvUint(
				envExternalAPIRetryMaxAttempts,
				defaultExternalAPIRetryMaxAttempts,
			),
			Delay: getEnvDuration(
				envExternalAPIRetryDelay,
				defaultExternalAPIRetryDelay,
			),
			RetryableHTTPStatuses: getEnvIntSlice(
				envExternalAPIRetryableHTTPStatuses,
				defaultExternalAPIRetryableHTTPStatuses,
			),
		},
		ExternalAPICircuitBreaker: CircuitBreakerConfig{
			Enabled: getEnvBool(
				envExternalAPICircuitBreakerEnabled,
				defaultExternalAPICircuitBreakerEnabled,
			),
			FailureRateThreshold: getEnvFloat64(
				envExternalAPICircuitBreakerFailureRateThreshold,
				defaultExternalAPICircuitBreakerFailureRateThreshold,
			),
			MinimumRequests: uint32(getEnvUint(
				envExternalAPICircuitBreakerMinimumRequests,
				defaultExternalAPICircuitBreakerMinimumRequests,
			)),
			SlidingWindowInterval: getEnvDuration(
				envExternalAPICircuitBreakerSlidingWindowInterval,
				defaultExternalAPICircuitBreakerSlidingWindowInterval,
			),
			SlidingWindowBucketPeriod: getEnvDuration(
				envExternalAPICircuitBreakerSlidingWindowBucketPeriod,
				defaultExternalAPICircuitBreakerSlidingWindowBucketPeriod,
			),
			WaitDurationInOpenState: getEnvDuration(
				envExternalAPICircuitBreakerWaitDurationInOpenState,
				defaultExternalAPICircuitBreakerWaitDurationInOpenState,
			),
			PermittedCallsInHalfOpenState: uint32(getEnvUint(
				envExternalAPICircuitBreakerPermittedCallsInHalfOpenState,
				defaultExternalAPICircuitBreakerPermittedCallsInHalfOpenState,
			)),
		},
		RateLimit: loadRateLimitConfig(),
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

func getEnvStringSlice(key string, defaultValue string) []string {
	value := getEnv(key, defaultValue)

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	return result
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

func getEnvFloat64(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseFloat(value, float64BitSize)
	if err != nil {
		log.Fatalf("invalid float value for %s: %s", key, value)
	}

	return parsed
}

func loadRateLimitConfig() RateLimitConfig {
	requestsPerSecond := getEnvFloat64(
		envRateLimitRequestsPerSecond,
		defaultRateLimitRequestsPerSecond,
	)
	if requestsPerSecond <= 0 {
		log.Fatalf("%s must be positive", envRateLimitRequestsPerSecond)
	}

	burst := getEnvInt(envRateLimitBurst, defaultRateLimitBurst)
	if burst <= 0 {
		log.Fatalf("%s must be positive", envRateLimitBurst)
	}

	cleanupInterval := getEnvDuration(
		envRateLimitCleanupInterval,
		defaultRateLimitCleanupInterval,
	)
	if cleanupInterval <= 0 {
		log.Fatalf("%s must be positive", envRateLimitCleanupInterval)
	}

	ttl := getEnvDuration(envRateLimitTTL, defaultRateLimitTTL)
	if ttl <= 0 {
		log.Fatalf("%s must be positive", envRateLimitTTL)
	}

	return RateLimitConfig{
		Enabled:           getEnvBool(envRateLimitEnabled, defaultRateLimitEnabled),
		RequestsPerSecond: requestsPerSecond,
		Burst:             burst,
		CleanupInterval:   cleanupInterval,
		TTL:               ttl,
	}
}
