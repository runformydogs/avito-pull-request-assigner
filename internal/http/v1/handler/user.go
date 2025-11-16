package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"pull-request-assigner/internal/apperrors"
	"pull-request-assigner/internal/domain/models"
	"pull-request-assigner/internal/lib/logger/sl"
	"pull-request-assigner/internal/service"
	"strings"
)

type (
	SetIsActiveRequest struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}

	GetReviewRequest struct {
		UserID string `json:"user_id"`
	}

	SetIsActiveResponse struct {
		User models.User `json:"user"`
	}

	GetReviewResponse struct {
		UserID       string                    `json:"user_id"`
		PullRequests []models.PullRequestShort `json:"pull_requests"`
	}

	UserErrorResponse struct {
		Error UserErrorDetail `json:"error"`
	}

	UserErrorDetail struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
)

type UserHandler struct {
	userService *service.UserService
	log         *slog.Logger
}

func NewUserHandler(userService *service.UserService, log *slog.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		log:         log,
	}
}

func (h *UserHandler) SetIsActive(w http.ResponseWriter, r *http.Request) {
	const op = "handler.user.setIsActive"

	log := h.log.With(
		slog.String("op", op),
	)

	var req SetIsActiveRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("invalid request body", sl.Err(err))
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if req.UserID == "" {
		log.Error("user_id is required")
		h.writeErrorResponse(w, http.StatusBadRequest, "USER_ID_REQUIRED", "user_id is required")
		return
	}

	if !strings.HasPrefix(req.UserID, "u") {
		log.Error("invalid user_id format", slog.String("user_id", req.UserID))
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_USER_ID", "user_id must start with 'u'")
		return
	}

	user, err := h.userService.SetUserActiveStatus(r.Context(), req.IsActive, req.UserID)
	if err != nil {
		log.Error("failed to set user active status", sl.Err(err))

		switch {
		case errors.Is(err, apperrors.ErrUserNotFound):
			h.writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		case errors.Is(err, apperrors.ErrInvalidUserID):
			h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_USER_ID", "invalid user_id format")
		default:
			h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set user active status")
		}
		return
	}

	response := SetIsActiveResponse{
		User: user,
	}

	h.writeJSON(w, http.StatusOK, response)
	log.Info("user active status updated successfully")
}

func (h *UserHandler) GetReview(w http.ResponseWriter, r *http.Request) {
	const op = "handler.user.getReview"

	log := h.log.With(
		slog.String("op", op),
	)

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		log.Error("user_id is required")
		h.writeErrorResponse(w, http.StatusBadRequest, "USER_ID_REQUIRED", "user_id query parameter is required")
		return
	}

	if !strings.HasPrefix(userID, "u") {
		log.Error("invalid user_id format", slog.String("user_id", userID))
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_USER_ID", "user_id must start with 'u'")
		return
	}

	prs, err := h.userService.GetUserReview(r.Context(), userID)
	if err != nil {
		log.Error("failed to get user reviews", sl.Err(err))

		switch {
		case errors.Is(err, apperrors.ErrInvalidUserID):
			h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_USER_ID", "invalid user_id format")
		default:
			h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get user reviews")
		}
		return
	}

	response := GetReviewResponse{
		UserID:       userID,
		PullRequests: prs,
	}

	h.writeJSON(w, http.StatusOK, response)
	log.Info("user reviews retrieved successfully",
		slog.Int("pull_request_count", len(prs)))
}

func (h *UserHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Error("failed to encode JSON response", sl.Err(err))
	}
}

func (h *UserHandler) writeErrorResponse(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResp := UserErrorResponse{
		Error: UserErrorDetail{
			Code:    code,
			Message: message,
		},
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		h.log.Error("failed to encode error response", sl.Err(err))
	}
}
