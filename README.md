# Order Processing Service

демонстрационный микросервис на Go, предназначенный для обработки, хранения и отображения данных о заказах. Сервис реализует пайплайн получения данных из очереди сообщений **Kafka**, сохранения их в базу данных **PostgreSQL** и кэширования в памяти для быстрого доступа через REST API и простой веб-интерфейс.

##  Технологический стек

-   **Язык:** Go 1.24
-   **База данных:** PostgreSQL
-   **Очередь сообщений:** Apache Kafka
-   **HTTP Framework:** Gin
-   **Драйвер PostgreSQL:** pgx
-   **Работа с Kafka:** segmentio/kafka-go
-   **Конфигурация:** godotenv, cleanenv
-   **Контейнеризация:** Docker, Docker Compose

##  Начало работы


### Установка и запуск

1.  **Клонируйте репозиторий:**
    ```bash
    git clone https://github.com/stpnv0/WB-Tech-School-L0.git
    cd order-service
    ```

2.  **Настройте окружение:**
    Скопируйте файл с примером переменных окружения или заполните его своими данными.
    ```bash
    cp .env.example .env
    ```

3.  **Поднять контейнеры:**
    ```bash
    docker compose up -d --build
    ```

4.  **Отправьте тестовые сообщения в kafka через скрипт - будет отправлено 4 сообщения: 2 валидных(b563feb7b2b84b6test, ok_multi_items_1), 2 невалидных (mismatch_tx, bad_json):**
    ```bash
    chmod +x scripts/send_test_orders.sh
    ./scripts/send_test_orders.sh
    ```
5. **Зайдите на UI (http://localhost:8081) и введите order_uid из скрипта, например:** 
    ```
    b563feb7b2b84b6test
    ```
    ```
    ok_multi_items_1
    ```
#### **P.s. Невалидные JSON'ы (mismatch_tx, bad_json) отправлены в DLQ**

---

## Профилирование и оптимизация

Сервис был профилирован и оптимизирован по CPU и памяти с помощью `pprof`, `benchstat`, `trace` и нагрузочного тестирования.

Результаты (10000 заказов в БД, 50 конкурентных воркеров, 35s):

| Cache size    | Hit rate | RPS         | p50     | p95      | p99     |
|---------------|----------|-------------|---------|----------|---------|
| Before (1000) | ~10%     | 762         | 67ms    | 124ms    | 228ms   |
| **1000**      | ~10%     | **5 101**   | 7.6ms   | 21.2ms   | 44.7ms  |
| **2000**      | ~20%     | **5 691**   | 7.4ms   | 19.4ms   | 31.0ms  |
| **3000**      | ~30%     | **6 960**   | 6.1ms   | 15.4ms   | 27.5ms  |

При том же размере кэша (1000) RPS вырос с 762 до 5101 (**x6.7**) благодаря оптимизациям кода и БД.

Основные изменения:
- Убран `gin.Logger` middleware (84% cum allocs)
- Убран `slog.With()` с hot path (-20.5MB allocs)
- Заменён `encoding/json` на `goccy/go-json` (-24MB allocs)
- Добавлен индекс на `items.order_id` (Seq Scan 32ms → Index Scan 0.18ms)
- Настроен `pgxpool.MaxConns`

Подробное описание процесса, инструментов и обоснование каждого изменения: **[PROFILING.md](PROFILING.md)**

### Запуск профилирования

```bash
cd order-service

# 1. Поднять инфраструктуру
docker compose up -d --build

# 2. Засеять тестовые данные (10000 заказов)
DB_HOST=localhost DB_PORT=5433 go run ./cmd/seed/ -n 10000

# 3. Нагрузочное тестирование
go run ./cmd/loadtest/ -url http://localhost:8081 -concurrency 50 -duration 35s

# 4. Снять профили (пока идёт нагрузка)
curl -o profiles/cpu.prof 'http://localhost:6060/debug/pprof/profile?seconds=30'
curl -o profiles/allocs.prof http://localhost:6060/debug/pprof/allocs

# 5. Прочитать профили
go tool pprof -text -cum profiles/allocs.prof
go tool pprof -text profiles/cpu.prof
```

### Бенчмарки

```bash
cd order-service
go test -bench=. -benchmem -count=10 ./internal/cache/ ./internal/service/ ./internal/handlers/ ./internal/validator/
```

---

##  Структура Order-service

```
.
├── cmd/
│   ├── app/              # Точка входа приложения (main.go, debug.go — pprof сервер)
│   ├── loadtest/         # Утилита нагрузочного тестирования
│   └── seed/             # Генератор тестовых данных
├── internal/
│   ├── cache/            # Реализация LRU-кеша + бенчмарки
│   ├── config/           # Управление конфигурацией (.env)
│   ├── handlers/         # HTTP-обработчики (Gin) + бенчмарки
│   ├── kafka/            # Kafka-консьюмер и продюсер
│   ├── models/           # Структуры данных (заказы, платежи и т.д.)
│   ├── repository/       # Слой доступа к данным (PostgreSQL)
│   ├── router/           # Настройка маршрутов HTTP
│   ├── service/          # Слой бизнес-логики + бенчмарки
│   └── validator/        # Валидация сообщений из Kafka + бенчмарки
├── benchmarks/           # Результаты бенчмарков (before/after)
├── profiles/             # pprof-профили (CPU, allocs, heap, trace)
├── migrations/           # SQL-скрипты для миграции базы данных
├── scripts/              # Вспомогательные скрипты
├── web/static/           # Веб-интерфейс
├── PROFILING.md          # Детальное описание профилирования и оптимизаций
├── .env.example          # Шаблон файла с переменными окружения
├── docker-compose.yml    # Конфигурация для запуска инфраструктуры
├── Dockerfile            # Инструкции для сборки Docker-образа
└── go.mod                # Зависимости проекта
```
