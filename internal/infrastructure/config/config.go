package config

import (
	"log"
	"os"
)

type Config struct {
	TelegramToken string `env:"TELEGRAM_TOKEN" required:"true"`
}

func MustLoad() *Config {
	token := os.Getenv("TELEGRAM_TOKEN")

	if token == "" {
		log.Fatal("TELEGRAM_TOKEN is not set")
	}

	return &Config{
		TelegramToken: token,
	}
}
