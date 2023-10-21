package orchestrator

import (
	"extendable_storage/internal/entities"
	"extendable_storage/internal/logger"
	"log/slog"
)

type Service struct {
	logger logger.AppLogger
}

var _ DataRouter = (*Service)(nil)

func NewService(logger logger.AppLogger) *Service {
	return &Service{
		logger: logger.With(slog.String("service", "orchestrator")),
	}
}

func (s *Service) GetFileChunk(chunk *entities.FileChunk) ([]byte, error) {
	panic("implement me")
}

func (s *Service) SaveFileChunk(chunk *entities.FileChunk, data []byte) error {
	panic("implement me")
}

func (s *Service) PurgeFileChunks(chunks []*entities.FileChunk) error {
	panic("implement me")
}
