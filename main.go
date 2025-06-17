package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"file-agent/internal/handlers"
	"file-agent/internal/middleware"
	"file-agent/internal/storage"

	"github.com/gorilla/mux"
)

func main() {
	// Получаем порт из environment variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // значение по умолчанию
	}

	// Получаем S3 настройки из environment variables
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	if s3Endpoint == "" {
		s3Endpoint = "https://s3.lisacorp.com" // значение по умолчанию
	}

	accessKey := os.Getenv("S3_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "dushes" // значение по умолчанию
	}

	secretKey := os.Getenv("S3_SECRET_KEY")
	if secretKey == "" {
		secretKey = "lsagdfo43qwfoylasdgf4qy9203w7rey" // значение по умолчанию
	}

	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		bucket = "files" // значение по умолчанию
	}

	// Получаем максимальный размер файла из environment variable
	maxFileSizeStr := os.Getenv("MAX_FILE_SIZE")
	var maxFileSize int64 = 100 << 20 // 100MB по умолчанию
	if maxFileSizeStr != "" {
		if size, err := strconv.ParseInt(maxFileSizeStr, 10, 64); err == nil && size > 0 {
			maxFileSize = size
		} else {
			log.Printf("Invalid MAX_FILE_SIZE value: %s, using default: %d bytes", maxFileSizeStr, maxFileSize)
		}
	}

	// Создаем S3 storage
	s3Storage, err := storage.NewS3Storage(s3Endpoint, accessKey, secretKey, bucket)
	if err != nil {
		log.Fatalf("Failed to initialize S3 storage: %v", err)
	}

	// Инициализируем хендлеры
	fileHandler := handlers.NewFileHandler(s3Storage, maxFileSize)

	// Настраиваем роутер
	r := mux.NewRouter()

	// Применяем CORS middleware
	r.Use(middleware.CORSMiddleware)

	// Health check роуты для Kubernetes (должны быть первыми)
	r.HandleFunc("/health", healthCheck).Methods("GET")
	r.HandleFunc("/ready", readinessCheck).Methods("GET")

	// API роуты
	r.HandleFunc("/", fileHandler.UploadFile).Methods("POST", "OPTIONS")
	r.HandleFunc("/{id}", fileHandler.DownloadFile).Methods("GET", "OPTIONS")
	r.HandleFunc("/metadata/{id}", fileHandler.GetFileMetadata).Methods("GET", "OPTIONS")

	// Настраиваем сервер
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запускаем сервер в горутине
	go func() {
		log.Printf("Server starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Ожидаем сигнал для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	// Graceful shutdown с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// Health check для Kubernetes liveness probe
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Readiness check для Kubernetes readiness probe
func readinessCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}
