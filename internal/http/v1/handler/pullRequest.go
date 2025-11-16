package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"pull-request-assigner/internal/apperrors"
	"pull-request-assigner/internal/domain/models"
	"pull-request-assigner/internal/lib/logger/sl"
	"pull-request-assigner/internal/service"
	"time"
)

type (
	CreatePRRequest struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}

	CreatePRResponse struct {
		PR *PullRequestWithReviewers `json:"pr"`
	}

	MergePRRequest struct {
		PullRequestID string `json:"pull_request_id"`
	}

	MergePRResponse struct {
		PR *PullRequestWithReviewers `json:"pr"`
	}

	ReassignReviewerRequest struct {
		PullRequestID string `json:"pull_request_id"`
		OldReviewerID string `json:"old_reviewer_id"`
	}

	ReassignReviewerResponse struct {
		PR         *PullRequestWithReviewers `json:"pr"`
		ReplacedBy string                    `json:"replaced_by"`
	}

	PullRequestWithReviewers struct {
		PullRequestID     string   `json:"pull_request_id"`
		PullRequestName   string   `json:"pull_request_name"`
		AuthorID          string   `json:"author_id"`
		Status            string   `json:"status"`
		AssignedReviewers []string `json:"assigned_reviewers"`
		MergedAt          string   `json:"mergedAt,omitempty"`
	}

	PRErrorResponse struct {
		Error PRErrorDetail `json:"error"`
	}

	PRErrorDetail struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
)

type PullRequestHandler struct {
	prService *service.PullRequestService
	log       *slog.Logger
}

func NewPullRequestHandler(prService *service.PullRequestService, log *slog.Logger) *PullRequestHandler {
	return &PullRequestHandler{
		prService: prService,
		log:       log,
	}
}

func (h *PullRequestHandler) CreatePR(w http.ResponseWriter, r *http.Request) {
	const op = "handler.pullRequest.CreatePR"

	log := h.log.With(slog.String("op", op))

	var req CreatePRRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("invalid request body", sl.Err(err))
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if req.PullRequestID == "" {
		log.Error("pull_request_id is required")
		h.writeErrorResponse(w, http.StatusBadRequest, "PR_ID_REQUIRED", "pull_request_id is required")
		return
	}

	if req.PullRequestName == "" {
		log.Error("pull_request_name is required")
		h.writeErrorResponse(w, http.StatusBadRequest, "PR_NAME_REQUIRED", "pull_request_name is required")
		return
	}

	if req.AuthorID == "" {
		log.Error("author_id is required")
		h.writeErrorResponse(w, http.StatusBadRequest, "AUTHOR_REQUIRED", "author_id is required")
		return
	}

	pr := models.PullRequest{
		PullRequestId:   req.PullRequestID,
		PullRequestName: req.PullRequestName,
		AuthorID:        req.AuthorID,
	}

	createdPR, reviewers, err := h.prService.CreatePRWithReviewers(r.Context(), pr)
	if err != nil {
		log.Error("failed to create PR", sl.Err(err))

		switch {
		case errors.Is(err, apperrors.ErrPRExists):
			h.writeErrorResponse(w, http.StatusConflict, "PR_EXISTS",
				fmt.Sprintf("PR %s already exists", req.PullRequestID))
		case errors.Is(err, apperrors.ErrPRAuthorNotFound):
			h.writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		case errors.Is(err, apperrors.ErrPRTeamNotFound):
			h.writeErrorResponse(w, http.StatusNotFound, "TEAM_NOT_FOUND", "author team not found")
		case errors.Is(err, apperrors.ErrNoReviewerCandidates):
			h.writeErrorResponse(w, http.StatusNotFound, "NO_REVIEWERS", "no active reviewers available in team")
		default:
			h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create PR")
		}
		return
	}

	response := CreatePRResponse{
		PR: &PullRequestWithReviewers{
			PullRequestID:     createdPR.PullRequestId,
			PullRequestName:   createdPR.PullRequestName,
			AuthorID:          createdPR.AuthorID,
			Status:            createdPR.Status,
			AssignedReviewers: reviewers,
			MergedAt:          formatMergedAt(createdPR.MergedAt),
		},
	}

	h.writeJSON(w, http.StatusCreated, response)
	log.Info("PR created successfully")
}

