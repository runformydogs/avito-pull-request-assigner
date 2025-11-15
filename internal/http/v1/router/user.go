package router

import (
	"github.com/go-chi/chi/v5"
	"log/slog"
	"pull-request-assigner/internal/http/v1/handler"
	"pull-request-assigner/internal/service"
)

type UserRouter struct {
	handler *handler.UserHandler
}

func NewUserRouter(userService *service.UserService, log *slog.Logger) *UserRouter {
	return &UserRouter{
		handler: handler.NewUserHandler(userService, log),
	}
}
func (ur *UserRouter) SetupRoutes(r chi.Router) {

	r.Route("/users", func(r chi.Router) {
		r.Post("/setIsActive", ur.handler.SetIsActive)

		r.Get("/getReview", ur.handler.GetReview)
	})

}
