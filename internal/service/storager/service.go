package storager

import (
	"context"
	"extendable_storage/internal/entities"
	"extendable_storage/internal/logger"
	"extendable_storage/internal/utils"
	"fmt"
	"log/slog"
	"os"
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
	// todo add check for data dir exist for calculate usage
	return &Service{
		ctx:         ctx,
		conf:        conf,
		nodeID:      conf.NodeID,
		logger:      log.With(slog.String("service", "storager")),
		maxBytesLen: uint64(conf.MaxLimitMB) * bytesToMB,
	}
}

func (s *Service) GetUsage() (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	percentage := (float64(s.currentUsage) / float64(s.maxBytesLen)) * 100
	return percentage, nil
}

func (s *Service) GetFile(chunk *entities.FileChunk) ([]byte, error) {
	filePath, err := s.predictFilePath(s.conf, chunk)
	if err != nil {
		return nil, fmt.Errorf("error create dir for file getting: %w", err)
	}
	return os.ReadFile(filePath)
}

func (s *Service) SaveFile(chunk *entities.FileChunk, data []byte) error {
	filePath, err := s.predictFilePath(s.conf, chunk)
	if err != nil {
		return fmt.Errorf("error create dir for file saving: %w", err)
	}
	if err = os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("error save file: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentUsage += uint64(len(data))
	return nil
}

func (s *Service) SaveFromSource(chunksFrom, chunksTo uint32, source DataKeeper) error {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		errList = make([]error, 0, chunksTo-chunksFrom)
	)
	s.logger.Info("start save from source", slog.Int64("from", int64(chunksFrom)), slog.Int64("to", int64(chunksTo)))
	for i := chunksFrom; i <= chunksTo; i++ {
		wg.Add(1)
		go func(j uint32) {
			defer wg.Done()
			data, checkSum, err := source.ServeChunksInRange(j)
			if err != nil {
				s.logger.Error("error get file from source", err)
				mu.Lock()
				errList = append(errList, fmt.Errorf("error get file from source for %d: %w", j, err))
				mu.Unlock()
				return
			}
			if checkSum == -1 {
				return // no data in this range
			}
			dataPath, zipPath, err := s.predictZipPath(s.conf, j)
			if err != nil {
				s.logger.Error("error create dir for file saving", err)
				mu.Lock()
				errList = append(errList, fmt.Errorf("error create dir for file saving for %d: %w", j, err))
				mu.Unlock()
				return
			}
			if err = os.WriteFile(zipPath+"1_tmp.zip", data, 0600); err != nil {
				s.logger.Error("error save file", err)
				mu.Lock()
				errList = append(errList, fmt.Errorf("error save file for %d: %w", j, err))
				mu.Unlock()
				return
			}
			savedChecksum, err := CalculateCRC32(zipPath + "1_tmp.zip")
			if err != nil {
				s.logger.Error("error calculate checksum", err)
				mu.Lock()
				errList = append(errList, fmt.Errorf("error calculate checksum for %d: %w", j, err))
				mu.Unlock()
				return
			}
			if savedChecksum != uint32(checkSum) {
				s.logger.Error("checksum mismatch", err)
				mu.Lock()
				errList = append(errList, fmt.Errorf("checksum mismatch for %d: %w", j, err))
				mu.Unlock()
				return
			}
			if err = UnpackZip(zipPath+"1_tmp.zip", dataPath); err != nil {
				s.logger.Error("error unpack zip", err)
				mu.Lock()
				errList = append(errList, fmt.Errorf("error unpack zip for %d: %w", j, err))
				mu.Unlock()
				return
			}
			s.logger.Info("file saved", slog.Int64("position", int64(j)))
			dataSize, err := calculateFolderSize(dataPath)
			if err != nil {
				s.logger.Error("error get file info", err)
				mu.Lock()
				errList = append(errList, fmt.Errorf("error get file info for %d: %w", j, err))
				mu.Unlock()
				return
			}
			s.mu.Lock()
			s.currentUsage += dataSize
			s.mu.Unlock()
		}(i)
	}
	wg.Wait()
	if len(errList) > 0 {
		return fmt.Errorf("error save from source: %w", errList[0])
	}
	return nil
}

func (s *Service) DropChunksInRange(chunksFrom, chunksTo uint32) error {
	for i := chunksFrom; i <= chunksTo; i++ {
		dataPath, _, err := s.predictZipPath(s.conf, i)
		if err != nil {
			return fmt.Errorf("error create dir for file saving: %w", err)
		}
		dataExist, err := s.checkDirExist(dataPath)
		if err != nil {
			return fmt.Errorf("error check dir exist: %w", err)
		}
		if !dataExist {
			continue
		}

		removedSize, err := deleteFolderAndCalculateSize(dataPath)
		if err != nil {
			return fmt.Errorf("error delete folder %d: %v", i, err)
		}
		s.mu.Lock()
		s.currentUsage -= removedSize
		s.mu.Unlock()
	}
	return nil
}

func (s *Service) ServeChunksInRange(chunksRange uint32) (data []byte, checkSum int32, err error) {
	dataPath, zipPath, err := s.predictZipPath(s.conf, chunksRange)
	if err != nil {
		return nil, 0, fmt.Errorf("error create dir for file saving: %w", err)
	}
	defer s.deleteFile(zipPath)

	// check folder exist
	dataExist, err := s.checkDirExist(dataPath)
	if err != nil {
		return nil, 0, fmt.Errorf("error check dir exist: %w", err)
	}
	if !dataExist {
		return nil, -1, nil
	}

	checkSumTmp, err := ZipAndCalculateCRC32(dataPath, zipPath)
	if err != nil {
		return nil, 0, fmt.Errorf("error zip file: %w", err)
	}
	data, err = os.ReadFile(zipPath)
	if err != nil {
		return nil, 0, fmt.Errorf("error get file content: %w", err)
	}
	return data, int32(checkSumTmp), nil
}

func (s *Service) PurgeFileChunks(chunks []*entities.FileChunk) error {
	panic("implement me")
}

func (s *Service) predictFilePath(conf *Config, chunk *entities.FileChunk) (filePath string, err error) {
	parentDir := fmt.Sprintf("%s/%s/%d/%s", conf.DataDir, s.nodeID, chunk.Hash()%entities.CircleSectors, utils.HashString(chunk.String())[:4])
	filePath = parentDir + "/" + utils.HashString(chunk.String())
	if err = s.createDirIfNotExist(parentDir); err != nil {
		return "", fmt.Errorf("error create dir for file saving: %w", err)
	}
	return filePath, nil
}

func (s *Service) predictZipPath(conf *Config, positionID uint32) (dataPath, zipPath string, err error) {
	zipPath = fmt.Sprintf("%s/%s/zip/%d/", conf.DataDir, s.nodeID, positionID)
	if err = s.createDirIfNotExist(zipPath); err != nil {
		return "", "", fmt.Errorf("error create dir for file saving: %w", err)
	}
	return fmt.Sprintf("%s/%s/%d/", conf.DataDir, s.nodeID, positionID), fmt.Sprintf("%s%d.zip", zipPath, positionID), nil
}
