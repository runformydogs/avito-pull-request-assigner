package v1

import (
	"github.com/go-chi/chi/v5"
	"log/slog"
	router2 "pull-request-assigner/internal/http/v1/router"
	"pull-request-assigner/internal/service"
)

type Router interface {
	SetupRoutes(r chi.Router)
}

type RouterDependencies struct {
	TeamService        *service.TeamService
	UserService        *service.UserService
	PullRequestService *service.PullRequestService
	Log                *slog.Logger
}

func SetupRoutes(r chi.Router, deps *RouterDependencies) {
	routers := []Router{
		router2.NewTeamRouter(deps.TeamService, deps.Log),
		router2.NewUserRouter(deps.UserService, deps.Log),
		router2.NewPullRequestRouter(deps.PullRequestService, deps.Log),
	}

	for _, serviceRouter := range routers {
		serviceRouter.SetupRoutes(r)
	}
}
