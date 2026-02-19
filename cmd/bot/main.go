package main

import (
	"log"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/commands"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/dispatch"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/adapters"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
)

func main() {
	cfg := config.MustLoad()

	bot, err := adapters.NewBot(cfg.TelegramToken)
	if err != nil {
		log.Fatal("failed to create bot", err)
	}

	if err := bot.SetCommands(); err != nil {
		log.Println("failed to set bot commands:", err)
	}

	startCmd := commands.NewStartCommand()
	helpCmd := commands.NewHelpCommand()
	unknownCmd := commands.NewUnknownCommand()

	dispatcher := dispatch.NewDispatcher(
		[]domain.Command{
			startCmd,
			helpCmd,
		},
		unknownCmd,
	)

	log.Println("bot started successfully")

	updates := bot.ListenUpdates()

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		text := update.Message.Text

		log.Printf("received message: chat_id=%d text=%s", chatID, text)

		cmd := dispatcher.Dispatch(text)

		response, err := cmd.Execute(chatID)
		if err != nil {
			log.Println("command execution error:", err)
			continue
		}

		if err := bot.SendMessage(chatID, response); err != nil {
			log.Println("failed to send message:", err)
		}
	}

}
