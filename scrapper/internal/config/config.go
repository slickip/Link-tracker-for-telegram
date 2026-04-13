package config

import "os"

const (
	defaultBotHTTPURL       = "http://localhost:8080"
	defaultBotGRPCAddr      = "localhost:8082"
	defaultScrapperGRPCAddr = "localhost:8083"
)

type Config struct {
	BotHTTPURL       string
	BotGRPCAddr      string
	ScrapperGRPCAddr string
}

func MustLoad() Config {
	return Config{
		BotHTTPURL:       getEnv("BOT_HTTP_URL", defaultBotHTTPURL),
		BotGRPCAddr:      getEnv("BOT_GRPC_ADDR", defaultBotGRPCAddr),
		ScrapperGRPCAddr: getEnv("SCRAPPER_GRPC_ADDR", defaultScrapperGRPCAddr),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
