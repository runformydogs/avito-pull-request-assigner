package apperrors

import "errors"

var (
	ErrTeamExists       = errors.New("team already exists")
	ErrTeamNotFound     = errors.New("team not found")
	ErrTeamNameRequired = errors.New("team name is required")
	ErrMembersRequired  = errors.New("team must have at least one member")
)
