package storager

import (
	"extendable_storage/internal/entities"
	"extendable_storage/internal/logger"
	"log/slog"
)

type Service struct {
	logger logger.AppLogger
}

var _ DataKeeper = (*Service)(nil)

func NewService(logger logger.AppLogger) *Service {
	return &Service{
		logger: logger.With(slog.String("service", "storager")),
	}
}

func (s *Service) GetUsage() (int64, error) {
	panic("implement me")
}

func (s *Service) GetFile(file *entities.FileChunk) ([]byte, error) {
	panic("implement me")
}

func (s *Service) SaveFile(file *entities.FileChunk, data []byte) error {
	panic("implement me")
}

func (s *Service) SaveFromSource(chunks []*entities.FileChunk, source string) error {
	panic("implement me")
}

func (s *Service) CheckFilesExistence(chunks []*entities.FileChunk) (map[string]bool, error) {
	panic("implement me")
}

func (s *Service) PurgeFileChunks(chunks []*entities.FileChunk) error {
	panic("implement me")
}
