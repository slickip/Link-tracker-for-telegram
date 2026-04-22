package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	BotHTTPURL       string
	BotGRPCAddr      string
	ScrapperHTTPAddr string
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

	botHTTPURL := os.Getenv("BOT_HTTP_URL")
	if botHTTPURL == "" {
		botHTTPURL = "http://localhost:8080"
	}

	botGRPCAddr := os.Getenv("BOT_GRPC_ADDR")
	if botGRPCAddr == "" {
		botGRPCAddr = "localhost:8082"
	}

	scrapperHTTPAddr := os.Getenv("SCRAPPER_HTTP_ADDR")
	if scrapperHTTPAddr == "" {
		scrapperHTTPAddr = ":8081"
	}

	scrapperGRPCAddr := os.Getenv("SCRAPPER_GRPC_ADDR")
	if scrapperGRPCAddr == "" {
		scrapperGRPCAddr = ":8083"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:12345@localhost:5432/notesdb?sslmode=disable"
	}

	accessTypeStr := os.Getenv("ACCESS_TYPE")
	if accessTypeStr == "" {
		accessTypeStr = "SQL"
	}

	accessType := AccessType(accessTypeStr)
	if accessType != AccessTypeSQL && accessType != AccessTypeORM {
		log.Fatalf("invalid ACCESS_TYPE: %s", accessTypeStr)
	}

	return &Config{
		BotHTTPURL:       botHTTPURL,
		BotGRPCAddr:      botGRPCAddr,
		ScrapperHTTPAddr: scrapperHTTPAddr,
		ScrapperGRPCAddr: scrapperGRPCAddr,
		DatabaseURL:      databaseURL,
		AccessType:       AccessType(accessType),
	}
}
