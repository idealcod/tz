# Этап сборки
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем файлы go.mod и go.sum
COPY go.mod go.sum ./
# Устанавливаем переменную окружения для загрузки модулей
ENV GOPROXY=https://proxy.golang.org
RUN go env -w GO111MODULE=on && go mod download

# Копируем весь исходный код
COPY . .

# Диагностика: проверяем наличие файлов в директории internal/
RUN ls -la && ls -la cmd/ && ls -la internal/ && ls -la internal/config/ && ls -la internal/handler/ && ls -la internal/model/ && ls -la internal/repository/ && ls -la internal/service/

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/main ./cmd/main.go

# Финальный этап
FROM alpine:latest

WORKDIR /app

# Копируем бинарный файл из этапа сборки
COPY --from=builder /app/main .

# Копируем .env и миграции
COPY .env .
COPY internal/migrations/ ./internal/migrations/

# Открываем порт
EXPOSE 8080

# Команда для запуска приложения
CMD ["./main"]