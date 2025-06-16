package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// FileMetadata содержит метаданные загруженного файла
type FileMetadata struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
}

// saveMetadata сохраняет метаданные файла
func (fh *FileHandler) saveMetadata(metadata FileMetadata) error {
	metadataPath := filepath.Join(fh.storagePath, metadata.ID+".meta")

	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	return os.WriteFile(metadataPath, data, 0644)
}

// loadMetadata загружает метаданные файла
func (fh *FileHandler) loadMetadata(fileID string) (*FileMetadata, error) {
	metadataPath := filepath.Join(fh.storagePath, fileID+".meta")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, err
	}

	var metadata FileMetadata
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}
