package sampler

import (
	"extendable_storage/internal/logger"
	"extendable_storage/internal/repository/sampler"
	"log/slog"
)

type Service struct {
	log  logger.AppLogger
	repo *sampler.Repo
}

func InitService(log logger.AppLogger, repo *sampler.Repo) *Service {
	return &Service{
		repo: repo,
		log:  log.With(slog.String("service", "sampler")),
	}
}
