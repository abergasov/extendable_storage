package storager

import (
	"context"
	"extendable_storage/internal/entities"
	"extendable_storage/internal/logger"
	"extendable_storage/internal/utils"
	"fmt"
	"log/slog"
	"sync"
)

const (
	bytesToMB = 1024 * 1024
)

type Config struct {
	MaxLimitMB int
	NodeID     string
	DataDir    string
}

type Service struct {
	ctx          context.Context
	mu           sync.RWMutex
	conf         *Config
	maxBytesLen  uint64
	currentUsage uint64
	nodeID       string
	logger       logger.AppLogger
}

var _ DataKeeper = (*Service)(nil)

func NewService(ctx context.Context, conf *Config, log logger.AppLogger) *Service {
	return &Service{
		ctx:         ctx,
		conf:        conf,
		nodeID:      conf.NodeID,
		logger:      log.With(slog.String("service", "storager")),
		maxBytesLen: uint64(conf.MaxLimitMB) * bytesToMB,
	}
}

func (s *Service) GetUsage() (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	percentage := (float64(s.currentUsage) / float64(s.maxBytesLen)) * 100
	return uint64(percentage), nil
}

func (s *Service) GetFile(chunk *entities.FileChunk) ([]byte, error) {
	filePath, err := s.predictFilePath(s.conf, chunk)
	if err != nil {
		return nil, fmt.Errorf("error create dir for file getting: %w", err)
	}
	return s.getFileContent(filePath)
}

func (s *Service) SaveFile(chunk *entities.FileChunk, data []byte) error {
	if len(data) > entities.ChunkSize {
		return errFileToLarge
	}
	filePath, err := s.predictFilePath(s.conf, chunk)
	if err != nil {
		return fmt.Errorf("error create dir for file saving: %w", err)
	}
	if err = s.saveToFile(filePath, data); err != nil {
		return fmt.Errorf("error save file: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentUsage += uint64(len(data))
	return nil
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

func (s *Service) predictFilePath(conf *Config, chunk *entities.FileChunk) (filePath string, err error) {
	parentDir := conf.DataDir + "/" + utils.HashString(chunk.String())[:4]
	filePath = parentDir + "/" + utils.HashString(chunk.String())
	if err := s.createDirIfNotExist(parentDir); err != nil {
		return "", fmt.Errorf("error create dir for file saving: %w", err)
	}
	return filePath, nil
}
