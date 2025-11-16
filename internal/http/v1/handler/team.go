package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"pull-request-assigner/internal/apperrors"
	"pull-request-assigner/internal/domain/models"
	"pull-request-assigner/internal/lib/logger/sl"
	"pull-request-assigner/internal/service"
)

type (
	CreateTeamRequest struct {
		TeamName string        `json:"team_name"`
		Members  []models.User `json:"members"`
	}

	CreateTeamResponse struct {
		TeamName string        `json:"team_name"`
		Members  []models.User `json:"members"`
	}

	GetTeamResponse struct {
		TeamName string        `json:"team_name"`
		Members  []models.User `json:"members"`
	}

	TeamErrorResponse struct {
		Error TeamErrorDetail `json:"error"`
	}

	TeamErrorDetail struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	DeactivateTeamUsersResponse struct {
		TeamName         string `json:"team_name"`
		DeactivatedUsers int    `json:"deactivated_users"`
	}
)

type TeamHandler struct {
	teamService *service.TeamService
	log         *slog.Logger
}

func NewTeamHandler(teamService *service.TeamService, log *slog.Logger) *TeamHandler {
	return &TeamHandler{
		teamService: teamService,
		log:         log,
	}
}

func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	const op = "handler.team.CreateTeam"

	log := h.log.With(
		slog.String("op", op),
	)

	var req CreateTeamRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("invalid request body", sl.Err(err))
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if req.TeamName == "" {
		log.Error("team_name is required")
		h.writeErrorResponse(w, http.StatusBadRequest, "TEAM_NAME_REQUIRED", "team_name is required")
		return
	}

	if len(req.Members) == 0 {
		log.Error("team must have at least one member")
		h.writeErrorResponse(w, http.StatusBadRequest, "MEMBERS_REQUIRED", "team must have at least one member")
		return
	}

	for i, member := range req.Members {
		if member.UserID == "" {
			h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_MEMBER",
				fmt.Sprintf("user_id is required for member at index %d", i))
			return
		}
		if member.Username == "" {
			h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_MEMBER",
				fmt.Sprintf("username is required for member at index %d", i))
			return
		}
	}

	team := models.Team{
		TeamName: req.TeamName,
		Members:  req.Members,
	}

	createdTeam, err := h.teamService.CreateTeamWithMembers(r.Context(), team)
	if err != nil {
		log.Error("failed to create team", sl.Err(err))

		switch {
		case errors.Is(err, apperrors.ErrTeamExists):
			h.writeErrorResponse(w, http.StatusBadRequest, "TEAM_EXISTS",
				fmt.Sprintf("team %s already exists", req.TeamName))
		case errors.Is(err, apperrors.ErrTeamNameRequired):
			h.writeErrorResponse(w, http.StatusBadRequest, "TEAM_NAME_REQUIRED", "team_name is required")
		case errors.Is(err, apperrors.ErrMembersRequired):
			h.writeErrorResponse(w, http.StatusBadRequest, "MEMBERS_REQUIRED", "team must have at least one member")
		default:
			h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create team")
		}
		return
	}

	response := CreateTeamResponse{
		TeamName: createdTeam.TeamName,
		Members:  createdTeam.Members,
	}

	h.writeJSON(w, http.StatusCreated, response)
	log.Info("team created successfully")
}

func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	const op = "handler.team.GetTeam"

	log := h.log.With(
		slog.String("op", op),
	)

	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		log.Error("team_name is required")
		h.writeErrorResponse(w, http.StatusBadRequest, "TEAM_NAME_REQUIRED", "team_name query parameter is required")
		return
	}

	team, err := h.teamService.GetTeamWithMembers(r.Context(), teamName)
	if err != nil {
		log.Error("failed to get team", sl.Err(err))

		switch {
		case errors.Is(err, apperrors.ErrTeamNotFound):
			h.writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		case errors.Is(err, apperrors.ErrTeamNameRequired):
			h.writeErrorResponse(w, http.StatusBadRequest, "TEAM_NAME_REQUIRED", "team_name is required")
		default:
			h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get team")
		}
		return
	}

	response := GetTeamResponse{
		TeamName: team.TeamName,
		Members:  team.Members,
	}

	h.writeJSON(w, http.StatusOK, response)
	log.Info("team retrieved successfully")
}

func (h *TeamHandler) DeactivateTeamUsers(w http.ResponseWriter, r *http.Request) {
	const op = "handler.team.DeactivateTeamUsers"

	log := h.log.With(
		slog.String("op", op),
	)

	// Получаем team_name из query параметров (как в GetTeam)
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		log.Error("team_name is required")
		h.writeErrorResponse(w, http.StatusBadRequest, "TEAM_NAME_REQUIRED", "team_name query parameter is required")
		return
	}

	deactivatedCount, err := h.teamService.DeactivateTeamUsers(r.Context(), teamName)
	if err != nil {
		log.Error("failed to deactivate team users", sl.Err(err))

		switch {
		case errors.Is(err, apperrors.ErrTeamNotFound):
			h.writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		case errors.Is(err, apperrors.ErrTeamNameRequired):
			h.writeErrorResponse(w, http.StatusBadRequest, "TEAM_NAME_REQUIRED", "team_name is required")
		default:
			h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to deactivate team users")
		}
		return
	}

	response := DeactivateTeamUsersResponse{
		TeamName:         teamName,
		DeactivatedUsers: deactivatedCount,
	}

	h.writeJSON(w, http.StatusOK, response)
	log.Info("team users deactivated successfully",
		slog.String("team_name", teamName),
		slog.Int("deactivated_count", deactivatedCount))
}

func (h *TeamHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Error("failed to encode JSON response", sl.Err(err))
	}
}

func (h *TeamHandler) writeErrorResponse(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResp := TeamErrorResponse{
		Error: TeamErrorDetail{
			Code:    code,
			Message: message,
		},
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		h.log.Error("failed to encode error response", sl.Err(err))
	}
}
