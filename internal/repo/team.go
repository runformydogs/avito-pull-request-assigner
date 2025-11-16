package repo

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"pull-request-assigner/internal/apperrors"
	"pull-request-assigner/internal/domain/models"
	"strconv"
)

type TeamRepo struct {
	storage *sqlx.DB
}

func NewTeamRepo(storage *sqlx.DB) *TeamRepo {
	return &TeamRepo{storage: storage}
}

func (r *TeamRepo) CreateTeam(teamName string) error {
	const op = "repo.team.CreateTeam"

	query := `INSERT INTO teams (team_name) VALUES ($1)`

	_, err := r.storage.Exec(query, teamName)
	if err != nil {
		if isDuplicateKeyError(err) {
			return fmt.Errorf("%s: %w", op, apperrors.ErrTeamExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *TeamRepo) TeamExists(teamName string) (bool, error) {
	const op = "repo.team.TeamExists"

	query := `SELECT COUNT(*) FROM teams WHERE team_name = $1`

	var count int
	err := r.storage.Get(&count, query, teamName)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return count > 0, nil
}

func (r *TeamRepo) AddTeamMembers(teamName string, members []models.User) error {
	const op = "repo.team.AddTeamMembers"

	tx, err := r.storage.Beginx()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback()

	userQuery := `
		INSERT INTO users (user_id, username, team_name, is_active) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			username = EXCLUDED.username,
			team_name = EXCLUDED.team_name,
			is_active = EXCLUDED.is_active
	`

	for _, member := range members {
		var userIDInt int
		_, err := fmt.Sscanf(member.UserID, "u%d", &userIDInt)
		if err != nil {
			return fmt.Errorf("%s: %w", op, apperrors.ErrInvalidUserID)
		}

		_, err = tx.Exec(userQuery, userIDInt, member.Username, teamName, member.IsActive)
		if err != nil {
			return fmt.Errorf("%s: failed to upsert user %s: %w", op, member.UserID, err)
		}
	}

	memberQuery := `INSERT INTO team_members (team_name, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`

	for _, member := range members {
		var userIDInt int
		_, err := fmt.Sscanf(member.UserID, "u%d", &userIDInt)
		if err != nil {
			return fmt.Errorf("%s: %w", op, apperrors.ErrInvalidUserID)
		}

		_, err = tx.Exec(memberQuery, teamName, userIDInt)
		if err != nil {
			return fmt.Errorf("%s: failed to add team member %s: %w", op, member.UserID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return nil
}

func (r *TeamRepo) GetTeamWithMembers(teamName string) (*models.Team, error) {
	const op = "repo.team.GetTeamWithMembers"

	exists, err := r.TeamExists(teamName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if !exists {
		return nil, fmt.Errorf("%s: %w", op, apperrors.ErrTeamNotFound)
	}

	query := `
		SELECT 
			u.user_id,
			u.username,
			u.team_name,
			u.is_active
		FROM users u
		JOIN team_members tm ON u.user_id = tm.user_id
		WHERE tm.team_name = $1
	`

	var members []models.User
	err = r.storage.Select(&members, query, teamName)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get team members: %w", op, err)
	}

	for i := range members {
		id, _ := strconv.Atoi(members[i].UserID)
		members[i].UserID = fmt.Sprintf("u%d", id)
	}

	team := &models.Team{
		TeamName: teamName,
		Members:  members,
	}

	return team, nil
}

func (r *TeamRepo) DeactivateTeamUsers(teamName string) (int, error) {
	const op = "repo.team.DeactivateTeamUsers"

	query := `
        UPDATE users 
        SET is_active = false 
        WHERE team_name = $1 AND is_active = true
    `

	result, err := r.storage.Exec(query, teamName)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return int(rowsAffected), nil
}

func isDuplicateKeyError(err error) bool {
	if err.Error() == "pq: duplicate key value violates unique constraint" {
		return true
	}
	return false
}
