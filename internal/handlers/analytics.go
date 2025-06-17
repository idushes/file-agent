package handlers

import (
	"context"
	"encoding/json"
	"file-agent/internal/storage"
	"net/http"
	"sort"
	"time"
)

// AnalyticsHandler содержит обработчики для аналитики
type AnalyticsHandler struct {
	storage *storage.S3Storage
}

// NewAnalyticsHandler создает новый AnalyticsHandler
func NewAnalyticsHandler(s3Storage *storage.S3Storage) *AnalyticsHandler {
	return &AnalyticsHandler{
		storage: s3Storage,
	}
}

// PeriodStats статистика за период
type PeriodStats struct {
	Period    string `json:"period"`
	FileCount int64  `json:"file_count"`
	TotalSize int64  `json:"total_size"`
}

// UserStats статистика по пользователю
type UserStats struct {
	User      string `json:"user"`
	FileCount int64  `json:"file_count"`
	TotalSize int64  `json:"total_size"`
}

// AnalyticsResponse ответ с аналитикой
type AnalyticsResponse struct {
	TotalFiles int64         `json:"total_files"`
	TotalSize  int64         `json:"total_size"`
	Periods    []PeriodStats `json:"periods"`
	TopUsers   []UserStats   `json:"top_users"`
}

// GetAnalytics обрабатывает запрос аналитики
func (ah *AnalyticsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		return
	}

	// Получаем все метаданные
	ctx := context.Background()
	allMetadata, err := ah.storage.ListAllMetadata(ctx)
	if err != nil {
		writeErrorResponse(w, "Failed to load metadata", http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()

	// Инициализируем статистику
	response := AnalyticsResponse{
		TotalFiles: int64(len(allMetadata)),
		Periods:    make([]PeriodStats, 0, 3),
		TopUsers:   make([]UserStats, 0),
	}

	// Подсчитываем общий размер
	var totalSize int64
	for _, meta := range allMetadata {
		totalSize += meta.Size
	}
	response.TotalSize = totalSize

	// Анализируем по периодам
	dayAgo := now.AddDate(0, 0, -1)
	weekAgo := now.AddDate(0, 0, -7)
	monthAgo := now.AddDate(0, -1, 0)

	periods := map[string]time.Time{
		"last_day":   dayAgo,
		"last_week":  weekAgo,
		"last_month": monthAgo,
	}

	for periodName, since := range periods {
		var count int64
		var size int64

		for _, meta := range allMetadata {
			if meta.UploadedAt.After(since) {
				count++
				size += meta.Size
			}
		}

		response.Periods = append(response.Periods, PeriodStats{
			Period:    periodName,
			FileCount: count,
			TotalSize: size,
		})
	}

	// Анализируем по пользователям
	userStatsMap := make(map[string]*UserStats)

	for _, meta := range allMetadata {
		user := meta.UploadedBy
		if user == "" {
			user = "anonymous"
		}

		if stats, exists := userStatsMap[user]; exists {
			stats.FileCount++
			stats.TotalSize += meta.Size
		} else {
			userStatsMap[user] = &UserStats{
				User:      user,
				FileCount: 1,
				TotalSize: meta.Size,
			}
		}
	}

	// Преобразуем в слайс и сортируем по размеру
	var userStats []UserStats
	for _, stats := range userStatsMap {
		userStats = append(userStats, *stats)
	}

	sort.Slice(userStats, func(i, j int) bool {
		return userStats[i].TotalSize > userStats[j].TotalSize
	})

	// Берем топ-10
	if len(userStats) > 10 {
		userStats = userStats[:10]
	}

	response.TopUsers = userStats

	// Возвращаем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// writeErrorResponse записывает ошибку в ответ
func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
