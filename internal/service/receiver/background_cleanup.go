package receiver

import (
	"extendable_storage/internal/entities"
	"time"
)

func (s *Service) cleanupBadChunks() {
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ticker.C:
			s.wg.Add(1)
			s.cleanData()
			s.wg.Done()
		case <-s.ctx.Done():
			return
		}
	}
}

// cleanData purge uncompleted or failed uploads
func (s *Service) cleanData() {
	// 1. load files with purge status and as orchestrator for each file chunk
	purgeCandidate, err := s.repo.GetChunksByStatus(s.ctx, entities.FileStatusPurge)
	if err != nil {
		s.logger.Error("error get purge candidate", err)
		return
	}
	s.cleanupFiles(purgeCandidate)

	// 2. load files with new status and which not updated for 21 hour
	purgeCandidate, err = s.repo.GetChunksUpdatedBeforeDataWithStatus(s.ctx, entities.FileStatusNew, time.Now().Add(-24*time.Hour))
	if err != nil {
		s.logger.Error("error get purge candidate", err)
		return
	}
	s.cleanupFiles(purgeCandidate)
}

func (s *Service) cleanupFiles(purgeCandidate []*entities.File) {
	for _, file := range purgeCandidate {
		if err := s.router.PurgeFileChunks(file.Chunks); err != nil {
			s.logger.Error("error purge file chunks", err)
			continue
		}
		if err := s.repo.DeleteFile(s.ctx, file.ID); err != nil {
			s.logger.Error("error delete file", err)
		}
	}
}
