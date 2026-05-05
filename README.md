# LinkTracker

**LinkTracker** – Telegram-бот, который отслеживает изменения на веб-страницах и оперативно информирует пользователя о них

В рамках первого домашнего задания реализована базовая обработка команд:

- /start — приветственное сообщение
- /help — список доступных команд
- неизвестные команды — сообщение об ошибке

Бот использует лп и автоматически регистрирует команды в Telegram при запуске


## Требования

- Go 1.24.1
- Telegram Bot Token 


## Настройка

### 1. Клонировать проект

git clone [<repo-url>](https://gitlab.education.tbank.ru/backend-academy-go-2025/homework-forks/i.poltorakova-73112/link-tracker.git)
cd link-tracker


### 2. Создать файл .env

В корне проекта создать файл .env:

TELEGRAM_TOKEN=your_telegram_token_here


### 3. Установить зависимости

go mod tidy


## Запуск

Из корня проекта выполнить:

go run ./bot/cmd/bot.go

После запуска в консоли появится сообщение о старте бота


## Проверка

Открыть Telegram, найти своего бота и отправить:

/start

Ожидается приветственное сообщение

## Запуск через Docker Compose

1. В корне проекта создайте файл `.env` (минимум):
```bash
TELEGRAM_TOKEN=your_telegram_token_here
```

2. Запустите сервисы:
```bash
docker compose up --build
```

3. Посмотрите логи:
```bash
docker compose logs -f bot scrapper
```

Остановить:
```bash
docker compose down
```

Сбросить данные PostgreSQL (осторожно, удалятся volume):
```bash
docker compose down -v
```

### Что в `.env` нужно для успешного запуска

Для `docker compose` минимум нужен только `TELEGRAM_TOKEN` — остальное задаётся в `docker-compose.yml`.

## Локальный запуск через `go run`

Пакеты запускаются отдельными entrypoint’ами:
```bash
go run ./bot/cmd/bot.go
go run ./scrapper/cmd/scrapper.go
```

В `.env` для локального запуска должны быть:
- `TELEGRAM_TOKEN`
- `SCRAPPER_URL`
- `DATABASE_URL` (для *конкретного* процесса: `bot` и `scrapper` должны быть на разных базах, например `.../bot_db` и `.../scrapper_db`)
- `ACCESS_TYPE` (`SQL` или `ORM`)

Плюс адреса:
- Для `bot`: `BOT_HTTP_ADDR`, `BOT_GRPC_ADDR`
- Для `scrapper`: `SCRAPPER_HTTP_ADDR`, `SCRAPPER_GRPC_ADDR`, `BOT_HTTP_URL`, `BOT_GRPC_ADDR`
