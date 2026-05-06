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

func MustLoad() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found")
	}

	botHTTPURL := getEnv("BOT_HTTP_URL", "http://localhost:8080")
	botGRPCAddr := getEnv("BOT_GRPC_ADDR", "localhost:8082")
	scrapperHTTPAddr := getEnv("SCRAPPER_HTTP_ADDR", ":8081")
	scrapperGRPCAddr := getEnv("SCRAPPER_GRPC_ADDR", ":8083")

	databaseURL := getEnv(
		"DATABASE_URL",
		"postgres://postgres:12345@localhost:5432/notesdb?sslmode=disable",
	)

	accessTypeStr := getEnv("ACCESS_TYPE", "SQL")
	accessType := AccessType(accessTypeStr)

	if accessType != AccessTypeSQL && accessType != AccessTypeORM {
		log.Fatalf("invalid ACCESS_TYPE: %s", accessTypeStr)
	}

	linkBatchSize := getEnvInt("LINK_BATCH_SIZE", 100)
	if linkBatchSize < 50 || linkBatchSize > 500 {
		log.Fatalf("LINK_BATCH_SIZE must be between 50 and 500")
	}

	workerCount := getEnvInt("WORKER_COUNT", 4)
	if workerCount <= 0 {
		log.Fatalf("WORKER_COUNT must be positive")
	}

	externalAPIPerPage := getEnvInt("EXTERNAL_API_PER_PAGE", 100)
	if externalAPIPerPage <= 0 {
		log.Fatalf("EXTERNAL_API_PER_PAGE must be positive")
	}

	return &Config{
		BotHTTPURL:       botHTTPURL,
		BotGRPCAddr:      botGRPCAddr,
		ScrapperHTTPAddr: scrapperHTTPAddr,
		ScrapperGRPCAddr: scrapperGRPCAddr,
		DatabaseURL:      databaseURL,
		AccessType:       accessType,

		CheckInterval: getEnvDuration("CHECK_INTERVAL", 30*time.Second),
		LinkBatchSize: linkBatchSize,
		WorkerCount:   workerCount,

		GitHubBaseURL:        getEnv("GITHUB_BASE_URL", "https://api.github.com"),
		GitHubToken:          os.Getenv("GITHUB_TOKEN"),
		StackOverflowBaseURL: getEnv("STACKOVERFLOW_BASE_URL", "https://api.stackexchange.com/2.3"),
		StackOverflowSite:    getEnv("STACKOVERFLOW_SITE", "stackoverflow"),
		ExternalAPITimeout:   getEnvDuration("EXTERNAL_API_TIMEOUT", 10*time.Second),
		ExternalAPIPerPage:   externalAPIPerPage,
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
