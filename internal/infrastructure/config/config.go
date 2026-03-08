package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	ScrapperURL   string
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

	return &Config{
		TelegramToken: token,
		ScrapperURL:   scrapperURL,
	}
}
