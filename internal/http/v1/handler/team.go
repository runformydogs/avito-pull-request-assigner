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
	CreateTeamRequest struct {
		TeamName string        `json:"team_name"`
		Members  []models.User `json:"members"`
	}

	CreateTeamResponse struct {
		Team models.Team `json:"team"`
	}

	GetTeamResponse struct {
		Team models.Team `json:"team"`
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
		h.writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.TeamName == "" {
		log.Error("team_name is required")
		h.writeError(w, http.StatusBadRequest, "team_name is required", nil)
		return
	}

	if len(req.Members) == 0 {
		log.Error("team must have at least one member")
		h.writeError(w, http.StatusBadRequest, "team must have at least one member", nil)
		return
	}

	for i, member := range req.Members {
		if member.UserID == "" {
			log.Error("user_id is required for all members", slog.Int("member_index", i))
			h.writeError(w, http.StatusBadRequest,
				fmt.Sprintf("user_id is required for member at index %d", i), nil)
			return
		}
		if member.Username == "" {
			log.Error("username is required for all members", slog.Int("member_index", i))
			h.writeError(w, http.StatusBadRequest,
				fmt.Sprintf("username is required for member at index %d", i), nil)
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

		if err.Error() == fmt.Sprintf("%s: team %s already exists", op, req.TeamName) {
			h.writeError(w, http.StatusBadRequest, "team already exists", err)
		} else {
			h.writeError(w, http.StatusInternalServerError, "failed to create team", err)
		}
		return
	}

	response := CreateTeamResponse{
		Team: *createdTeam,
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
		h.writeError(w, http.StatusBadRequest, "team_name query parameter is required", nil)
		return
	}

	team, err := h.teamService.GetTeamWithMembers(r.Context(), teamName)
	if err != nil {
		log.Error("failed to get team", sl.Err(err))

		if err.Error() == fmt.Sprintf("%s: team not found", op) {
			h.writeError(w, http.StatusNotFound, "team not found", err)
		} else {
			h.writeError(w, http.StatusInternalServerError, "failed to get team", err)
		}
		return
	}

	response := GetTeamResponse{
		Team: *team,
	}

	h.writeJSON(w, http.StatusOK, response)
	log.Info("team retrieved successfully")
}

func (h *TeamHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("Error encoding JSON response: %v\n", err)
	}
}

func (h *TeamHandler) writeError(w http.ResponseWriter, status int, message string, err error) {
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
