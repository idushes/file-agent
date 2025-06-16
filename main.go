package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"file-agent/internal/handlers"
	"file-agent/internal/middleware"

	"github.com/gorilla/mux"
)

func main() {
	// Получаем путь для хранения файлов из environment variable
	storagePath := os.Getenv("STORAGE_PATH")
	if storagePath == "" {
		storagePath = "./uploads" // значение по умолчанию
	}

	// Создаем директорию для хранения файлов если она не существует
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}

	// Инициализируем хендлеры
	fileHandler := handlers.NewFileHandler(storagePath)

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

	// Настраиваем сервер
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запускаем сервер в горутине
	go func() {
		log.Println("Server starting on :8080")
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
