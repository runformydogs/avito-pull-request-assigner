FROM golang:1.23.0-alpine AS builder

WORKDIR /app

# Копируем файлы модулей и загружаем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/main.go

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Копируем бинарник из builder stage
COPY --from=builder /app/main .

# Копируем миграции
COPY --from=builder /app/internal/lib/migrator/migrations ./internal/lib/migrator/migrations

# Создаем не-root пользователя для безопасности
RUN adduser -D -s /bin/sh appuser
USER appuser

EXPOSE 8080

CMD ["./main"]