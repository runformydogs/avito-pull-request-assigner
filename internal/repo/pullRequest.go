package repo

import (
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"pull-request-assigner/internal/apperrors"
	"pull-request-assigner/internal/domain/models"
	"time"
)

type PullRequestRepo struct {
	storage *sqlx.DB
}

func NewPullRequestRepo(storage *sqlx.DB) *PullRequestRepo {
	return &PullRequestRepo{storage: storage}
}

func (r *PullRequestRepo) CreatePR(pr models.PullRequest) error {
	const op = "repo.pullrequest.CreatePR"

	query := `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	authorID, err := extractUserID(pr.AuthorID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, apperrors.ErrAuthorRequired)
	}

	_, err = r.storage.Exec(query, pr.PullRequestId, pr.PullRequestName, authorID, pr.Status, pr.CreatedAt)
	if err != nil {
		if isDuplicateKeyError(err) {
			return fmt.Errorf("%s: %w", op, apperrors.ErrPRExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *PullRequestRepo) PRExists(prID string) (bool, error) {
	const op = "repo.pullRequest.PRExists"

	query := `SELECT COUNT(*) FROM pull_requests WHERE pull_request_id = $1`

	var count int
	err := r.storage.Get(&count, query, prID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return count > 0, nil
}

func (r *PullRequestRepo) GetPR(prID string) (*models.PullRequest, error) {
	const op = "repo.pullRequest.GetPR"

	query := `
		SELECT 
			pull_request_id,
			pull_request_name,
			author_id,
			status,
			created_at,
			merged_at
		FROM pull_requests 
		WHERE pull_request_id = $1
	`

	var pr struct {
		PullRequestId   string       `db:"pull_request_id"`
		PullRequestName string       `db:"pull_request_name"`
		AuthorID        int          `db:"author_id"`
		Status          string       `db:"status"`
		CreatedAt       time.Time    `db:"created_at"`
		MergedAt        sql.NullTime `db:"merged_at"`
	}

	err := r.storage.Get(&pr, query, prID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("%s: %w", op, apperrors.ErrPRNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result := &models.PullRequest{
		PullRequestId:   pr.PullRequestId,
		PullRequestName: pr.PullRequestName,
		AuthorID:        fmt.Sprintf("u%d", pr.AuthorID),
		Status:          pr.Status,
		CreatedAt:       pr.CreatedAt,
		MergedAt:        pr.MergedAt,
	}

	return result, nil
}

func (r *PullRequestRepo) GetPRWithReviewers(prID string) (*models.PullRequest, []string, error) {
	const op = "repo.pullRequest.GetPRWithReviewers"

	pr, err := r.GetPR(prID)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	reviewersQuery := `
		SELECT reviewer_id 
		FROM pr_reviewers 
		WHERE pull_request_id = $1
	`

	var reviewerIDs []int
	err = r.storage.Select(&reviewerIDs, reviewersQuery, prID)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: failed to get reviewers: %w", op, err)
	}

	reviewerStrs := make([]string, len(reviewerIDs))
	for i, id := range reviewerIDs {
		reviewerStrs[i] = fmt.Sprintf("u%d", id)
	}

	return pr, reviewerStrs, nil
}

func (r *PullRequestRepo) AddPRReviewers(prID string, reviewerIDs []string) error {
	const op = "repo.pullRequest.AddPRReviewers"

	tx, err := r.storage.Beginx()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback()

	query := `INSERT INTO pr_reviewers (pull_request_id, reviewer_id) VALUES ($1, $2)`

	for _, reviewerID := range reviewerIDs {
		reviewerIDInt, err := extractUserID(reviewerID)
		if err != nil {
			return fmt.Errorf("%s: %w", op, apperrors.ErrAuthorRequired)
		}

		_, err = tx.Exec(query, prID, reviewerIDInt)
		if err != nil {
			return fmt.Errorf("%s: failed to add reviewer %s: %w", op, reviewerID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return nil
}

func (r *PullRequestRepo) MergePR(prID string) error {
	const op = "repo.pullRequest.MergePR"

	query := `
		UPDATE pull_requests 
		SET status = 'MERGED', merged_at = $1
		WHERE pull_request_id = $2 AND status != 'MERGED'
	`

	result, err := r.storage.Exec(query, time.Now(), prID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if rowsAffected == 0 {
		exists, err := r.PRExists(prID)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		if exists {
			return nil
		}
		return fmt.Errorf("%s: %w", op, apperrors.ErrPRNotFound)
	}

	return nil
}

func (r *PullRequestRepo) GetAuthorTeam(authorID string) (string, error) {
	const op = "repo.pullRequest.GetAuthorTeam"

	authorIDInt, err := extractUserID(authorID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, apperrors.ErrAuthorRequired)
	}

	query := `SELECT team_name FROM users WHERE user_id = $1`

	var teamName string
	err = r.storage.Get(&teamName, query, authorIDInt)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return "", fmt.Errorf("%s: %w", op, apperrors.ErrPRAuthorNotFound)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return teamName, nil
}

func (r *PullRequestRepo) GetActiveTeamMembers(teamName string, excludeUserIDs []string) ([]string, error) {
	const op = "repo.pullRequest.GetActiveTeamMembers"

	query := `
		SELECT user_id 
		FROM users 
		WHERE team_name = $1 AND is_active = true
	`

	var userIDs []int
	err := r.storage.Select(&userIDs, query, teamName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result := make([]string, 0)
	excludeMap := make(map[string]bool)
	for _, id := range excludeUserIDs {
		excludeMap[id] = true
	}

	for _, id := range userIDs {
		userIDStr := fmt.Sprintf("u%d", id)
		if !excludeMap[userIDStr] {
			result = append(result, userIDStr)
		}
	}

	return result, nil
}

func (r *PullRequestRepo) ReplaceReviewer(prID string, oldReviewerID string, newReviewerID string) error {
	const op = "repo.pullRequest.ReplaceReviewer"

	tx, err := r.storage.Beginx()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback()

	checkQuery := `SELECT COUNT(*) FROM pr_reviewers WHERE pull_request_id = $1 AND reviewer_id = $2`
	var count int
	oldReviewerIDInt, _ := extractUserID(oldReviewerID)
	err = tx.Get(&count, checkQuery, prID, oldReviewerIDInt)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if count == 0 {
		return fmt.Errorf("%s: %w", op, apperrors.ErrReviewerNotAssigned)
	}

	deleteQuery := `DELETE FROM pr_reviewers WHERE pull_request_id = $1 AND reviewer_id = $2`
	_, err = tx.Exec(deleteQuery, prID, oldReviewerIDInt)
	if err != nil {
		return fmt.Errorf("%s: failed to remove old reviewer: %w", op, err)
	}

	newReviewerIDInt, _ := extractUserID(newReviewerID)
	insertQuery := `INSERT INTO pr_reviewers (pull_request_id, reviewer_id) VALUES ($1, $2)`
	_, err = tx.Exec(insertQuery, prID, newReviewerIDInt)
	if err != nil {
		return fmt.Errorf("%s: failed to add new reviewer: %w", op, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return nil
}

func extractUserID(userIDStr string) (int, error) {
	var userID int
	_, err := fmt.Sscanf(userIDStr, "u%d", &userID)
	if err != nil {
		return 0, apperrors.ErrAuthorRequired
	}
	return userID, nil
}
