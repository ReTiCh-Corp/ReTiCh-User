package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage stores avatars on the local filesystem.
type LocalStorage struct {
	uploadsDir string
	baseURL    string
}

func NewLocalStorage(uploadsDir, baseURL string) *LocalStorage {
	return &LocalStorage{uploadsDir: uploadsDir, baseURL: baseURL}
}

func (s *LocalStorage) Upload(_ context.Context, userID, ext string, content io.Reader) (string, error) {
	filename := userID + ext
	dstPath := filepath.Join(s.uploadsDir, filename)

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("create avatar file: %w", err)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, content); err != nil {
		return "", fmt.Errorf("write avatar file: %w", err)
	}

	return s.baseURL + "/uploads/" + filename, nil
}

func (s *LocalStorage) Delete(_ context.Context, blobName string) error {
	dstPath := filepath.Join(s.uploadsDir, blobName)
	err := os.Remove(dstPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete avatar file: %w", err)
	}
	return nil
}
