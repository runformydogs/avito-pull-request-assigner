package models

type Team struct {
	TeamName string `db:"team_name" json:"team_name"`
	Members  []User `db:"-" json:"members"`
}

type TeamMember struct {
	TeamName string `db:"team_name"`
	UserID   string `db:"user_id"`
}
