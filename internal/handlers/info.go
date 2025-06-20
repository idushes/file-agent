package handlers

import (
	"encoding/json"
	"net/http"
)

// InfoHandler обрабатывает запросы на получение информации о сервисе
type InfoHandler struct{}

// NewInfoHandler создает новый экземпляр InfoHandler
func NewInfoHandler() *InfoHandler {
	return &InfoHandler{}
}

// InfoResponse структура ответа с информацией о сервисе
type InfoResponse struct {
	Service     string                  `json:"service"`
	Version     string                  `json:"version"`
	Description string                  `json:"description"`
	Endpoints   map[string]EndpointInfo `json:"endpoints"`
}

// EndpointInfo структура с информацией об эндпоинте
type EndpointInfo struct {
	Method      string            `json:"method"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Response    interface{}       `json:"response,omitempty"`
}

// GetInfo возвращает информацию о сервисе и его эндпоинтах
func (h *InfoHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	info := InfoResponse{
		Service:     "File Agent Service",
		Version:     "1.0.0",
		Description: "Сервис для загрузки, хранения и получения файлов с использованием S3-совместимого хранилища",
		Endpoints: map[string]EndpointInfo{
			"POST /": {
				Method:      "POST",
				Description: "Загрузить файл в хранилище",
				Parameters: map[string]string{
					"file":        "Файл для загрузки (multipart/form-data)",
					"uploaded_by": "Имя пользователя или идентификатор загрузившего (опционально)",
				},
				Response: map[string]interface{}{
					"id":       "Уникальный идентификатор файла",
					"filename": "Оригинальное имя файла",
					"url":      "Относительный URL для скачивания файла",
				},
			},
			"GET /{id}": {
				Method:      "GET",
				Description: "Скачать файл по его идентификатору",
				Parameters: map[string]string{
					"id": "Уникальный идентификатор файла",
				},
				Headers: map[string]string{
					"Content-Type":        "MIME-тип файла",
					"Content-Disposition": "Оригинальное имя файла",
				},
			},
			"GET /metadata/{id}": {
				Method:      "GET",
				Description: "Получить метаданные файла без его скачивания",
				Parameters: map[string]string{
					"id": "Уникальный идентификатор файла",
				},
				Response: map[string]interface{}{
					"id":          "Уникальный идентификатор файла",
					"filename":    "Оригинальное имя файла",
					"size":        "Размер файла в байтах",
					"uploaded_at": "Время загрузки файла в формате RFC3339",
					"uploaded_by": "Имя пользователя или идентификатор загрузившего (если указано)",
				},
			},
			"GET /analytics": {
				Method:      "GET",
				Description: "Получить статистику использования сервиса",
				Response: map[string]interface{}{
					"total_files":        "Общее количество файлов",
					"total_size":         "Общий размер всех файлов в байтах",
					"total_size_human":   "Общий размер в читаемом формате",
					"average_file_size":  "Средний размер файла в байтах",
					"files_by_extension": "Распределение файлов по расширениям",
				},
			},
			"GET /info": {
				Method:      "GET",
				Description: "Получить информацию о сервисе и доступных эндпоинтах",
				Response:    "Этот документ",
			},
			"GET /health": {
				Method:      "GET",
				Description: "Проверка состояния сервиса (liveness probe)",
				Response:    "OK",
			},
			"GET /ready": {
				Method:      "GET",
				Description: "Проверка готовности сервиса (readiness probe)",
				Response:    "Ready",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
