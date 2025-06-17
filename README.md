# File Agent

Простой Go-сервер для загрузки и скачивания файлов с использованием S3-совместимого хранилища (Minio).

## Функциональность

- Загрузка файлов через `multipart/form-data` на URL `/`
- Скачивание файлов по GET запросу на `/{uuid4}`
- Хранение файлов в S3-совместимом хранилище (Minio)
- Поддержка CORS
- Health checks для Kubernetes
- Graceful shutdown

## Использование

### Загрузка файла

```bash
# Загрузка файла без указания загрузившего
curl -X POST -F "file=@example.txt" http://localhost:8080/

# Загрузка файла с указанием загрузившего
curl -X POST -F "file=@example.txt" -F "uploaded_by=john.doe" http://localhost:8080/
```

Ответ:
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "filename": "example.txt",
  "url": "/123e4567-e89b-12d3-a456-426614174000"
}
```

### Скачивание файла

```bash
curl -O http://localhost:8080/123e4567-e89b-12d3-a456-426614174000
```

### Получение метаданных файла

```bash
curl http://localhost:8080/metadata/123e4567-e89b-12d3-a456-426614174000
```

Ответ:
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "filename": "example.txt",
  "size": 1024,
  "uploaded_at": "2025-06-16T05:30:00Z",
  "uploaded_by": "john.doe"
}
```

### Аналитика файлов

```bash
curl http://localhost:8080/analytics
```

Ответ:
```json
{
  "total_files": 150,
  "total_size": 52428800,
  "periods": [
    {
      "period": "last_day",
      "file_count": 12,
      "total_size": 3145728
    },
    {
      "period": "last_week", 
      "file_count": 45,
      "total_size": 15728640
    },
    {
      "period": "last_month",
      "file_count": 120,
      "total_size": 41943040
    }
  ],
  "top_users": [
    {
      "user": "john.doe",
      "file_count": 25,
      "total_size": 15728640
    },
    {
      "user": "jane.smith",
      "file_count": 18,
      "total_size": 12582912
    },
    {
      "user": "anonymous",
      "file_count": 107,
      "total_size": 24117248
    }
  ]
}
```

## Ограничение размера файлов

По умолчанию максимальный размер загружаемого файла составляет 100MB (104857600 байт). Вы можете настроить это значение с помощью переменной окружения `MAX_FILE_SIZE`.

### Примеры настройки лимита:

```bash
# Ограничить до 10MB
export MAX_FILE_SIZE=10485760

# Ограничить до 1MB  
export MAX_FILE_SIZE=1048576

# Ограничить до 500KB
export MAX_FILE_SIZE=512000
```

При превышении лимита сервер вернет ошибку `413 Request Entity Too Large`:

```json
{
  "error": "File size (1048576 bytes) exceeds maximum allowed size (512000 bytes)"
}
```

## Запуск

### Локально

```bash
go mod tidy
go run main.go
```

### Docker

#### Простой способ (Makefile)

```bash
# Собрать мультиплатформенный образ (amd64+arm64)
make build

# Собрать образ только для Kubernetes (amd64)
make build-amd64

# Протестировать образ
make test

# Запустить контейнер локально
make run

# Остановить контейнер
make stop

# Собрать и загрузить мультиплатформенный образ на Docker Hub
make all

# Показать справку
make help
```

#### Ручной способ

```bash
docker build -t file-agent .
docker run -p 8082:8082 \
  -e PORT=8082 \
  -e S3_ENDPOINT=https://s3.example.com \
  -e S3_ACCESS_KEY=dushes \
  -e S3_SECRET_KEY=dsadfghgjfhdsadfsghh \
  -e S3_BUCKET=files \
  -e MAX_FILE_SIZE=104857600 \
  file-agent
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: file-agent
spec:
  replicas: 3
  selector:
    matchLabels:
      app: file-agent
  template:
    metadata:
      labels:
        app: file-agent
    spec:
      containers:
      - name: file-agent
        image: file-agent:latest
        ports:
        - containerPort: 8082
        env:
        - name: PORT
          value: "8082"
        - name: S3_ENDPOINT
          value: "https://s3.example.com"
        - name: S3_ACCESS_KEY
          value: "dushes"
        - name: S3_SECRET_KEY
          value: "safdgfhgjfhgfdasfghhffds"
        - name: S3_BUCKET
          value: "files"
        - name: MAX_FILE_SIZE
          value: "104857600"
        livenessProbe:
          httpGet:
            path: /health
            port: 8082
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8082
          initialDelaySeconds: 5
          periodSeconds: 10

---
apiVersion: v1
kind: Service
metadata:
  name: file-agent-service
spec:
  selector:
    app: file-agent
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8082
  type: LoadBalancer
```

## Переменные окружения

- `PORT` - порт для запуска сервера (по умолчанию: `8080`)
- `S3_ENDPOINT` - URL для S3-совместимого хранилища (по умолчанию: `https://s3.example.com`)
- `S3_ACCESS_KEY` - ключ доступа к S3 (по умолчанию: `dushes`)
- `S3_SECRET_KEY` - секретный ключ S3 (по умолчанию: `dfsghjkfdsafghjfds`)
- `S3_BUCKET` - имя S3 бакета (по умолчанию: `files`)
- `MAX_FILE_SIZE` - максимальный размер загружаемого файла в байтах (по умолчанию: `104857600` = 100MB)

## Docker Hub

### Загрузка образа на Docker Hub

1. Войдите в Docker Hub:
```bash
docker login
```

2. Соберите и загрузите образ:
```bash
# Загрузить с версией latest
make all

# Загрузить с определенной версией
make all VERSION=v1.0.0

# Загрузить под другим пользователем
make all DOCKER_USERNAME=myuser VERSION=v1.0.0
```

### Использование готового образа

```bash
# Скачать образ с Docker Hub
docker pull dushes/file-agent:latest

# Запустить контейнер
docker run -p 8082:8082 \
  -e PORT=8082 \
  -e S3_ENDPOINT=https://s3.example.com \
  -e S3_ACCESS_KEY=your_key \
  -e S3_SECRET_KEY=your_secret \
  -e S3_BUCKET=files \
  -e MAX_FILE_SIZE=104857600 \
  dushes/file-agent:latest
```

## API Endpoints

- `POST /` - Загрузка файла (с опциональным полем `uploaded_by`)
- `GET /{id}` - Скачивание файла по ID
- `GET /metadata/{id}` - Получение метаданных файла
- `GET /analytics` - Аналитика файлов (статистика по периодам и пользователям)
- `GET /health` - Health check
- `GET /ready` - Readiness check
- `OPTIONS /` - CORS preflight для загрузки
- `OPTIONS /{id}` - CORS preflight для скачивания
- `OPTIONS /metadata/{id}` - CORS preflight для метаданных
- `OPTIONS /analytics` - CORS preflight для аналитики 