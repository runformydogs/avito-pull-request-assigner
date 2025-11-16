package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"pull-request-assigner/internal/apperrors"
	"pull-request-assigner/internal/domain/models"
	"pull-request-assigner/internal/lib/logger/sl"
	"time"
)

type PullRequestService struct {
	log      *slog.Logger
	prRepo   PullRequestProvider
	teamRepo TeamProvider
}

type PullRequestProvider interface {
	CreatePR(pr models.PullRequest) error
	PRExists(prID string) (bool, error)
	GetPR(prID string) (*models.PullRequest, error)
	GetPRWithReviewers(prID string) (*models.PullRequest, []string, error)
	AddPRReviewers(prID string, reviewerIDs []string) error
	MergePR(prID string) error
	GetAuthorTeam(authorID string) (string, error)
	GetActiveTeamMembers(teamName string, excludeUserIDs []string) ([]string, error)
	ReplaceReviewer(prID string, oldReviewerID string, newReviewerID string) error
}

func NewPullRequestService(
	log *slog.Logger,
	prRepo PullRequestProvider,
	teamRepo TeamProvider) *PullRequestService {
	return &PullRequestService{
		log:      log,
		prRepo:   prRepo,
		teamRepo: teamRepo,
	}
}

func (s *PullRequestService) CreatePRWithReviewers(ctx context.Context, pr models.PullRequest) (*models.PullRequest, []string, error) {
	const op = "service.pullRequest.CreatePRWithReviewers"

	log := s.log.With(
		slog.String("op", op),
		slog.String("pr_id", pr.PullRequestId),
		slog.String("author_id", pr.AuthorID),
	)

	log.Info("attempting to create PR with reviewers")

	if pr.PullRequestId == "" {
		log.Error("pull request id is required")
		return nil, nil, apperrors.ErrPRIDRequired
	}

	if pr.PullRequestName == "" {
		log.Error("pull request name is required")
		return nil, nil, apperrors.ErrPRNameRequired
	}

	if pr.AuthorID == "" {
		log.Error("author id is required")
		return nil, nil, apperrors.ErrAuthorRequired
	}

	exists, err := s.prRepo.PRExists(pr.PullRequestId)
	if err != nil {
		log.Error("failed to check PR existence", sl.Err(err))
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	if exists {
		log.Warn("PR already exists", slog.String("pr_id", pr.PullRequestId))
		return nil, nil, apperrors.ErrPRExists
	}

	teamName, err := s.prRepo.GetAuthorTeam(pr.AuthorID)
	if err != nil {
		if errors.Is(err, apperrors.ErrPRAuthorNotFound) {
			log.Warn("author not found", slog.String("author_id", pr.AuthorID))
			return nil, nil, apperrors.ErrPRAuthorNotFound
		}
		log.Error("failed to get author team", sl.Err(err))
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	teamMembers, err := s.prRepo.GetActiveTeamMembers(teamName, []string{pr.AuthorID})
	if err != nil {
		log.Error("failed to get team members", sl.Err(err))
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	if len(teamMembers) == 0 {
		log.Warn("no active team members available for review")
		return nil, nil, apperrors.ErrNoReviewerCandidates
	}

	reviewers := s.selectRandomReviewers(teamMembers, 2)

	pr.Status = "OPEN"
	pr.CreatedAt = time.Now()

	err = s.prRepo.CreatePR(pr)
	if err != nil {
		log.Error("failed to create PR", sl.Err(err))
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	if len(reviewers) > 0 {
		err = s.prRepo.AddPRReviewers(pr.PullRequestId, reviewers)
		if err != nil {
			log.Error("failed to add PR reviewers", sl.Err(err))
			return nil, nil, fmt.Errorf("%s: %w", op, err)
		}
	}

	createdPR, assignedReviewers, err := s.prRepo.GetPRWithReviewers(pr.PullRequestId)
	if err != nil {
		log.Error("failed to get created PR", sl.Err(err))
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("PR created successfully",
		slog.Int("reviewer_count", len(assignedReviewers)))

	return createdPR, assignedReviewers, nil
}

func (s *PullRequestService) MergePR(ctx context.Context, prID string) (*models.PullRequest, []string, error) {
	const op = "service.pullRequest.MergePR"

	log := s.log.With(
		slog.String("op", op),
		slog.String("pr_id", prID),
	)

	log.Info("attempting to merge PR")

	if prID == "" {
		log.Error("pull request id is required")
		return nil, nil, apperrors.ErrPRIDRequired
	}

	err := s.prRepo.MergePR(prID)
	if err != nil {
		if errors.Is(err, apperrors.ErrPRNotFound) {
			log.Warn("PR not found", slog.String("pr_id", prID))
			return nil, nil, apperrors.ErrPRNotFound
		}
		log.Error("failed to merge PR", sl.Err(err))
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	mergedPR, reviewers, err := s.prRepo.GetPRWithReviewers(prID)
	if err != nil {
		log.Error("failed to get merged PR", sl.Err(err))
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("PR merged successfully")
	return mergedPR, reviewers, nil
}

func (s *PullRequestService) ReassignReviewer(ctx context.Context, prID string, oldReviewerID string) (*models.PullRequest, []string, string, error) {
	const op = "service.pullRequest.ReassignReviewer"

	log := s.log.With(
		slog.String("op", op),
		slog.String("pr_id", prID),
		slog.String("old_reviewer_id", oldReviewerID),
	)

	log.Info("attempting to reassign reviewer")

	if prID == "" {
		log.Error("pull request id is required")
		return nil, nil, "", apperrors.ErrPRIDRequired
	}

	if oldReviewerID == "" {
		log.Error("old reviewer id is required")
		return nil, nil, "", apperrors.ErrOldReviewerRequired
	}

	pr, reviewers, err := s.prRepo.GetPRWithReviewers(prID)
	if err != nil {
		if errors.Is(err, apperrors.ErrPRNotFound) {
			log.Warn("PR not found", slog.String("pr_id", prID))
			return nil, nil, "", apperrors.ErrPRNotFound
		}
		log.Error("failed to get PR", sl.Err(err))
		return nil, nil, "", fmt.Errorf("%s: %w", op, err)
	}

	if pr.Status == "MERGED" {
		log.Warn("cannot reassign reviewer on merged PR", slog.String("pr_id", prID))
		return nil, nil, "", apperrors.ErrPRAlreadyMerged
	}

	oldReviewerAssigned := false
	for _, reviewer := range reviewers {
		if reviewer == oldReviewerID {
			oldReviewerAssigned = true
			break
		}
	}

	if !oldReviewerAssigned {
		log.Warn("reviewer not assigned to this PR", slog.String("reviewer_id", oldReviewerID))
		return nil, nil, "", apperrors.ErrReviewerNotAssigned
	}

	teamName, err := s.prRepo.GetAuthorTeam(pr.AuthorID)
	if err != nil {
		if errors.Is(err, apperrors.ErrPRAuthorNotFound) {
			log.Warn("author not found", slog.String("author_id", pr.AuthorID))
			return nil, nil, "", apperrors.ErrPRAuthorNotFound
		}
		log.Error("failed to get author team", sl.Err(err))
		return nil, nil, "", fmt.Errorf("%s: %w", op, err)
	}

	exclude := append(reviewers, pr.AuthorID)
	availableMembers, err := s.prRepo.GetActiveTeamMembers(teamName, exclude)
	if err != nil {
		log.Error("failed to get available team members", sl.Err(err))
		return nil, nil, "", fmt.Errorf("%s: %w", op, err)
	}

	if len(availableMembers) == 0 {
		log.Warn("no available replacement candidates in team")
		return nil, nil, "", apperrors.ErrNoReviewerCandidates
	}

	newReviewer := s.selectRandomReviewer(availableMembers)

	err = s.prRepo.ReplaceReviewer(prID, oldReviewerID, newReviewer)
	if err != nil {
		log.Error("failed to replace reviewer", sl.Err(err))
		return nil, nil, "", fmt.Errorf("%s: %w", op, err)
	}

	updatedPR, updatedReviewers, err := s.prRepo.GetPRWithReviewers(prID)
	if err != nil {
		log.Error("failed to get updated PR", sl.Err(err))
		return nil, nil, "", fmt.Errorf("%s: %w", op, err)
	}

	log.Info("reviewer reassigned successfully",
		slog.String("new_reviewer", newReviewer))

	return updatedPR, updatedReviewers, newReviewer, nil
}

func (s *PullRequestService) selectRandomReviewers(members []string, max int) []string {
	if len(members) <= max {
		shuffled := make([]string, len(members))
		copy(shuffled, members)
		rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})
		return shuffled
	}

	selected := make([]string, max)
	available := make([]string, len(members))
	copy(available, members)

	rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(available), func(i, j int) {
		available[i], available[j] = available[j], available[i]
	})

	copy(selected, available[:max])
	return selected
}

func (s *PullRequestService) selectRandomReviewer(members []string) string {
	if len(members) == 0 {
		return ""
	}
	rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(members), func(i, j int) {
		members[i], members[j] = members[j], members[i]
	})
	return members[0]
}
