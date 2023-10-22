package orchestrator

import (
	"context"
	"extendable_storage/internal/entities"
	"extendable_storage/internal/logger"
	"extendable_storage/internal/service/storager"
	"fmt"
	"log/slog"
	"sync"
)

const (
	nodeStateNotReady = iota
	nodeStatePreparing
	nodeStateReady
)

type Service struct {
	circle *Circle
	ctx    context.Context
	wg     sync.WaitGroup
	mu     sync.RWMutex
	logger logger.AppLogger
	Nodes  map[uint32]dataKeeperContainer
}

var _ DataRouter = (*Service)(nil)

func NewService(ctx context.Context, log logger.AppLogger) *Service {
	return &Service{
		circle: NewCircle(),
		ctx:    ctx,
		logger: log.With(slog.String("service", "orchestrator")),
	}
}

func (s *Service) Stop() {
	// todo add dump and restore server topology for ring
	s.wg.Wait()
}

func (s *Service) AddDataKeeper(serviceID string, storage storager.DataKeeper) error {
	rangeFrom, rangeTo, oldRangeTo, err := s.circle.AddServer(serviceID, storage)
	if err != nil {
		return fmt.Errorf("error add server in circle map: %w", err)
	}
	// start rebalancing process
	if oldRangeTo == rangeTo {
		// no need rebalance
		s.circle.MarkServerReady(serviceID)
		return nil
	}
	// new server need get all files in range [rangeFrom, rangeTo]
	// request them from server with id = [rangeFrom, oldRangeTo]
	sourceSrv, _, err := s.circle.GetServerForPosition(rangeTo)
	if err != nil {
		return fmt.Errorf("error get server for position: %w", err)
	}
	if err = storage.SaveFromSource(rangeFrom, rangeTo, sourceSrv); err != nil {
		return fmt.Errorf("error save from source: %w", err)
	}
	s.logger.Info("rebalance finished", slog.String("service_id", serviceID))
	s.circle.MarkServerReady(serviceID)
	if err = sourceSrv.DropChunksInRange(rangeFrom, rangeTo); err != nil {
		// todo move this to background process with retry logic
		return fmt.Errorf("error drop chunks in range: %w", err)
	}
	return nil
}

func (s *Service) GetFileChunk(chunk *entities.FileChunk) ([]byte, error) {
	srv, _, err := s.circle.GetServerForChunk(chunk)
	if err != nil {
		return nil, fmt.Errorf("error get server for chunk: %w", err)
	}
	return srv.GetFile(chunk)
}

func (s *Service) SaveFileChunk(chunk *entities.FileChunk, data []byte) error {
	srv, _, err := s.circle.GetServerForChunk(chunk)
	if err != nil {
		return fmt.Errorf("error get server for chunk: %w", err)
	}
	return srv.SaveFile(chunk, data)
}

func (s *Service) PurgeFileChunks(chunks []*entities.FileChunk) error {
	requests := make(map[string][]*entities.FileChunk, len(chunks))
	srvs := make(map[string]storager.DataKeeper, entities.CircleSectors)
	for _, chunk := range chunks {
		srv, srvID, err := s.circle.GetServerForChunk(chunk)
		if err != nil {
			return fmt.Errorf("error get server for chunk: %w", err)
		}
		if _, ok := requests[srvID]; !ok {
			requests[srvID] = make([]*entities.FileChunk, 0, len(chunks))
		}
		requests[srvID] = append(requests[srvID], chunk)
		srvs[srvID] = srv
	}
	var (
		wg      sync.WaitGroup
		errList = make([]error, 0, len(srvs))
	)
	wg.Add(len(srvs))
	for srvID := range srvs {
		go func(srvID string) {
			defer wg.Done()
			if err := srvs[srvID].PurgeFileChunks(requests[srvID]); err != nil {
				s.mu.Lock()
				errList = append(errList, err)
				s.mu.Unlock()
				s.logger.Error("error purge file chunks", err)
			}
		}(srvID)
	}
	wg.Wait()
	if len(errList) > 0 {
		return fmt.Errorf("error purge file chunks: %v", errList)
	}
	return nil
}
