package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Storage представляет S3-совместимое хранилище
type S3Storage struct {
	client *s3.Client
	bucket string
}

// FileMetadata содержит метаданные загруженного файла
type FileMetadata struct {
	ID         string    `json:"id"`
	Filename   string    `json:"filename"`
	Size       int64     `json:"size"`
	UploadedAt time.Time `json:"uploaded_at"`
	UploadedBy string    `json:"uploaded_by,omitempty"` // Информация о том, кто загрузил (опционально)
}

// NewS3Storage создает новый S3Storage
func NewS3Storage(endpoint, accessKey, secretKey, bucket string) (*S3Storage, error) {
	// Создаем кастомный endpoint для Minio
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:               endpoint,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
	})

	// Конфигурируем AWS SDK для работы с Minio
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion("us-east-1"), // Minio обычно не требует конкретного региона
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // Важно для Minio
	})

	storage := &S3Storage{
		client: client,
		bucket: bucket,
	}

	// Проверяем подключение к бакету
	if err := storage.checkBucket(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to access bucket: %w", err)
	}

	return storage, nil
}

// checkBucket проверяет доступность бакета
func (s *S3Storage) checkBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	return err
}

// SaveFile сохраняет файл в S3
func (s *S3Storage) SaveFile(ctx context.Context, fileID, filename string, content io.Reader, size int64, uploadedBy string) error {
	// Определяем ключ для файла
	fileKey := fmt.Sprintf("files/%s", fileID)

	// Загружаем файл в S3
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(fileKey),
		Body:          content,
		ContentLength: aws.Int64(size),
		Metadata: map[string]string{
			"original-filename": filename,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	// Сохраняем метаданные отдельно
	metadata := FileMetadata{
		ID:         fileID,
		Filename:   filename,
		Size:       size,
		UploadedAt: time.Now().UTC(),
		UploadedBy: uploadedBy,
	}

	if err := s.saveMetadata(ctx, metadata); err != nil {
		log.Printf("Failed to save metadata for file %s: %v", fileID, err)
		// Не возвращаем ошибку, так как файл уже загружен
	}

	return nil
}

// GetFile получает файл из S3
func (s *S3Storage) GetFile(ctx context.Context, fileID string) (io.ReadCloser, *FileMetadata, error) {
	// Загружаем метаданные
	metadata, err := s.loadMetadata(ctx, fileID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load metadata: %w", err)
	}

	// Определяем ключ для файла
	fileKey := fmt.Sprintf("files/%s", fileID)

	// Получаем файл из S3
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fileKey),
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get file from S3: %w", err)
	}

	return result.Body, metadata, nil
}

// saveMetadata сохраняет метаданные файла в S3
func (s *S3Storage) saveMetadata(ctx context.Context, metadata FileMetadata) error {
	metadataKey := fmt.Sprintf("metadata/%s.json", metadata.ID)

	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(metadataKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})

	if err != nil {
		return fmt.Errorf("failed to save metadata to S3: %w", err)
	}

	return nil
}

// loadMetadata загружает метаданные файла из S3
func (s *S3Storage) loadMetadata(ctx context.Context, fileID string) (*FileMetadata, error) {
	metadataKey := fmt.Sprintf("metadata/%s.json", fileID)

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(metadataKey),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get metadata from S3: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata FileMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// FileExists проверяет существование файла в S3
func (s *S3Storage) FileExists(ctx context.Context, fileID string) bool {
	fileKey := fmt.Sprintf("files/%s", fileID)

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fileKey),
	})

	return err == nil
}

// GetFileMetadata получает только метаданные файла без загрузки самого файла
func (s *S3Storage) GetFileMetadata(ctx context.Context, fileID string) (*FileMetadata, error) {
	return s.loadMetadata(ctx, fileID)
}

// ListAllMetadata получает метаданные всех файлов для аналитики
func (s *S3Storage) ListAllMetadata(ctx context.Context) ([]*FileMetadata, error) {
	// Список всех объектов в папке metadata/
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String("metadata/"),
	}

	var allMetadata []*FileMetadata

	// Получаем все страницы результатов
	paginator := s3.NewListObjectsV2Paginator(s.client, listInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list metadata objects: %w", err)
		}

		// Обрабатываем каждый объект метаданных
		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}

			// Извлекаем fileID из ключа (metadata/{fileID}.json)
			key := *obj.Key
			if !strings.HasSuffix(key, ".json") {
				continue
			}

			fileID := strings.TrimPrefix(key, "metadata/")
			fileID = strings.TrimSuffix(fileID, ".json")

			// Загружаем метаданные
			metadata, err := s.loadMetadata(ctx, fileID)
			if err != nil {
				log.Printf("Failed to load metadata for %s: %v", fileID, err)
				continue
			}

			allMetadata = append(allMetadata, metadata)
		}
	}

	return allMetadata, nil
}

// GetContentType определяет content type файла по расширению
func GetContentType(filename string) string {
	ext := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(ext, ".jpg") || strings.HasSuffix(ext, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(ext, ".png"):
		return "image/png"
	case strings.HasSuffix(ext, ".gif"):
		return "image/gif"
	case strings.HasSuffix(ext, ".pdf"):
		return "application/pdf"
	case strings.HasSuffix(ext, ".zip"):
		return "application/zip"
	case strings.HasSuffix(ext, ".txt"):
		return "text/plain"
	case strings.HasSuffix(ext, ".json"):
		return "application/json"
	case strings.HasSuffix(ext, ".xml"):
		return "application/xml"
	case strings.HasSuffix(ext, ".html"):
		return "text/html"
	case strings.HasSuffix(ext, ".css"):
		return "text/css"
	case strings.HasSuffix(ext, ".js"):
		return "application/javascript"
	default:
		return "application/octet-stream"
	}
}
