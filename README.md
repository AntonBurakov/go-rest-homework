# Go Auth Homework

Go-аналог Python/FastAPI сервиса `fastapi-auth-homework`.

Сервис реализует тот же внешний контракт:

- `POST /auth/register` - регистрация пользователя.
- `POST /auth/login` - выдача mock access token.
- `GET /users/me` - текущий пользователь по `Authorization: Bearer ...`.
- `GET /health` - health check для Consul.
- `GET /metrics` - Prometheus-style HTTP-метрики.

## Стек

- Go `net/http`.
- SQLite через `database/sql`.
- Kafka через `segmentio/kafka-go`.
- Consul через HTTP API.
- Vault через HTTP API, KV v2 path `secret/data/fastapi-auth`.
- Пароли через bcrypt.
- Токен совместим с Python-версией: `mock-token-{user_id}.{hmac_sha256}`.

## Структура

```text
cmd/
  server/     HTTP REST service
  consumer/   Kafka consumer
internal/
  config/     env config
  consul/     service registration
  httpapi/    handlers and middleware
  kafka/      producer and consumer
  metrics/    prometheus-style metrics
  security/   bcrypt and HMAC token
  service/    auth use cases
  storage/    SQLite users repository
  vault/      token_secret loader
```

## Запуск инфраструктуры

```bash
docker compose up -d
```

Записать секрет в Vault:

```bash
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=dev-root-token
vault kv put secret/fastapi-auth token_secret=dev-secret
```

## Запуск сервиса

```bash
export VAULT_TOKEN=dev-root-token
go mod tidy
go run ./cmd/server
```

Отдельно можно запустить Kafka consumer:

```bash
go run ./cmd/consumer
```

Для локального запуска без Vault:

```bash
VAULT_ENABLED=false APP_TOKEN_SECRET=dev-secret go run ./cmd/server
```

## API examples

```bash
curl http://localhost:8000/health

curl -X POST http://localhost:8000/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"secret"}'

curl -X POST http://localhost:8000/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"secret"}'

curl http://localhost:8000/users/me \
  -H "Authorization: Bearer $TOKEN"

curl http://localhost:8000/metrics
```

## Переменные окружения

- `SERVICE_PORT`, default `8000`.
- `DB_PATH`, default `app_data/auth.db`.
- `KAFKA_BOOTSTRAP`, default `localhost:9092`.
- `KAFKA_TOPIC`, default `user_events`.
- `KAFKA_CONSUMER_GROUP`, default `fastapi-auth-group`.
- `CONSUL_ENABLED`, default `true`.
- `CONSUL_HTTP_ADDR`, default `http://localhost:8500`.
- `SERVICE_NAME`, default `fastapi-auth`.
- `SERVICE_ID`, default `fastapi-auth-8000`.
- `SERVICE_ADDRESS`, default `127.0.0.1`.
- `SERVICE_HEALTH_CHECK_URL`, default `http://host.docker.internal:8000/health`.
- `VAULT_ENABLED`, default `true`.
- `VAULT_ADDR`, default `http://localhost:8200`.
- `VAULT_TOKEN`.
- `VAULT_TOKEN_SECRET_PATH`, default `secret/data/fastapi-auth`.
- `APP_TOKEN_SECRET`, используется только при `VAULT_ENABLED=false`.

## Эквивалентность Python-сервису

- FastAPI routes заменены на `net/http` handlers.
- SQLAlchemy repository заменён на `database/sql` repository.
- `passlib[bcrypt]` заменён на `golang.org/x/crypto/bcrypt`.
- `kafka-python` producer/consumer заменены на `segmentio/kafka-go`.
- Consul и Vault, как и в Python, вызываются через HTTP API.
- Middleware сохраняет поведение Python-версии: создаёт/пробрасывает `X-Trace-Id`, пишет structured logs, считает HTTP-метрики.
- Событие Kafka сохраняет форму Python-сервиса: `event_id`, `event_name=user_registered`, `user_id`, `email`, `trace_id`.

## Различия Go и Python

- В Python инфраструктура встраивалась через dependency injection FastAPI; в Go зависимости явно собираются в `cmd/server/main.go`.
- В Python request lifecycle управляется FastAPI/Starlette; в Go таймауты и graceful shutdown задаются через `context` и `http.Server`.
- В Python схемы Pydantic валидируют email автоматически; в Go валидация описана явно в handler.
- В Python ORM скрывает SQL; в Go слой `storage` использует явные SQL-запросы через `database/sql`.
- В Go ошибки возвращаются значениями, поэтому HTTP-слой явно сопоставляет service errors со статусами `409`, `401`, `500`.

## Проверка

```bash
go test ./...
```
