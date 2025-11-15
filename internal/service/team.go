package service

import (
	"context"
	"fmt"
	"log/slog"
	"pull-request-assigner/internal/domain/models"
	"pull-request-assigner/internal/lib/logger/sl"
)

type TeamService struct {
	log      *slog.Logger
	teamRepo TeamProvider
}

type TeamProvider interface {
	CreateTeam(teamName string) error
	TeamExists(teamName string) (bool, error)
	AddTeamMembers(teamName string, members []models.User) error
	GetTeamWithMembers(teamName string) (*models.Team, error)
}

func NewTeamService(
	log *slog.Logger,
	teamRepo TeamProvider) *TeamService {
	return &TeamService{
		log:      log,
		teamRepo: teamRepo,
	}
}

func (s *TeamService) CreateTeamWithMembers(ctx context.Context, team models.Team) (*models.Team, error) {
	const op = "service.team.CreateTeamWithMembers"

	log := s.log.With(
		slog.String("op", op),
		slog.String("team_name", team.TeamName),
	)

	log.Info("attempting to create team with members")

	if team.TeamName == "" {
		log.Error("team name is required")
		return nil, fmt.Errorf("%s: team name is required", op)
	}

	if len(team.Members) == 0 {
		log.Error("team must have at least one member")
		return nil, fmt.Errorf("%s: team must have at least one member", op)
	}

	exists, err := s.teamRepo.TeamExists(team.TeamName)
	if err != nil {
		log.Error("failed to check team existence", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if exists {
		log.Warn("team already exists")
		return nil, fmt.Errorf("%s: team %s already exists", op, team.TeamName)
	}

	err = s.teamRepo.CreateTeam(team.TeamName)
	if err != nil {
		log.Error("failed to create team", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = s.teamRepo.AddTeamMembers(team.TeamName, team.Members)
	if err != nil {
		log.Error("failed to add team members", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	createdTeam, err := s.teamRepo.GetTeamWithMembers(team.TeamName)
	if err != nil {
		log.Error("failed to get created team", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("team created successfully",
		slog.Int("member_count", len(createdTeam.Members)))

	return createdTeam, nil
}

func (s *TeamService) GetTeamWithMembers(ctx context.Context, teamName string) (*models.Team, error) {
	const op = "service.team.GetTeamWithMembers"

	log := s.log.With(
		slog.String("op", op),
		slog.String("team_name", teamName),
	)

	log.Info("attempting to get team with members")

	if teamName == "" {
		log.Error("team name is required")
		return nil, fmt.Errorf("%s: team name is required", op)
	}

	team, err := s.teamRepo.GetTeamWithMembers(teamName)
	if err != nil {
		log.Error("failed to get team", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("team retrieved successfully",
		slog.Int("member_count", len(team.Members)))

	return team, nil
}
