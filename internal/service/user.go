package service

import (
	"context"
	"fmt"
	"log/slog"
	"pull-request-assigner/internal/domain/models"
	"pull-request-assigner/internal/lib/logger/sl"
	"strconv"
)

type UserService struct {
	log          *slog.Logger
	userProvider UserProvider
}

type UserProvider interface {
	SetIsActive(isActive bool, userID int) (models.User, error)
	GetReview(userID int) ([]models.PullRequestShort, error)
}

func New(
	log *slog.Logger,
	userProvider UserProvider) *UserService {
	return &UserService{
		log:          log,
		userProvider: userProvider,
	}
}

func (s *UserService) SetUserActiveStatus(ctx context.Context, isActive bool, userID string) (models.User, error) {
	const op = "service.user.SetUserActiveStatus"

	log := s.log.With(
		slog.String("op", op),
		slog.String("userID", userID),
		slog.Bool("isActive", isActive),
	)

	log.Info("attempting to change user active status")

	userIDInt, err := strconv.Atoi(userID[1:])
	if err != nil {
		log.Error("invalid user ID format", sl.Err(err))
		return models.User{}, fmt.Errorf("%s: invalid user ID format: %w", op, err)
	}

	user, err := s.userProvider.SetIsActive(isActive, userIDInt)
	if err != nil {
		log.Error("failed to set user active status", sl.Err(err))
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	status := "active"
	if !isActive {
		status = "not active"
	}
	log.Info("user status changed successfully", slog.String("status", status))

	return user, nil
}

func (s *UserService) GetUserReview(ctx context.Context, userID string) ([]models.PullRequestShort, error) {
	const op = "service.user.GetUserReviews"

	log := s.log.With(
		slog.String("op", op),
		slog.String("userID", userID),
	)

	log.Info("attempting to get user reviews")

	userIDInt, err := strconv.Atoi(userID[1:])
	if err != nil {
		log.Error("invalid user ID format", sl.Err(err))
		return nil, fmt.Errorf("%s: invalid user ID format: %w", op, err)
	}

	prs, err := s.userProvider.GetReview(userIDInt)
	if err != nil {
		log.Error("failed to get reviews", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("successfully retrieved user reviews",
		slog.Int("pullRequestCount", len(prs)))

	return prs, nil
}
