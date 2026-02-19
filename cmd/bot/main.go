package bot

import (
	"log"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/adapters"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
)

func main() {
	cfg := config.MustLoad()

	bot, err := adapters.NewBot(cfg.TelegramToken)
	if err != nil {
		log.Fatal(err)
	}
}
