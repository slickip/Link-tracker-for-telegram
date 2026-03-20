package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	ScrapperURL   string
	BotGRPCAddr   string

	ScrapperGRPCAddr string
}

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

	botGRPCAddr := os.Getenv("BOT_GRPC_ADDR")
	if botGRPCAddr == "" {
		botGRPCAddr = "localhost:8082"
	}

	scrapperGRPCAddr := os.Getenv("SCRAPPER_GRPC_ADDR")
	if scrapperGRPCAddr == "" {
		scrapperGRPCAddr = "localhost:8083"
	}

	return &Config{
		TelegramToken:    token,
		ScrapperURL:      scrapperURL,
		BotGRPCAddr:      botGRPCAddr,
		ScrapperGRPCAddr: scrapperGRPCAddr,
	}
}
