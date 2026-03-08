package main

import (
	"log"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/services"
	router "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/handlers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/repositories"
)

func main() {

	chatRepo := repositories.NewInMemoryChatRepository()
	linkRepo := repositories.NewInMemoryLinkRepository()

	chatService := services.NewChatService(chatRepo)
	linkService := services.NewLinkService(linkRepo, chatRepo)

	chatHandler := handlers.NewChatHandler(chatService)
	linkHandler := handlers.NewLinkHandler(linkService)

	router := router.NewRouter(chatHandler, linkHandler)

	log.Println("Scrapper started on :8081")

	log.Fatal(http.ListenAndServe(":8081", router))
}
