package repo

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"pull-request-assigner/internal/domain/models"
)

type UserRepo struct {
	storage *sqlx.DB
}

func New(storage *sqlx.DB) *UserRepo {
	return &UserRepo{storage: storage}
}

func (r *UserRepo) SetIsActive(isActive bool, userID int) (models.User, error) {
	const op = "repo.user.SetIsActive"

	query := `UPDATE users SET is_active = $1 WHERE user_id = $2 
        RETURNING user_id, username, team_name, is_active
    `

	var user models.User
	err := r.storage.QueryRowx(query, isActive, userID).StructScan(&user)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (r *UserRepo) GetReview(userID int) ([]models.PullRequestShort, error) {
	const op = "repo.user.GetReview"

	query := `
        SELECT pr.* 
        FROM pull_requests pr
        JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
        WHERE prr.reviewer_id = $1`

	var prs []models.PullRequestShort

	err := r.storage.Select(&prs, query, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return prs, nil
}
