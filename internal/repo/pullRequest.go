package repo

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"pull-request-assigner/internal/domain/models"
	"time"
)

type PullRequestRepo struct {
	storage *sqlx.DB
}

func NewPullRequestRepo(storage *sqlx.DB) *PullRequestRepo {
	return &PullRequestRepo{storage: storage}
}

func (r *PullRequestRepo) CreatePullRequest(pr models.PullRequest, reviewers []string) error {
	const op = "repo.pullRequest.CreatePullRequest"

	tx, err := r.storage.Beginx()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback()

	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)`
	err = tx.Get(&exists, checkQuery, pr.PullRequestId)
	if err != nil {
		return fmt.Errorf("%s: failed to check PR existence: %w", op, err)
	}
	if exists {
		return fmt.Errorf("%s: PR already exists", op)
	}

	prQuery := `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = tx.Exec(prQuery, pr.PullRequestId, pr.PullRequestName, pr.AuthorID, pr.Status, pr.CreatedAt)
	if err != nil {
		return fmt.Errorf("%s: failed to create PR: %w", op, err)
	}

	reviewerQuery := `INSERT INTO pr_reviewers (pull_request_id, reviewer_id) VALUES ($1, $2)`
	for _, reviewer := range reviewers {
		var reviewerIDInt int
		_, err := fmt.Sscanf(reviewer, "u%d", &reviewerIDInt)
		if err != nil {
			return fmt.Errorf("%s: invalid reviewer_id format %s: %w", op, reviewer, err)
		}

		_, err = tx.Exec(reviewerQuery, pr.PullRequestId, reviewerIDInt)
		if err != nil {
			return fmt.Errorf("%s: failed to add reviewer %s: %w", op, reviewer, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return nil
}

func (r *PullRequestRepo) GetPullRequest(prID string) (*models.PullRequest, error) {
	const op = "repo.pullRequest.GetPullRequest"

	query := `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests WHERE pull_request_id = $1
	`

	var pr models.PullRequest
	err := r.storage.Get(&pr, query, prID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &pr, nil
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

func (r *PullRequestRepo) MergePullRequest(prID string) (*models.PullRequest, error) {
	const op = "repo.pullRequest.MergePullRequest"

	tx, err := r.storage.Beginx()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback()

	exists, err := r.PRExists(prID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if !exists {
		return nil, fmt.Errorf("%s: PR not found", op)
	}

	query := `
		UPDATE pull_requests 
		SET status = 'MERGED', merged_at = $1 
		WHERE pull_request_id = $2
		RETURNING pull_request_id, pull_request_name, author_id, status, created_at, merged_at
	`

	var pr models.PullRequest
	err = tx.QueryRowx(query, time.Now(), prID).StructScan(&pr)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to merge PR: %w", op, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return &pr, nil
}

func (r *PullRequestRepo) GetAssignedReviewers(prID string) ([]string, error) {
	const op = "repo.pullRequest.GetAssignedReviewers"

	query := `
		SELECT u.user_id 
		FROM users u
		JOIN pr_reviewers prr ON u.user_id = prr.reviewer_id
		WHERE prr.pull_request_id = $1
	`

	var reviewerIDs []int
	err := r.storage.Select(&reviewerIDs, query, prID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get reviewers: %w", op, err)
	}

	reviewers := make([]string, len(reviewerIDs))
	for i, id := range reviewerIDs {
		reviewers[i] = fmt.Sprintf("u%d", id)
	}

	return reviewers, nil
}

func (r *PullRequestRepo) GetTeamMembers(teamName string) ([]models.User, error) {
	const op = "repo.pullRequest.GetTeamMembers"

	query := `
		SELECT user_id, username, team_name, is_active
		FROM users 
		WHERE team_name = $1 AND is_active = true
	`

	var members []models.User
	err := r.storage.Select(&members, query, teamName)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get team members: %w", op, err)
	}

	for i := range members {
		members[i].UserID = fmt.Sprintf("u%d", members[i].UserID)
	}

	return members, nil
}

func (r *PullRequestRepo) GetAuthorTeam(authorID string) (string, error) {
	const op = "repo.pullRequest.GetAuthorTeam"

	var authorIDInt int
	_, err := fmt.Sscanf(authorID, "u%d", &authorIDInt)
	if err != nil {
		return "", fmt.Errorf("%s: invalid author_id format %s: %w", op, authorID, err)
	}

	query := `SELECT team_name FROM users WHERE user_id = $1`

	var teamName string
	err = r.storage.Get(&teamName, query, authorIDInt)
	if err != nil {
		return "", fmt.Errorf("%s: failed to get author team: %w", op, err)
	}

	return teamName, nil
}

func (r *PullRequestRepo) ReassignReviewer(prID string, oldReviewerID string, newReviewerID string) error {
	const op = "repo.pullRequest.ReassignReviewer"

	tx, err := r.storage.Beginx()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback()

	var oldReviewerIDInt, newReviewerIDInt int
	_, err = fmt.Sscanf(oldReviewerID, "u%d", &oldReviewerIDInt)
	if err != nil {
		return fmt.Errorf("%s: invalid old_reviewer_id format %s: %w", op, oldReviewerID, err)
	}
	_, err = fmt.Sscanf(newReviewerID, "u%d", &newReviewerIDInt)
	if err != nil {
		return fmt.Errorf("%s: invalid new_reviewer_id format %s: %w", op, newReviewerID, err)
	}

	var assigned bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM pr_reviewers WHERE pull_request_id = $1 AND reviewer_id = $2)`
	err = tx.Get(&assigned, checkQuery, prID, oldReviewerIDInt)
	if err != nil {
		return fmt.Errorf("%s: failed to check reviewer assignment: %w", op, err)
	}
	if !assigned {
		return fmt.Errorf("%s: reviewer is not assigned to this PR", op)
	}

	updateQuery := `
		UPDATE pr_reviewers 
		SET reviewer_id = $1 
		WHERE pull_request_id = $2 AND reviewer_id = $3
	`

	_, err = tx.Exec(updateQuery, newReviewerIDInt, prID, oldReviewerIDInt)
	if err != nil {
		return fmt.Errorf("%s: failed to reassign reviewer: %w", op, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return nil
}
