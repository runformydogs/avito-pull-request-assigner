package service

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"pull-request-assigner/internal/domain/models"
	"pull-request-assigner/internal/lib/logger/sl"
	"time"
)

type PullRequestService struct {
	log    *slog.Logger
	prRepo PullRequestProvider
}

type PullRequestProvider interface {
	CreatePullRequest(pr models.PullRequest, reviewers []string) error
	GetPullRequest(prID string) (*models.PullRequest, error)
	PRExists(prID string) (bool, error)
	MergePullRequest(prID string) (*models.PullRequest, error)
	GetAssignedReviewers(prID string) ([]string, error)
	GetTeamMembers(teamName string) ([]models.User, error)
	GetAuthorTeam(authorID string) (string, error)
	ReassignReviewer(prID string, oldReviewerID string, newReviewerID string) error
}

func NewPullRequestService(
	log *slog.Logger,
	prRepo PullRequestProvider) *PullRequestService {
	return &PullRequestService{
		log:    log,
		prRepo: prRepo,
	}
}

type (
	CreatePullRequestResponse struct {
		PR                models.PullRequest `json:"pr"`
		AssignedReviewers []string           `json:"assigned_reviewers"`
	}

	ReassignReviewerResponse struct {
		PR         models.PullRequest `json:"pr"`
		ReplacedBy string             `json:"replaced_by"`
	}
)

