# LinkTracker

**LinkTracker** — Telegram-бот, который отслеживает изменения на веб-страницах и сообщает пользователю о них

Реализованы команды `/start`, `/help`, обработка неизвестных команд, а также асинхронная доставка уведомлений об обновлениях ссылок из сервиса **Scrapper** в **Bot** через **Apache Kafka** (с опциональным HTTP/gRPC), transactional outbox на стороне Scrapper, DLQ и повторные попытки на стороне Bot

## Требования

- Go 1.24.1
- Docker и Docker Compose (для Kafka, PostgreSQL и запуска всего стека)
- Для интеграционных тестов с Kafka: доступ к Docker API (как у Testcontainers). На Windows при ошибках линковки `confluent-kafka-go` удобнее запускать тесты в **WSL/Linux** или в CI (образ `golang` на Linux)

## Клонирование и зависимости

```bash
git clone <repo-url>
cd link-tracker
go mod download
```

## Файл `.env`

Создайте файл `.env` в **корне репозитория** (файл уже в `.gitignore`). Минимальные и типичные переменные:

### Обязательные для Bot (локально и в Docker)

| Переменная | Описание |
|------------|----------|
| `TELEGRAM_TOKEN` | Токен Telegram-бота от [@BotFather](https://t.me/BotFather) |

### Bot (локальный `go run`, без Docker)

| Переменная | Пример | Описание |
|------------|--------|----------|
| `TELEGRAM_TOKEN` | — | см. выше |
| `SCRAPPER_URL` | `http://localhost:8081` | HTTP Scrapper |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/bot_db?sslmode=disable` | БД бота |
| `ACCESS_TYPE` | `SQL` или `ORM` | тип доступа к БД |
| `BOT_HTTP_ADDR` | `:8080` | HTTP бота |
| `BOT_GRPC_ADDR` | `:8082` | gRPC бота |
| `SCRAPPER_GRPC_ADDR` | `localhost:8083` | gRPC Scrapper |

### Kafka (Bot и Scrapper)

Используются, когда Scrapper шлёт уведомления через Kafka (по умолчанию так и задано в коде и в `docker-compose`)

| Переменная | Пример (хост после `docker compose up`) | Описание |
|------------|------------------------------------------|----------|
| `KAFKA_BOOTSTRAP_SERVERS` | `localhost:19092,localhost:19093,localhost:19094` | Брокеры |
| `KAFKA_LINK_UPDATES_TOPIC` | `link-updates` | Основной топик событий |
| `KAFKA_DLQ_TOPIC` | `link-updates-dlq` | Dead Letter Queue (только Bot) |
| `KAFKA_CONSUMER_GROUP` | `bot-link-updates` | Группа консьюмера (Bot) |
| `KAFKA_CLIENT_ID` | `bot` / `scrapper` | Идентификатор клиента |
| `KAFKA_MAX_RETRIES` | `3` | Повторы бизнес-обработки перед DLQ (Bot) |
| `KAFKA_SERIALIZATION_FORMAT` | `JSON` (по умолчанию) или `AVRO` | Формат значения в Kafka |
| `SCHEMA_REGISTRY_URL` | `http://localhost:18085` | Нужен при `KAFKA_SERIALIZATION_FORMAT=AVRO` |
| `KAFKA_LINK_UPDATES_SUBJECT` | `link-updates-value` | Subject в Schema Registry для Avro |

### Scrapper (локальный `go run`)

| Переменная | Пример | Описание |
|------------|--------|----------|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/scrapper_db?sslmode=disable` | БД Scrapper (должна отличаться от БД бота) |
| `ACCESS_TYPE` | `SQL` или `ORM` | |
| `BOT_HTTP_URL` | `http://localhost:8080` | HTTP бота (при `NOTIFICATION_TRANSPORT=HTTP`) |
| `BOT_GRPC_ADDR` | `localhost:8082` | gRPC бота (при `NOTIFICATION_TRANSPORT=GRPC`) |
| `NOTIFICATION_TRANSPORT` | `KAFKA` (по умолчанию), `HTTP` или `GRPC` | Канал нотификаций |
| `OUTBOX_ENABLED` | `true` | Transactional outbox (имеет смысл при Kafka) |
| `OUTBOX_PUBLISH_INTERVAL` | `5s` | Период вычитки outbox |
| `OUTBOX_BATCH_SIZE` | `100` | Размер батча outbox |

Переменные для GitHub/StackOverflow и интервалов сканирования см. в `scrapper/internal/config/config.go` и `bot/internal/infrastructure/config/config.go`

## Запуск через Docker Compose

1. Создайте `.env` в корне с минимум:

   ```bash
   TELEGRAM_TOKEN=your_token_here
   ```

2. Запуск:

   ```bash
   docker compose up --build
   ```

3. Логи:

   ```bash
   docker compose logs -f bot scrapper
   ```

4. Остановка и сброс данных Postgres:

   ```bash
   docker compose down
   docker compose down -v
   ```

В compose поднимаются: PostgreSQL, три брокера Kafka + Zookeeper, Schema Registry, **Kafka UI** на [http://localhost:8090](http://localhost:8090), сервис создания топиков, **scrapper** и **bot**

## Локальный запуск (`go run`)

Терминал 1 — Bot:

```bash
go run ./bot/cmd/bot.go
```

Терминал 2 — Scrapper:

```bash
go run ./scrapper/cmd/scrapper.go
```

Нужны заполненный `.env`, доступные Postgres и Kafka (если выбран транспорт Kafka), см. таблицы выше

## Kafka: топики и настройки

Топики создаются сервисом **`kafka-init-topics`** в `docker-compose.yml` (скрипт на базе `kafka-topics`).

### `link-updates` (основной поток нотификаций)

- **Partitions: 3** — запас по параллелизму: несколько консьюмеров в одной группе могут читать разные партиции; при росте нагрузки проще масштабировать Bot
- **Replication factor: 3** — соответствует кластеру из трёх брокеров: допускается отказ одного брокера без потери доступности лидер-реплик при `min.insync.replicas=2`.
- **`min.insync.replicas=2`** — продюсер с `acks=all` (так настроен код) не получит подтверждение, пока запись не попадёт минимум на две реплики, что согласовано с отказоустойчивостью кластера

### `link-updates-dlq` (Dead Letter Queue)

- Те же **partitions** и **replication factor**, чтобы не было узкого места и DLQ переживал тот же отказ брокера, что и основной топик
- **`retention.ms=2592000000`** (~30 суток): «ядовитые» или проблемные сообщения дольше доступны для ручного разбора.

Формат полезной нагрузки в основном топике по умолчанию — **JSON** (`pkg/api.LinkUpdate`), как и при синхронном HTTP. Для бонусного режима **Avro** в compose заданы `SCHEMA_REGISTRY_URL` и `KAFKA_SERIALIZATION_FORMAT=AVRO`.

## Интеграционные тесты (Scrapper → Kafka → Bot)

Используется **один** Kafka Testcontainers на пакет `pkg/kafka` (`TestMain`).

Проверка полного JSON-пути продьюсера Scrapper и консьюмера бота (включая текст, как у пользователя в Telegram):

```bash
go test -tags=integration -timeout 20m -count=1 ./pkg/kafka/... -run TestScraperKafkaProducerToBotConsumerEndToEnd
```

Все интеграционные тесты Kafka (продьюсер, консьюмер, DLQ, e2e):

```bash
go test -tags=integration -timeout 20m -count=1 ./pkg/kafka/...
```

## Проверка в Telegram

После запуска бота отправьте `/start` в чат с ботом — ожидается приветствие.

## Краткая архитектура нотификаций

- Scrapper выбирает транспорт по `NOTIFICATION_TRANSPORT` (по умолчанию **Kafka**).
- При Kafka и включённом outbox события сначала пишутся в таблицу outbox, затем фоновый паблишер отправляет их в Kafka и помечает запись отправленной
- Bot подписан на `link-updates`, валидирует сообщение, при ошибках обработки делает повторы, при исчерпании — отправляет в **DLQ**; ошибки десериализации/валидации сразу в DLQ

## Нагрузочное тестирование


### Без кэша

| Request | RPS | Avg, ms | p50, ms | p99, ms | 200 | 502/504 | 500 | Other statuses | Errors |
|---|---:|---:|---:|---:|---:|---:|---:|---|---:|
| GET /list | 347.15 | 50.28 | 46.34 | 136.78 | 104129 | 0 | 0 | map[] | 16 |
| POST /links | 3.60 | 60.74 | 57.61 | 155.30 | 1079 | 0 | 0 | map[] | 0 |

### С кэшем
| Request | RPS | Avg, ms | p50, ms | p99, ms | 200 | 502/504 | 500 | Other statuses | Errors |
|---|---:|---:|---:|---:|---:|---:|---:|---|---:|
| GET /list | 493.07 | 35.41 | 28.28 | 102.24 | 147904 | 0 | 0 | map[] | 16 |
| POST /links | 4.76 | 41.16 | 35.85 | 108.55 | 1429 | 0 | 0 | map[] | 0 |

### Client-side кэш

| Request | RPS | Avg, ms | p50, ms | p99, ms | 200 | 502/504 | 500 | Other statuses | Errors |
|---|---:|---:|---:|---:|---:|---:|---:|---|---:|
| GET /list | 509.57 | 34.30 | 25.94 | 98.72 | 152829 | 0 | 0 | map[] | 41 |
| POST /links | 5.17 | 38.09 | 32.01 | 104.51 | 1549 | 0 | 0 | map[] | 1 |

### Выводы
По результатам нагрузочного тестирования видно, что добавление кэширования в Valkey заметно улучшило производительность `GET /list`

Без кэша `GET /list` обрабатывался со средней задержкой **50.28 ms** и RPS **347.15**. После включения Valkey cache среднее время ответа снизилось до **35.41 ms**, а RPS вырос до **493.07**. Таким образом, средняя задержка уменьшилась примерно на **30%**, а пропускная способность выросла примерно на **42%**. Также улучшились перцентили: p50 снизился с **46.34 ms** до **28.28 ms**, а p99 — с **136.78 ms** до **102.24 ms**

Client-side caching дал дополнительное, но уже менее значительное ускорение по сравнению с обычным Valkey cache. Для `GET /list` RPS вырос до **509.57**, среднее время ответа снизилось до **34.30 ms**, p50 — до **25.94 ms**, а p99 — до **98.72 ms**. Это показывает, что client-side cache немного снижает накладные расходы на обращения к Valkey, но основной прирост производительности уже достигается за счёт обычного серверного кэширования в Valkey

`POST /links` не кэшируется, так как это операция изменения состояния. Она была включена в нагрузочный тест для имитации реального сценария, где пользователи не только читают список ссылок, но и иногда добавляют новые ссылки. После успешного `POST /links` кэш для соответствующего `Tg-Chat-Id` инвалидируется, чтобы следующий `GET /list` вернул актуальные данные. Улучшение показателей `POST /links` в режимах с кэшем можно объяснить снижением общей нагрузки на PostgreSQL из-за того, что большая часть `GET /list` обрабатывается через Valkey

Ошибки в тестах были единичными по сравнению с общим количеством успешных запросов. Например, для `GET /list` было выполнено более 100–150 тысяч успешных запросов в каждом режиме, при этом количество client-side errors осталось очень маленьким. Ошибок уровня `500`, `502` и `504` зафиксировано не было, что говорит о стабильной работе системы под нагрузкой

Итог: использование Valkey для кэширования `GET /list` действительно повысило производительность Scrapper API. Наиболее заметный эффект был получен при переходе от режима без кэша к обычному Valkey cache. Client-side caching дополнительно улучшил метрики, но прирост оказался меньше, чем от самого факта добавления распределённого in-memory кэша