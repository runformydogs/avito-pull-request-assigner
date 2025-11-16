package repo

import (
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"pull-request-assigner/internal/apperrors"
	"pull-request-assigner/internal/domain/models"
	"strconv"
)

type UserRepo struct {
	storage *sqlx.DB
}

func NewUserRepo(storage *sqlx.DB) *UserRepo {
	return &UserRepo{storage: storage}
}

func (r *UserRepo) SetIsActive(isActive bool, userID int) (models.User, error) {
	const op = "repo.user.SetIsActive"

	query := `UPDATE users SET is_active = $1 WHERE user_id = $2 
        RETURNING user_id, username, team_name, is_active
    `

	var user models.User
	err := r.storage.Get(&user, query, isActive, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, apperrors.ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	id, _ := strconv.Atoi(user.UserID)
	user.UserID = fmt.Sprintf("u%d", id)

	return user, nil
}

func (r *UserRepo) GetReview(userID int) ([]models.PullRequestShort, error) {
	const op = "repo.user.GetReview"

	query := `
        SELECT 
            pr.pull_request_id,
            pr.pull_request_name, 
            pr.author_id,
            pr.status
        FROM pull_requests pr
        JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
        WHERE prr.reviewer_id = $1`

	var prs []models.PullRequestShort

	err := r.storage.Select(&prs, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return []models.PullRequestShort{}, nil
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for i := range prs {
		authorIDInt, err := strconv.Atoi(prs[i].AuthorID)
		if err != nil {
			continue
		}
		prs[i].AuthorID = fmt.Sprintf("u%d", authorIDInt)
	}

	return prs, nil
}
