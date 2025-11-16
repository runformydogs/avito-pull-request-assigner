package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"pull-request-assigner/internal/apperrors"
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
	DeactivateTeamUsers(teamName string) (int, error)
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
		return nil, apperrors.ErrTeamNameRequired
	}

	if len(team.Members) == 0 {
		log.Error("team must have at least one member")
		return nil, apperrors.ErrMembersRequired
	}

	for i, member := range team.Members {
		if member.UserID == "" {
			return nil, fmt.Errorf("%s: user_id is required for member at index %d", op, i)
		}
		if member.Username == "" {
			return nil, fmt.Errorf("%s: username is required for member at index %d", op, i)
		}
	}

	exists, err := s.teamRepo.TeamExists(team.TeamName)
	if err != nil {
		log.Error("failed to check team existence", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if exists {
		log.Warn("team already exists", slog.String("team_name", team.TeamName))
		return nil, apperrors.ErrTeamExists
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
		return nil, apperrors.ErrTeamNameRequired
	}

	team, err := s.teamRepo.GetTeamWithMembers(teamName)
	if err != nil {
		if errors.Is(err, apperrors.ErrTeamNotFound) {
			log.Warn("team not found", slog.String("team_name", teamName))
			return nil, apperrors.ErrTeamNotFound
		}
		log.Error("failed to get team", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("team retrieved successfully",
		slog.Int("member_count", len(team.Members)))

	return team, nil
}

func (s *TeamService) DeactivateTeamUsers(ctx context.Context, teamName string) (int, error) {
	const op = "service.team.DeactivateTeamUsers"

	log := s.log.With(
		slog.String("op", op),
		slog.String("team_name", teamName),
	)

	log.Info("attempting to deactivate team users")

	if teamName == "" {
		log.Error("team name is required")
		return 0, apperrors.ErrTeamNameRequired
	}

	exists, err := s.teamRepo.TeamExists(teamName)
	if err != nil {
		log.Error("failed to check team existence", sl.Err(err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	if !exists {
		log.Warn("team not found", slog.String("team_name", teamName))
		return 0, apperrors.ErrTeamNotFound
	}

	deactivatedCount, err := s.teamRepo.DeactivateTeamUsers(teamName)
	if err != nil {
		log.Error("failed to deactivate team users", sl.Err(err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("team users deactivated successfully",
		slog.Int("deactivated_count", deactivatedCount))

	return deactivatedCount, nil
}
