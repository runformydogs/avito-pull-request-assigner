package router

import (
	"github.com/go-chi/chi/v5"
	"log/slog"
	"pull-request-assigner/internal/http/v1/handler"
	"pull-request-assigner/internal/service"
)

type TeamRouter struct {
	handler *handler.TeamHandler
}

func NewTeamRouter(teamService *service.TeamService, log *slog.Logger) *TeamRouter {
	return &TeamRouter{
		handler: handler.NewTeamHandler(teamService, log),
	}
}
func (tr *TeamRouter) SetupRoutes(r chi.Router) {

	r.Route("/team", func(r chi.Router) {
		r.Post("/add", tr.handler.CreateTeam)

		r.Get("/get", tr.handler.GetTeam)
	})

}
