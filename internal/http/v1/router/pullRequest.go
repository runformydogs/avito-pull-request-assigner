package router

import (
	"github.com/go-chi/chi/v5"
	"log/slog"
	"pull-request-assigner/internal/http/v1/handler"
	"pull-request-assigner/internal/service"
)

type PullRequestRouter struct {
	handler *handler.PullRequestHandler
}

func NewPullRequestRouter(pullRequestService *service.PullRequestService, log *slog.Logger) *PullRequestRouter {
	return &PullRequestRouter{
		handler: handler.NewPullRequestHandler(pullRequestService, log),
	}
}
func (prr *PullRequestRouter) SetupRoutes(r chi.Router) {

	r.Route("/pullRequest", func(r chi.Router) {
		r.Post("/create", prr.handler.CreatePullRequest)
		r.Post("/merge", prr.handler.MergePullRequest)
		r.Post("/reassign", prr.handler.ReassignReviewer)
	})

}
