package rest

import (
	"context"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
	v1 "pull-request-assigner/internal/http/v1"
)

type App struct {
	log        *slog.Logger
	deps       *v1.RouterDependencies
	httpServer *http.Server
}

func New(
	log *slog.Logger,
	deps *v1.RouterDependencies,
	port string,
) *App {
	r := chi.NewRouter()

	v1.SetupRoutes(r, deps)

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	return &App{
		log:        log,
		deps:       deps,
		httpServer: httpServer,
	}
}

func (a *App) Run() error {
	const op = "app.rest.Run"
	a.log.With(slog.String("op", op)).Info("starting REST server", "port", a.httpServer.Addr)
	return a.httpServer.ListenAndServe()
}

func (a *App) Stop(ctx context.Context) error {
	const op = "app.rest.Stop"
	a.log.With(slog.String("op", op)).Info("stopping REST server")
	return a.httpServer.Shutdown(ctx)
}
