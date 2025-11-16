package v1

import (
	"github.com/go-chi/chi/v5"
	"log/slog"
	"pull-request-assigner/internal/http/v1/router"
	"pull-request-assigner/internal/service"
)

type Router interface {
	SetupRoutes(r chi.Router)
}

type RouterDependencies struct {
	TeamService        *service.TeamService
	UserService        *service.UserService
	PullRequestService *service.PullRequestService
	StatsService       *service.StatsService
}

func SetupRoutes(r chi.Router, deps *RouterDependencies, log *slog.Logger) {
	routers := []Router{
		router.NewTeamRouter(deps.TeamService, log),
		router.NewUserRouter(deps.UserService, log),
		router.NewPullRequestRouter(deps.PullRequestService, log),
		router.NewStatsRouter(deps.StatsService, log),
	}

	for _, serviceRouter := range routers {
		serviceRouter.SetupRoutes(r)
	}
}
