package handlers

import (
	"context"
	"encoding/json"
	"file-agent/internal/storage"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// FileHandler содержит обработчики для работы с файлами
type FileHandler struct {
	storage     *storage.S3Storage
	maxFileSize int64
}

// NewFileHandler создает новый FileHandler
func NewFileHandler(s3Storage *storage.S3Storage, maxFileSize int64) *FileHandler {
	return &FileHandler{
		storage:     s3Storage,
		maxFileSize: maxFileSize,
	}
}

// UploadResponse структура ответа при загрузке файла
type UploadResponse struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

// ErrorResponse структура для ошибок
type ErrorResponse struct {
	Error string `json:"error"`
}

// UploadFile обрабатывает загрузку файлов
func (fh *FileHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		return
	}

	// Ограничиваем размер загружаемого файла
	r.ParseMultipartForm(fh.maxFileSize)

	file, header, err := r.FormFile("file")
	if err != nil {
		fh.writeError(w, "Failed to get file from request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Проверяем размер файла
	if header.Size > fh.maxFileSize {
		fh.writeError(w, fmt.Sprintf("File size (%d bytes) exceeds maximum allowed size (%d bytes)", header.Size, fh.maxFileSize), http.StatusRequestEntityTooLarge)
		return
	}

	// Получаем информацию о том, кто загружает файл (опционально)
	uploadedBy := r.FormValue("uploaded_by")

	// Генерируем UUID для файла
	fileID := uuid.New().String()

	// Сохраняем файл в S3
	ctx := context.Background()
	err = fh.storage.SaveFile(ctx, fileID, header.Filename, file, header.Size, uploadedBy)
	if err != nil {
		fh.writeError(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
		return
	}

	// Возвращаем ответ
	response := UploadResponse{
		ID:       fileID,
		Filename: header.Filename,
		URL:      fmt.Sprintf("/%s", fileID),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// DownloadFile обрабатывает скачивание файлов
func (fh *FileHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		return
	}

	vars := mux.Vars(r)
	fileID := vars["id"]

	if fileID == "" {
		fh.writeError(w, "File ID is required", http.StatusBadRequest)
		return
	}

	// Получаем файл из S3
	ctx := context.Background()
	fileReader, metadata, err := fh.storage.GetFile(ctx, fileID)
	if err != nil {
		fh.writeError(w, "File not found", http.StatusNotFound)
		return
	}
	defer fileReader.Close()

	// Определяем MIME-тип файла
	contentType := storage.GetContentType(metadata.Filename)

	// Устанавливаем заголовки
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", metadata.Filename))

	// Копируем содержимое файла в ответ
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, fileReader)
	if err != nil {
		// Логируем ошибку, но не можем уже изменить статус ответа
		fmt.Printf("Error sending file: %v\n", err)
	}
}

// GetFileMetadata обрабатывает получение метаданных файла
func (fh *FileHandler) GetFileMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		return
	}

	vars := mux.Vars(r)
	fileID := vars["id"]

	if fileID == "" {
		fh.writeError(w, "File ID is required", http.StatusBadRequest)
		return
	}

	// Получаем метаданные файла из S3
	ctx := context.Background()
	metadata, err := fh.storage.GetFileMetadata(ctx, fileID)
	if err != nil {
		fh.writeError(w, "File not found", http.StatusNotFound)
		return
	}

	// Возвращаем метаданные
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metadata)
}

// writeError записывает ошибку в ответ
func (fh *FileHandler) writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