func (s *PullRequestService) CreatePullRequest(ctx context.Context, pr models.PullRequest) (*CreatePullRequestResponse, error) {
	const op = "service.pullRequest.CreatePullRequest"

	log := s.log.With(
		slog.String("op", op),
		slog.String("pr_id", pr.PullRequestId),
		slog.String("author_id", pr.AuthorID),
	)

	log.Info("attempting to create pull request")

	exists, err := s.prRepo.PRExists(pr.PullRequestId)
	if err != nil {
		log.Error("failed to check PR existence", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if exists {
		log.Warn("PR already exists")
		return nil, fmt.Errorf("%s: PR already exists", op)
	}

	teamName, err := s.prRepo.GetAuthorTeam(pr.AuthorID)
	if err != nil {
		log.Error("failed to get author team", sl.Err(err))
		return nil, fmt.Errorf("%s: author/team not found: %w", op, err)
	}

	teamMembers, err := s.prRepo.GetTeamMembers(teamName)
	if err != nil {
		log.Error("failed to get team members", sl.Err(err))
		return nil, fmt.Errorf("%s: failed to get team members: %w", op, err)
	}

	var candidates []models.User
	for _, member := range teamMembers {
		if member.UserID != pr.AuthorID {
			candidates = append(candidates, member)
		}
	}

	if len(candidates) == 0 {
		log.Error("no available reviewers in team")
		return nil, fmt.Errorf("%s: no available reviewers in team", op)
	}

	reviewers := s.selectRandomReviewers(candidates, 2)
	reviewerIDs := make([]string, len(reviewers))
	for i, reviewer := range reviewers {
		reviewerIDs[i] = reviewer.UserID
	}

	pr.Status = "OPEN"
	pr.CreatedAt = time.Now()

	err = s.prRepo.CreatePullRequest(pr, reviewerIDs)
	if err != nil {
		log.Error("failed to create PR", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("PR created successfully",
		slog.Int("reviewer_count", len(reviewerIDs)),
		slog.Any("reviewers", reviewerIDs))

	response := &CreatePullRequestResponse{
		PR:                pr,
		AssignedReviewers: reviewerIDs,
	}

	return response, nil
}

func (s *PullRequestService) MergePullRequest(ctx context.Context, prID string) (*models.PullRequest, error) {
	const op = "service.pullrequest.MergePullRequest"

	log := s.log.With(
		slog.String("op", op),
		slog.String("pr_id", prID),
	)

	log.Info("attempting to merge pull request")

	if prID == "" {
		log.Error("pull_request_id is required")
		return nil, fmt.Errorf("%s: pull_request_id is required", op)
	}

	pr, err := s.prRepo.MergePullRequest(prID)
	if err != nil {
		log.Error("failed to merge PR", sl.Err(err))
		if err.Error() == fmt.Sprintf("%s: PR not found", op) {
			return nil, fmt.Errorf("%s: PR not found", op)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("PR merged successfully")
	return pr, nil
}

func (s *PullRequestService) ReassignReviewer(ctx context.Context, prID string, oldReviewerID string) (*ReassignReviewerResponse, error) {
	const op = "service.pullRequest.ReassignReviewer"

	log := s.log.With(
		slog.String("op", op),
		slog.String("pr_id", prID),
		slog.String("old_reviewer_id", oldReviewerID),
	)

	log.Info("attempting to reassign reviewer")

	// Валидация
	if prID == "" {
		log.Error("pull_request_id is required")
		return nil, fmt.Errorf("%s: pull_request_id is required", op)
	}
	if oldReviewerID == "" {
		log.Error("old_reviewer_id is required")
		return nil, fmt.Errorf("%s: old_reviewer_id is required", op)
	}

	// Получаем информацию о PR
	pr, err := s.prRepo.GetPullRequest(prID)
	if err != nil {
		log.Error("failed to get PR", sl.Err(err))
		return nil, fmt.Errorf("%s: PR not found", op)
	}

	// Проверяем, что PR не мерджен
	if pr.Status == "MERGED" {
		log.Error("cannot reassign on merged PR")
		return nil, fmt.Errorf("%s: cannot reassign on merged PR", op)
	}

	// Получаем текущих ревьюверов
	currentReviewers, err := s.prRepo.GetAssignedReviewers(prID)
	if err != nil {
		log.Error("failed to get assigned reviewers", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Проверяем, что старый ревьювер назначен на PR
	found := false
	for _, reviewer := range currentReviewers {
		if reviewer == oldReviewerID {
			found = true
			break
		}
	}
	if !found {
		log.Error("reviewer is not assigned to this PR")
		return nil, fmt.Errorf("%s: reviewer is not assigned to this PR", op)
	}

	// Получаем команду автора
	teamName, err := s.prRepo.GetAuthorTeam(pr.AuthorID)
	if err != nil {
		log.Error("failed to get author team", sl.Err(err))
		return nil, fmt.Errorf("%s: failed to get author team: %w", op, err)
	}

	// Получаем активных членов команды
	teamMembers, err := s.prRepo.GetTeamMembers(teamName)
	if err != nil {
		log.Error("failed to get team members", sl.Err(err))
		return nil, fmt.Errorf("%s: failed to get team members: %w", op, err)
	}

	// Фильтруем доступных кандидатов (исключая автора, старого ревьювера и текущих ревьюверов)
	var candidates []models.User
	for _, member := range teamMembers {
		if member.UserID != pr.AuthorID &&
			member.UserID != oldReviewerID &&
			!contains(currentReviewers, member.UserID) {
			candidates = append(candidates, member)
		}
	}

	if len(candidates) == 0 {
		log.Error("no active replacement candidate in team")
		return nil, fmt.Errorf("%s: no active replacement candidate in team", op)
	}

	// Выбираем случайного кандидата
	newReviewer := candidates[rand.Intn(len(candidates))]

	// Выполняем переназначение
	err = s.prRepo.ReassignReviewer(prID, oldReviewerID, newReviewer.UserID)
	if err != nil {
		log.Error("failed to reassign reviewer", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	//updatedReviewers, err := s.prRepo.GetAssignedReviewers(prID)
	//if err != nil {
	//	log.Error("failed to get updated reviewers", sl.Err(err))
	//	return nil, fmt.Errorf("%s: %w", op, err)
	//}

	log.Info("reviewer reassigned successfully",
		slog.String("new_reviewer", newReviewer.UserID))

	response := &ReassignReviewerResponse{
		PR:         *pr,
		ReplacedBy: newReviewer.UserID,
	}

	return response, nil
}

func (s *PullRequestService) selectRandomReviewers(candidates []models.User, max int) []models.User {
	if len(candidates) <= max {
		shuffled := make([]models.User, len(candidates))
		copy(shuffled, candidates)
		rand.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})
		return shuffled
	}

	selected := make([]models.User, 0, max)
	used := make(map[int]bool)

	for len(selected) < max && len(selected) < len(candidates) {
		idx := rand.Intn(len(candidates))
		if !used[idx] {
			selected = append(selected, candidates[idx])
			used[idx] = true
		}
	}

	return selected
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
