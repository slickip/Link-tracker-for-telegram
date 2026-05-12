package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken    string
	ScrapperURL      string
	BotHTTPAddr      string
	BotGRPCAddr      string
	ScrapperGRPCAddr string
	DatabaseURL      string
	AccessType       AccessType
}

type AccessType string

const (
	AccessTypeSQL AccessType = "SQL"
	AccessTypeORM AccessType = "ORM"
)

const (
	envTelegramToken    = "TELEGRAM_TOKEN"
	envScrapperURL      = "SCRAPPER_URL"
	envBotHTTPAddr      = "BOT_HTTP_ADDR"
	envBotGRPCAddr      = "BOT_GRPC_ADDR"
	envScrapperGRPCAddr = "SCRAPPER_GRPC_ADDR"
	envDatabaseURL      = "DATABASE_URL"
	envAccessType       = "ACCESS_TYPE"
)

const (
	defaultBotHTTPAddr      = ":8080"
	defaultBotGRPCAddr      = "localhost:8082"
	defaultScrapperGRPCAddr = "localhost:8083"
	defaultDatabaseURL      = "postgres://postgres:12345@localhost:5432/notesdb?sslmode=disable"
	defaultAccessType       = string(AccessTypeSQL)
)

func MustLoad() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found")
	}

	token := getRequiredEnv(envTelegramToken)
	scrapperURL := getRequiredEnv(envScrapperURL)

	botHTTPAddr := getEnv(envBotHTTPAddr, defaultBotHTTPAddr)
	botGRPCAddr := getEnv(envBotGRPCAddr, defaultBotGRPCAddr)
	scrapperGRPCAddr := getEnv(envScrapperGRPCAddr, defaultScrapperGRPCAddr)
	databaseURL := getEnv(envDatabaseURL, defaultDatabaseURL)

	accessTypeStr := getEnv(envAccessType, defaultAccessType)
	accessType := AccessType(accessTypeStr)

	if accessType != AccessTypeSQL && accessType != AccessTypeORM {
		log.Fatalf("invalid %s: %s", envAccessType, accessTypeStr)
	}

	return &Config{
		TelegramToken:    token,
		ScrapperURL:      scrapperURL,
		BotHTTPAddr:      botHTTPAddr,
		BotGRPCAddr:      botGRPCAddr,
		ScrapperGRPCAddr: scrapperGRPCAddr,
		DatabaseURL:      databaseURL,
		AccessType:       accessType,
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
