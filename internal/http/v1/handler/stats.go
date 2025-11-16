package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"pull-request-assigner/internal/lib/logger/sl"
	"pull-request-assigner/internal/service"
)

type (
	PRStatsResponse struct {
		Stats PRStatsData `json:"stats"`
	}

	PRStatsData struct {
		TotalPRs          int     `json:"total_prs"`
		OpenPRs           int     `json:"open_prs"`
		MergedPRs         int     `json:"merged_prs"`
		AvgReviewersPerPR float64 `json:"avg_reviewers_per_pr"`
	}

	StatsErrorResponse struct {
		Error StatsErrorDetail `json:"error"`
	}

	StatsErrorDetail struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
)

type StatsHandler struct {
	statsService *service.StatsService
	log          *slog.Logger
}

func NewStatsHandler(statsService *service.StatsService, log *slog.Logger) *StatsHandler {
	return &StatsHandler{
		statsService: statsService,
		log:          log,
	}
}

func (h *StatsHandler) GetPRStats(w http.ResponseWriter, r *http.Request) {
	const op = "handler.stats.GetPRStats"

	log := h.log.With(slog.String("op", op))

	log.Info("handling PR stats request")

	stats, err := h.statsService.GetPRStats(r.Context())
	if err != nil {
		log.Error("failed to get PR stats", sl.Err(err))
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get PR statistics")
		return
	}

	response := PRStatsResponse{
		Stats: PRStatsData{
			TotalPRs:          stats.TotalPRs,
			OpenPRs:           stats.OpenPRs,
			MergedPRs:         stats.MergedPRs,
			AvgReviewersPerPR: stats.AvgReviewersPerPR,
		},
	}

	h.writeJSON(w, http.StatusOK, response)
	log.Info("PR stats returned successfully",
		slog.Int("total_prs", stats.TotalPRs),
		slog.Int("open_prs", stats.OpenPRs))
}

func (h *StatsHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Error("failed to encode JSON response", sl.Err(err))
	}
}

func (h *StatsHandler) writeErrorResponse(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResp := StatsErrorResponse{
		Error: StatsErrorDetail{
			Code:    code,
			Message: message,
		},
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		h.log.Error("failed to encode error response", sl.Err(err))
	}
}
