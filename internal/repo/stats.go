package repo

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"pull-request-assigner/internal/domain/models"
)

type StatsRepo struct {
	storage *sqlx.DB
}

func NewStatsRepo(storage *sqlx.DB) *StatsRepo {
	return &StatsRepo{storage: storage}
}

func (r *StatsRepo) GetPRStats() (*models.PRStats, error) {
	const op = "repo.stats.GetPRStats"

	prStatsQuery := `
		SELECT 
			COUNT(*) as total_prs,
			COUNT(CASE WHEN status = 'OPEN' THEN 1 END) as open_prs,
			COUNT(CASE WHEN status = 'MERGED' THEN 1 END) as merged_prs
		FROM pull_requests
	`

	var prStats struct {
		TotalPRs  int `db:"total_prs"`
		OpenPRs   int `db:"open_prs"`
		MergedPRs int `db:"merged_prs"`
	}

	err := r.storage.Get(&prStats, prStatsQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	avgReviewersQuery := `
		SELECT 
			CASE 
				WHEN COUNT(DISTINCT pr.pull_request_id) = 0 THEN 0
				ELSE CAST(COUNT(prr.reviewer_id) AS FLOAT) / COUNT(DISTINCT pr.pull_request_id)
			END as avg_reviewers
		FROM pull_requests pr
		LEFT JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
	`

	var avgReviewers float64
	err = r.storage.Get(&avgReviewers, avgReviewersQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &models.PRStats{
		TotalPRs:          prStats.TotalPRs,
		OpenPRs:           prStats.OpenPRs,
		MergedPRs:         prStats.MergedPRs,
		AvgReviewersPerPR: avgReviewers,
	}, nil
}
