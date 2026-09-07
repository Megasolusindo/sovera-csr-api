package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type StorageService interface {
	SaveTemplate(ctx context.Context, orgID, fileType string, content []byte) (string, error)
	GetTemplate(ctx context.Context, s3Key string) ([]byte, error)
	DeleteTemplate(ctx context.Context, s3Key string) error
}

type LocalFallbackStorage struct {
	baseDir string
	mu      sync.RWMutex
}

func NewStorageService() StorageService {
	baseDir := os.Getenv("STORAGE_LOCAL_DIR")
	if baseDir == "" {
		baseDir = "/tmp/sovera_storage"
	}
	_ = os.MkdirAll(baseDir, 0755)
	return &LocalFallbackStorage{baseDir: baseDir}
}

func (s *LocalFallbackStorage) SaveTemplate(ctx context.Context, orgID, fileType string, content []byte) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cleanOrgID := strings.TrimPrefix(orgID, "org_")
	cleanFileType := strings.ToLower(strings.TrimSpace(fileType))
	if cleanFileType != "pptx" && cleanFileType != "docx" {
		cleanFileType = "pptx"
	}

	fileName := fmt.Sprintf("master_%s.%s", cleanFileType, cleanFileType)
	relKey := filepath.Join("templates", cleanOrgID, fileName)
	fullPath := filepath.Join(s.baseDir, relKey)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create storage directory: %w", err)
	}

	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to save template file: %w", err)
	}

	return relKey, nil
}

func (s *LocalFallbackStorage) GetTemplate(ctx context.Context, s3Key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fullPath := filepath.Join(s.baseDir, filepath.Clean(s3Key))
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("template file not found in storage: %w", err)
	}

	return data, nil
}

func (s *LocalFallbackStorage) DeleteTemplate(ctx context.Context, s3Key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fullPath := filepath.Join(s.baseDir, filepath.Clean(s3Key))
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete template file: %w", err)
	}

	return nil
}

// ReadAll helper function
func ReadAll(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	return buf.Bytes(), err
}
