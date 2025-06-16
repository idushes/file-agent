# Стадия сборки
FROM golang:1.21-alpine AS builder

# Устанавливаем git для загрузки зависимостей
RUN apk add --no-cache git

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем go.mod и go.sum
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Финальная стадия - минимальный образ
FROM alpine:latest

# Добавляем ca-certificates для HTTPS запросов
RUN apk --no-cache add ca-certificates

# Создаем пользователя для запуска приложения
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Создаем директорию для приложения
WORKDIR /app

# Копируем собранное приложение
COPY --from=builder /app/main .

# Создаем директорию для uploads
RUN mkdir -p /app/uploads && \
    chown -R appuser:appgroup /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Открываем порт
EXPOSE 8080

# Устанавливаем переменную окружения для хранения файлов
ENV STORAGE_PATH=/app/uploads

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Запускаем приложение
CMD ["./main"] 