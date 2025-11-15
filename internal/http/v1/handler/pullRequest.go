package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"pull-request-assigner/internal/domain/models"
	"pull-request-assigner/internal/lib/logger/sl"
	"pull-request-assigner/internal/service"
)

type (
	CreatePullRequestRequest struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}

	CreatePullRequestResponse struct {
		PR                models.PullRequest `json:"pr"`
		AssignedReviewers []string           `json:"assigned_reviewers"`
	}

	MergePullRequestRequest struct {
		PullRequestID string `json:"pull_request_id"`
	}

	MergePullRequestResponse struct {
		PR models.PullRequest `json:"pr"`
	}

	ReassignReviewerRequest struct {
		PullRequestID string `json:"pull_request_id"`
		OldReviewerID string `json:"old_reviewer_id"`
	}

	ReassignReviewerResponse struct {
		PR         models.PullRequest `json:"pr"`
		ReplacedBy string             `json:"replaced_by"`
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

func (h *PullRequestHandler) CreatePullRequest(w http.ResponseWriter, r *http.Request) {
	const op = "handler.pullRequest.CreatePullRequest"

	log := h.log.With(
		slog.String("op", op),
	)

	var req CreatePullRequestRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("invalid request body", sl.Err(err))
		h.writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.PullRequestID == "" {
		log.Error("pull_request_id is required")
		h.writeError(w, http.StatusBadRequest, "pull_request_id is required", nil)
		return
	}
	if req.PullRequestName == "" {
		log.Error("pull_request_name is required")
		h.writeError(w, http.StatusBadRequest, "pull_request_name is required", nil)
		return
	}
	if req.AuthorID == "" {
		log.Error("author_id is required")
		h.writeError(w, http.StatusBadRequest, "author_id is required", nil)
		return
	}

	pr := models.PullRequest{
		PullRequestId:   req.PullRequestID,
		PullRequestName: req.PullRequestName,
		AuthorID:        req.AuthorID,
	}

	response, err := h.prService.CreatePullRequest(r.Context(), pr)
	if err != nil {
		log.Error("failed to create pull request", sl.Err(err))

		switch err.Error() {
		case fmt.Sprintf("%s: PR already exists", op):
			h.writeError(w, http.StatusConflict, "PR id already exists", err)
		case fmt.Sprintf("%s: author/team not found", op):
			h.writeError(w, http.StatusNotFound, "author/team not found", err)
		default:
			h.writeError(w, http.StatusInternalServerError, "failed to create pull request", err)
		}
		return
	}

	h.writeJSON(w, http.StatusCreated, response)
	log.Info("pull request created successfully")
}

func (h *PullRequestHandler) MergePullRequest(w http.ResponseWriter, r *http.Request) {
	const op = "handler.pullRequest.MergePullRequest"

	log := h.log.With(
		slog.String("op", op),
	)

	var req MergePullRequestRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("invalid request body", sl.Err(err))
		h.writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.PullRequestID == "" {
		log.Error("pull_request_id is required")
		h.writeError(w, http.StatusBadRequest, "pull_request_id is required", nil)
		return
	}

	pr, err := h.prService.MergePullRequest(r.Context(), req.PullRequestID)
	if err != nil {
		log.Error("failed to merge pull request", sl.Err(err))

		if err.Error() == fmt.Sprintf("%s: PR not found", op) {
			h.writeError(w, http.StatusNotFound, "PR not found", err)
		} else {
			h.writeError(w, http.StatusInternalServerError, "failed to merge pull request", err)
		}
		return
	}

	response := MergePullRequestResponse{
		PR: *pr,
	}

	h.writeJSON(w, http.StatusOK, response)
	log.Info("pull request merged successfully")
}

func (h *PullRequestHandler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	const op = "handler.pullRequest.ReassignReviewer"

	log := h.log.With(
		slog.String("op", op),
	)

	var req ReassignReviewerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("invalid request body", sl.Err(err))
		h.writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.PullRequestID == "" {
		log.Error("pull_request_id is required")
		h.writeError(w, http.StatusBadRequest, "pull_request_id is required", nil)
		return
	}
	if req.OldReviewerID == "" {
		log.Error("old_reviewer_id is required")
		h.writeError(w, http.StatusBadRequest, "old_reviewer_id is required", nil)
		return
	}

	response, err := h.prService.ReassignReviewer(r.Context(), req.PullRequestID, req.OldReviewerID)
	if err != nil {
		log.Error("failed to reassign reviewer", sl.Err(err))

		switch err.Error() {
		case fmt.Sprintf("%s: PR not found", op):
			h.writeError(w, http.StatusNotFound, "PR not found", err)
		case fmt.Sprintf("%s: cannot reassign on merged PR", op):
			h.writeError(w, http.StatusConflict, "cannot reassign on merged PR", err)
		case fmt.Sprintf("%s: reviewer is not assigned to this PR", op):
			h.writeError(w, http.StatusConflict, "reviewer is not assigned to this PR", err)
		case fmt.Sprintf("%s: no active replacement candidate in team", op):
			h.writeError(w, http.StatusConflict, "no active replacement candidate in team", err)
		default:
			h.writeError(w, http.StatusInternalServerError, "failed to reassign reviewer", err)
		}
		return
	}

	h.writeJSON(w, http.StatusOK, response)
	log.Info("reviewer reassigned successfully")
}

func (h *PullRequestHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("Error encoding JSON response: %v\n", err)
	}
}

func (h *PullRequestHandler) writeError(w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResp := ErrorResponse{
		Error: message,
	}
	if err != nil {
		errorResp.Details = err.Error()
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		fmt.Printf("Error encoding error response: %v\n", err)
	}
}
