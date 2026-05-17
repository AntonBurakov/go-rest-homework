# go-rest-homework

Минимальный REST-сервис на Go, сделанный только на стандартной библиотеке.

## Что внутри

- `cmd/server` - точка входа приложения.
- `internal/httpapi` - HTTP handlers, middleware и JSON-ответы.
- `internal/store` - потокобезопасное in-memory хранилище задач.
- `context` используется для таймаута каждого запроса, отмены операций хранилища и graceful shutdown.

## Запуск

```bash
go run ./cmd/server
```

Сервис слушает порт `8080`.

## API

```bash
curl http://localhost:8080/healthz

curl http://localhost:8080/todos

curl -X POST http://localhost:8080/todos \
  -H 'Content-Type: application/json' \
  -d '{"title":"learn go"}'

curl http://localhost:8080/todos/1

curl -X PATCH http://localhost:8080/todos/1 \
  -H 'Content-Type: application/json' \
  -d '{"done":true}'

curl -X DELETE http://localhost:8080/todos/1
```

## Проверка

```bash
go test ./...
```
