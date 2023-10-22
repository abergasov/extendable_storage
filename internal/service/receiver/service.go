package receiver

import (
	"context"
	"database/sql"
	"errors"
	"extendable_storage/internal/entities"
	"extendable_storage/internal/logger"
	"extendable_storage/internal/repository/file"
	"extendable_storage/internal/service/orchestrator"
	"extendable_storage/internal/utils"
	"fmt"
	"log/slog"
	"sync"
)

type Service struct {
	ctx    context.Context
	wg     sync.WaitGroup
	logger logger.AppLogger
	router orchestrator.DataRouter
	repo   *file.Repo
}

var _ DataReceiver = (*Service)(nil)

func NewService(ctx context.Context, log logger.AppLogger, router orchestrator.DataRouter, repo *file.Repo) *Service {
	srv := &Service{
		ctx:    ctx,
		logger: log.With(slog.String("service", "receiver")),
		router: router,
		repo:   repo,
	}
	go srv.cleanupBadChunks()
	return srv
}

func (s *Service) Stop() {
	s.wg.Wait()
}

func (s *Service) GetFile(ctx context.Context, fileID string) ([]byte, error) {
	chunks, err := s.repo.GetFileChunks(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("error get file chunks: %w", err)
	}
	data := make(map[int][]byte, len(chunks))
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		totalLen int
		errList  = make([]error, 0, len(chunks))
	)
	wg.Add(len(chunks))
	for i := range chunks {
		go func(j int) {
			defer wg.Done()
			singleChunk, err := s.router.GetFileChunk(chunks[j])
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errList = append(errList, err)
				return
			}
			totalLen += len(singleChunk)
			data[j] = singleChunk
		}(i)
	}
	wg.Wait()
	if len(errList) > 0 {
		return nil, fmt.Errorf("error get file chunks: %w", errList[0])
	}
	result := make([]byte, 0, totalLen)
	for i := 0; i < len(data); i++ {
		if len(data[i]) == 0 {
			return nil, fmt.Errorf("error get file chunks: %w", errList[0])
		}
		result = append(result, data[i]...)
	}
	return result, nil
}

func (s *Service) SaveFile(ctx context.Context, fileID string, data []byte) error {
	chunks, err := s.repo.GetFileChunks(ctx, fileID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("error check file exists: %w", err)
	}
	if len(chunks) > 0 {
		return fmt.Errorf("file already exists")
	}
	chunkedFile := chunkData(data, 6)

	chunkList := make([]*entities.FileChunk, 0, len(chunkedFile))
	for i := range chunkedFile {
		chunkList = append(chunkList, &entities.FileChunk{
			FileID:  fileID,
			ChunkID: utils.HashData(chunkedFile[i]),
		})
	}
	if err := s.repo.SaveFileChunks(ctx, fileID, chunkList); err != nil {
		return fmt.Errorf("error save file chunks: %w", err)
	}

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	errList := make([]error, 0, len(chunkedFile))
	wg.Add(len(chunkedFile))
	for i := range chunkedFile {
		go func(j int) {
			defer wg.Done()
			if err := s.router.SaveFileChunk(chunkList[j], chunkedFile[j]); err != nil {
				mu.Lock()
				errList = append(errList, err)
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if len(errList) > 0 {
		if err := s.repo.SetFileStatus(ctx, fileID, entities.FileStatusPurge); err != nil {
			return fmt.Errorf("error update file chunks status: %w", err)
		}
		return fmt.Errorf("error save file chunks: %w", errList[0])
	}

	if err := s.repo.SetFileStatus(ctx, fileID, entities.FileStatusComplete); err != nil {
		return fmt.Errorf("error mark file chunks completed: %w", err)
	}
	return nil
}
