package router

import (
	"github.com/go-chi/chi/v5"
	"log/slog"
	"pull-request-assigner/internal/http/v1/handler"
	"pull-request-assigner/internal/service"
)

type StatsRouter struct {
	handler *handler.StatsHandler
}

func NewStatsRouter(statsService *service.StatsService, log *slog.Logger) *StatsRouter {
	return &StatsRouter{
		handler: handler.NewStatsHandler(statsService, log),
	}
}

func (sr *StatsRouter) SetupRoutes(r chi.Router) {

	r.Route("/stats", func(r chi.Router) {
		r.Get("/prs", sr.handler.GetPRStats)
	})
}
