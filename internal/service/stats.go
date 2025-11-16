package service

import (
	"context"
	"fmt"
	"log/slog"
	"pull-request-assigner/internal/domain/models"
	"pull-request-assigner/internal/lib/logger/sl"
)

type StatsService struct {
	log       *slog.Logger
	statsRepo StatsProvider
}

type StatsProvider interface {
	GetPRStats() (*models.PRStats, error)
}

func NewStatsService(
	log *slog.Logger,
	statsRepo StatsProvider) *StatsService {
	return &StatsService{
		log:       log,
		statsRepo: statsRepo,
	}
}

func (s *StatsService) GetPRStats(ctx context.Context) (*models.PRStats, error) {
	const op = "service.stats.GetPRStats"

	log := s.log.With(slog.String("op", op))

	log.Info("getting PR statistics")

	stats, err := s.statsRepo.GetPRStats()
	if err != nil {
		log.Error("failed to get PR stats", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("PR statistics retrieved successfully",
		slog.Int("total_prs", stats.TotalPRs),
		slog.Int("open_prs", stats.OpenPRs),
		slog.Int("merged_prs", stats.MergedPRs))

	return stats, nil
}
