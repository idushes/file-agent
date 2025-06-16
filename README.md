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
curl -X POST -F "file=@example.txt" http://localhost:8080/
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

## Запуск

### Локально

```bash
go mod tidy
go run main.go
```

### Docker

#### Простой способ (Makefile)

```bash
# Собрать образ
make build

# Протестировать образ
make test

# Запустить контейнер локально
make run

# Остановить контейнер
make stop

# Собрать и загрузить на Docker Hub
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
  dushes/file-agent:latest
```

## API Endpoints

- `POST /` - Загрузка файла
- `GET /{id}` - Скачивание файла по ID
- `GET /health` - Health check
- `GET /ready` - Readiness check
- `OPTIONS /` - CORS preflight для загрузки
- `OPTIONS /{id}` - CORS preflight для скачивания 