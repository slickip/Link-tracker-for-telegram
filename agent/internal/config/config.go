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
	Kafka         KafkaConfig
	Filtering     FilteringConfig
	Summarization SummarizationConfig
	AI            AIConfig
}

type KafkaConfig struct {
	BootstrapServers    string
	RawUpdatesTopic     string
	ProcessedTopic      string
	DLQTopic            string
	ConsumerGroup       string
	ClientID            string
	MaxRetries          int
	SchemaRegistryURL   string
	RawUpdatesSubject   string
	ProcessedSubject    string
	SerializationFormat string
}

type FilteringConfig struct {
	StopWords       []string
	ExcludedAuthors []string
	MinLength       int
}

type SummarizationConfig struct {
	Threshold int
}

type AIConfig struct {
	Enabled bool
	APIURL  string
	Token   string
	Model   string
	Timeout time.Duration
}

const (
	defaultKafkaBootstrapServers = "localhost:19092,localhost:19093,localhost:19094"
	defaultRawUpdatesTopic       = "link.raw-updates"
	defaultProcessedTopic        = "link.processed-updates"
	defaultDLQTopic              = "link.raw-updates-dlq"
	defaultConsumerGroup         = "ai-agent"
	defaultClientID              = "ai-agent"
	defaultMaxRetries            = 3
	defaultSchemaRegistryURL     = "http://localhost:18085"
	defaultRawUpdatesSubject     = "link.raw-updates-value"
	defaultProcessedSubject      = "link.processed-updates-value"
	defaultSerializationFormat   = "JSON"

	defaultStopWords       = "spam,ads,promo"
	defaultExcludedAuthors = "bot-user"
	defaultMinLength       = 20
	defaultThreshold       = 500

	defaultAIEnabled = false
	defaultAIURL     = "https://api-inference.huggingface.co/models/facebook/bart-large-cnn"
	defaultAIModel   = "facebook/bart-large-cnn"
	defaultAITimeout = 20 * time.Second
)

func MustLoad() *Config {
	_ = godotenv.Load()

	return &Config{
		Kafka: KafkaConfig{
			BootstrapServers:    getEnv("AI_KAFKA_BOOTSTRAP_SERVERS", defaultKafkaBootstrapServers),
			RawUpdatesTopic:     getEnv("AI_KAFKA_RAW_UPDATES_TOPIC", defaultRawUpdatesTopic),
			ProcessedTopic:      getEnv("AI_KAFKA_PROCESSED_UPDATES_TOPIC", defaultProcessedTopic),
			DLQTopic:            getEnv("AI_KAFKA_DLQ_TOPIC", defaultDLQTopic),
			ConsumerGroup:       getEnv("AI_KAFKA_CONSUMER_GROUP", defaultConsumerGroup),
			ClientID:            getEnv("AI_KAFKA_CLIENT_ID", defaultClientID),
			MaxRetries:          getEnvInt("AI_KAFKA_MAX_RETRIES", defaultMaxRetries),
			SchemaRegistryURL:   getEnv("AI_SCHEMA_REGISTRY_URL", defaultSchemaRegistryURL),
			RawUpdatesSubject:   getEnv("AI_KAFKA_RAW_UPDATES_SUBJECT", defaultRawUpdatesSubject),
			ProcessedSubject:    getEnv("AI_KAFKA_PROCESSED_UPDATES_SUBJECT", defaultProcessedSubject),
			SerializationFormat: getEnv("AI_KAFKA_SERIALIZATION_FORMAT", defaultSerializationFormat),
		},
		Filtering: FilteringConfig{
			StopWords:       getEnvStringSlice("AI_FILTER_STOP_WORDS", defaultStopWords),
			ExcludedAuthors: getEnvStringSlice("AI_FILTER_EXCLUDED_AUTHORS", defaultExcludedAuthors),
			MinLength:       getEnvInt("AI_FILTER_MIN_LENGTH", defaultMinLength),
		},
		Summarization: SummarizationConfig{
			Threshold: getEnvInt("AI_SUMMARIZATION_THRESHOLD", defaultThreshold),
		},
		AI: AIConfig{
			Enabled: getEnvBool("AI_SUMMARIZER_ENABLED", defaultAIEnabled),
			APIURL:  getEnv("AI_SUMMARIZER_API_URL", defaultAIURL),
			Token:   os.Getenv("AI_SUMMARIZER_TOKEN"),
			Model:   getEnv("AI_SUMMARIZER_MODEL", defaultAIModel),
			Timeout: getEnvDuration("AI_SUMMARIZER_TIMEOUT", defaultAITimeout),
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
