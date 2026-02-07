# Build stage
FROM golang:1.25.4-alpine AS builder

WORKDIR /app

# Установка зависимостей
RUN apk add --no-cache git make

# Копирование файлов зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/bot ./cmd/bot

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Копирование бинарного файла
COPY --from=builder /app/bin/bot .

# Создание необходимых директорий
RUN mkdir -p /root/configs /root/logs

# Копирование конфигурации
COPY configs/config.yaml.example /root/configs/config.yaml

# Экспорт порта для вебхука
EXPOSE 8443

# Запуск приложения
CMD ["./bot"]