func (h *PullRequestHandler) MergePR(w http.ResponseWriter, r *http.Request) {
	const op = "handler.pullRequest.MergePR"

	log := h.log.With(slog.String("op", op))

	var req MergePRRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("invalid request body", sl.Err(err))
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if req.PullRequestID == "" {
		log.Error("pull_request_id is required")
		h.writeErrorResponse(w, http.StatusBadRequest, "PR_ID_REQUIRED", "pull_request_id is required")
		return
	}

	mergedPR, reviewers, err := h.prService.MergePR(r.Context(), req.PullRequestID)
	if err != nil {
		log.Error("failed to merge PR", sl.Err(err))

		switch {
		case errors.Is(err, apperrors.ErrPRNotFound):
			h.writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		default:
			h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to merge PR")
		}
		return
	}

	response := MergePRResponse{
		PR: &PullRequestWithReviewers{
			PullRequestID:     mergedPR.PullRequestId,
			PullRequestName:   mergedPR.PullRequestName,
			AuthorID:          mergedPR.AuthorID,
			Status:            mergedPR.Status,
			AssignedReviewers: reviewers,
			MergedAt:          formatMergedAt(mergedPR.MergedAt),
		},
	}

	h.writeJSON(w, http.StatusOK, response)
	log.Info("PR merged successfully")
}

func (h *PullRequestHandler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	const op = "handler.pullRequest.ReassignReviewer"

	log := h.log.With(slog.String("op", op))

	var req ReassignReviewerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("invalid request body", sl.Err(err))
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if req.PullRequestID == "" {
		log.Error("pull_request_id is required")
		h.writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return
	}

	if req.OldReviewerID == "" {
		log.Error("old_reviewer_id is required")
		h.writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return
	}

	updatedPR, reviewers, newReviewer, err := h.prService.ReassignReviewer(r.Context(), req.PullRequestID, req.OldReviewerID)
	if err != nil {
		log.Error("failed to reassign reviewer", sl.Err(err))

		switch {
		case errors.Is(err, apperrors.ErrPRNotFound):
			h.writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		case errors.Is(err, apperrors.ErrPRAlreadyMerged):
			h.writeErrorResponse(w, http.StatusConflict, "PR_MERGED", "cannot reassign on merged PR")
		case errors.Is(err, apperrors.ErrReviewerNotAssigned):
			h.writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		case errors.Is(err, apperrors.ErrNoReviewerCandidates):
			h.writeErrorResponse(w, http.StatusConflict, "NO_CANDIDATE", "no active replacement candidate in team")
		default:
			h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to reassign reviewer")
		}
		return
	}

	response := ReassignReviewerResponse{
		PR: &PullRequestWithReviewers{
			PullRequestID:     updatedPR.PullRequestId,
			PullRequestName:   updatedPR.PullRequestName,
			AuthorID:          updatedPR.AuthorID,
			Status:            updatedPR.Status,
			AssignedReviewers: reviewers,
			MergedAt:          formatMergedAt(updatedPR.MergedAt),
		},
		ReplacedBy: newReviewer,
	}

	h.writeJSON(w, http.StatusOK, response)
	log.Info("reviewer reassigned successfully")
}

func (h *PullRequestHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Error("failed to encode JSON response", sl.Err(err))
	}
}

func (h *PullRequestHandler) writeErrorResponse(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResp := PRErrorResponse{
		Error: PRErrorDetail{
			Code:    code,
			Message: message,
		},
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		h.log.Error("failed to encode error response", sl.Err(err))
	}
}

func formatMergedAt(mergedAt sql.NullTime) string {
	if mergedAt.Valid {
		return mergedAt.Time.Format(time.RFC3339)
	}
	return ""
}
