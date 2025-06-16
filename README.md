# File Agent

Простой Go-сервер для загрузки и скачивания файлов.

## Функциональность

- Загрузка файлов через `multipart/form-data` на URL `/`
- Скачивание файлов по GET запросу на `/{uuid4}`
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

```bash
docker build -t file-agent .
docker run -p 8080:8080 -e STORAGE_PATH=/app/uploads file-agent
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
        - containerPort: 8080
        env:
        - name: STORAGE_PATH
          value: "/app/uploads"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        volumeMounts:
        - name: storage
          mountPath: /app/uploads
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: file-storage-pvc
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
      targetPort: 8080
  type: LoadBalancer
```

## Переменные окружения

- `STORAGE_PATH` - путь для хранения файлов (по умолчанию: `./uploads`)

## API Endpoints

- `POST /` - Загрузка файла
- `GET /{id}` - Скачивание файла по ID
- `GET /health` - Health check
- `GET /ready` - Readiness check
- `OPTIONS /` - CORS preflight для загрузки
- `OPTIONS /{id}` - CORS preflight для скачивания 