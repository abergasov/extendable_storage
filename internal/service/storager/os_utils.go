package storager

import (
	"errors"
	"extendable_storage/internal/entities"
	"log/slog"
	"os"
)

var (
	errFileToLarge = errors.New("file size exceeds the limit")
)

func (s *Service) createDirIfNotExist(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
	}
	return err
}

func (s *Service) getFileContent(filePath string) ([]byte, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Check if the file size exceeds the limit
	if fileInfo.Size() > entities.ChunkSize {
		s.logger.With(
			slog.Int("limit", entities.ChunkSize),
			slog.Int64("file_size", fileInfo.Size()),
		).Error("file size exceeds the limit", errFileToLarge)
		return nil, errFileToLarge
	}
	return os.ReadFile(filePath)
}

func (s *Service) saveToFile(filePath string, data []byte) error {
	err := os.WriteFile(filePath, data, 0600)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) deleteFile(filePath string) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return
	}
	if err = os.Remove(filePath); err != nil {
		s.logger.Error("error deleting file", err)
		return
	}
	s.mu.Lock()
	s.currentUsage -= uint64(fileInfo.Size())
	s.mu.Unlock()
}
