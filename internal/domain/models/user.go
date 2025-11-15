package models

type User struct {
	UserID   string `db:"user_id" json:"user_id"`
	Username string `db:"username" json:"username"`
	TeamName string `db:"team_name" json:"team_name"`
	IsActive bool   `db:"is_active" json:"is_active"`
}
