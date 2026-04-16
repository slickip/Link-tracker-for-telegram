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

func MustLoad() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found")
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN is not set")
	}

	scrapperURL := os.Getenv("SCRAPPER_URL")
	if scrapperURL == "" {
		log.Fatal("SCRAPPER_URL is not set")
	}

	botHTTPAddr := os.Getenv("BOT_HTTP_ADDR")
	if botHTTPAddr == "" {
		botHTTPAddr = ":8080"
	}

	botGRPCAddr := os.Getenv("BOT_GRPC_ADDR")
	if botGRPCAddr == "" {
		botGRPCAddr = "localhost:8082"
	}

	scrapperGRPCAddr := os.Getenv("SCRAPPER_GRPC_ADDR")
	if scrapperGRPCAddr == "" {
		scrapperGRPCAddr = "localhost:8083"
	}

	accessType := os.Getenv("ACCESS_TYPE")
	if accessType == "" {
		accessType = "SQL"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:12345@localhost:5432/notesdb?sslmode=disable"
	}
	return &Config{
		TelegramToken:    token,
		ScrapperURL:      scrapperURL,
		BotHTTPAddr:      botHTTPAddr,
		BotGRPCAddr:      botGRPCAddr,
		ScrapperGRPCAddr: scrapperGRPCAddr,
		DatabaseURL:      databaseURL,
		AccessType:       AccessType(accessType),
	}
}
