# Order Processing Service

демонстрационный микросервис на Go, предназначенный для обработки, хранения и отображения данных о заказах. Сервис реализует пайплайн получения данных из очереди сообщений **Kafka**, сохранения их в базу данных **PostgreSQL** и кэширования в памяти для быстрого доступа через REST API и простой веб-интерфейс.

## 🛠️ Технологический стек

-   **Язык:** Go 1.24+
-   **База данных:** PostgreSQL
-   **Очередь сообщений:** Apache Kafka
-   **HTTP Framework:** Gin
-   **Драйвер PostgreSQL:** pgx
-   **Работа с Kafka:** segmentio/kafka-go
-   **Конфигурация:** godotenv, cleanenv
-   **Контейнеризация:** Docker, Docker Compose

## 🚀 Начало работы


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

## 📁 Структура Order-service

```
.
├── cmd/app/              # Точка входа приложения (main.go)
├── internal/
│   ├── cache/            # Реализация LRU-кеша
│   ├── config/           # Управление конфигурацией (.env)
│   ├── handlers/         # HTTP-обработчики (Gin)
│   ├── kafka/            # Kafka-консьюмер и продюсер
│   ├── models/           # Структуры данных (заказы, платежи и т.д.)
│   ├── repository/       # Слой доступа к данным (PostgreSQL)
│   ├── router/           # Настройка маршрутов HTTP
│   ├── service/          # Слой бизнес-логики, оркестратор
│   └── validator/        # Валидация и проверка сообщений из Kafka
├── migrations/           # SQL-скрипты для миграции базы данных
├── scripts/              # Вспомогательные скрипты (например, для отправки данных)
├── web/static/           # Веб-интерфейс
├── .env.example          # Шаблон файла с переменными окружения
├── docker-compose.yml    # Конфигурация для запуска инфраструктуры
├── Dockerfile            # Инструкции для сборки Docker-образа приложения
└── go.mod                # Зависимости проекта
```
