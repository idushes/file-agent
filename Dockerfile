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

# Собираем приложение для целевой архитектуры
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETARCH
RUN echo "Building on $BUILDPLATFORM for $TARGETPLATFORM"
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -installsuffix cgo -o main .

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

# Настраиваем права доступа
RUN chown -R appuser:appgroup /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Открываем порт
EXPOSE 8080

# Устанавливаем переменные окружения
ENV PORT=8080
ENV S3_ENDPOINT=https://s3.lisacorp.com
ENV S3_ACCESS_KEY=dushes
ENV S3_SECRET_KEY=lsagdfo43qwfoylasdgf4qy9203w7rey
ENV S3_BUCKET=files
ENV MAX_FILE_SIZE=104857600

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/health || exit 1

# Запускаем приложение
CMD ["./main"] 