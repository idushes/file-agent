package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// FileHandler содержит обработчики для работы с файлами
type FileHandler struct {
	storagePath string
}

// NewFileHandler создает новый FileHandler
func NewFileHandler(storagePath string) *FileHandler {
	return &FileHandler{
		storagePath: storagePath,
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

	// Ограничиваем размер загружаемого файла (100MB)
	r.ParseMultipartForm(100 << 20) // 100MB

	file, header, err := r.FormFile("file")
	if err != nil {
		fh.writeError(w, "Failed to get file from request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Генерируем UUID для файла
	fileID := uuid.New().String()

	// Получаем расширение файла
	ext := filepath.Ext(header.Filename)
	fileName := fileID + ext

	// Создаем файл на диске
	filePath := filepath.Join(fh.storagePath, fileName)
	dst, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create file %s: %v", filePath, err)
		fh.writeError(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Копируем содержимое файла
	_, err = io.Copy(dst, file)
	if err != nil {
		log.Printf("Failed to copy file content: %v", err)
		fh.writeError(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Сохраняем метаданные файла
	metadata := FileMetadata{
		ID:       fileID,
		Filename: header.Filename,
		Path:     filePath,
		Size:     header.Size,
	}

	if err := fh.saveMetadata(metadata); err != nil {
		log.Printf("Failed to save metadata: %v", err)
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

	// Загружаем метаданные файла
	metadata, err := fh.loadMetadata(fileID)
	if err != nil {
		fh.writeError(w, "File not found", http.StatusNotFound)
		return
	}

	// Проверяем существование файла
	if _, err := os.Stat(metadata.Path); os.IsNotExist(err) {
		fh.writeError(w, "File not found", http.StatusNotFound)
		return
	}

	// Определяем MIME-тип файла
	ext := filepath.Ext(metadata.Filename)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Устанавливаем заголовки
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", metadata.Filename))

	// Отдаем файл
	http.ServeFile(w, r, metadata.Path)
}

// writeError записывает ошибку в ответ
func (fh *FileHandler) writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
