package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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

	ErrorResponse struct {
		Error   string `json:"error"`
		Details string `json:"details,omitempty"`
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
		h.writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.UserID == "" {
		log.Error("user_id is required")
		h.writeError(w, http.StatusBadRequest, "user_id is required", nil)
		return
	}

	if !strings.HasPrefix(req.UserID, "u") {
		log.Error("invalid user_id format", slog.String("user_id", req.UserID))
		h.writeError(w, http.StatusBadRequest, "user_id must start with 'u'", nil)
		return
	}

	user, err := h.userService.SetUserActiveStatus(r.Context(), req.IsActive, req.UserID)
	if err != nil {
		log.Error("failed to set user active status", sl.Err(err))

		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, "user not found", err)
		} else {
			h.writeError(w, http.StatusInternalServerError, "failed to set user active status", err)
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
		h.writeError(w, http.StatusBadRequest, "user_id query parameter is required", nil)
		return
	}

	if !strings.HasPrefix(userID, "u") {
		log.Error("invalid user_id format", slog.String("user_id", userID))
		h.writeError(w, http.StatusBadRequest, "user_id must start with 'u'", nil)
		return
	}

	prs, err := h.userService.GetUserReview(r.Context(), userID)
	if err != nil {
		log.Error("failed to get user reviews", sl.Err(err))
		h.writeError(w, http.StatusInternalServerError, "failed to get user reviews", err)
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

func (h *UserHandler) convertUserID(userID string) (string, error) {
	if len(userID) > 1 && userID[0] == 'u' {
		return userID, nil
	}
	return userID, errors.New("invalid user_id format")
}

func (h *UserHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("Error encoding JSON response: %v\n", err)
	}
}

func (h *UserHandler) writeError(w http.ResponseWriter, status int, message string, err error) {
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
