package app

import (
	"context"
	"log/slog"
	"pull-request-assigner/internal/app/rest"
	"pull-request-assigner/internal/config"
	v1 "pull-request-assigner/internal/http/v1"
	"pull-request-assigner/internal/lib/migrator"
	"pull-request-assigner/internal/repo"
	"pull-request-assigner/internal/service"
	"pull-request-assigner/internal/storage/postgresql"
	"time"
)

type App struct {
	log     *slog.Logger
	storage *postgresql.Storage
	restApp *rest.App
}

func MustNew(log *slog.Logger) *App {
	cfg := config.MustLoad()

	if err := migrator.RunMigrations(cfg.Postgres, log); err != nil {
		log.Error("failed to run migrations", "error", err)
		panic(err)
	}

	storage := postgresql.Init(cfg.Postgres)

	userRepo := repo.NewUserRepo(storage.GetDB())
	teamRepo := repo.NewTeamRepo(storage.GetDB())
	pullRequestRepo := repo.NewPullRequestRepo(storage.GetDB())

	userService := service.NewUserService(log, userRepo)
	teamService := service.NewTeamService(log, teamRepo)
	pullRequestService := service.NewPullRequestService(log, pullRequestRepo, teamRepo)

	routerDependencies := v1.RouterDependencies{
		UserService:        userService,
		TeamService:        teamService,
		PullRequestService: pullRequestService,
	}

	restApp := rest.New(
		log,
		&routerDependencies,
		cfg.Server.Port,
	)

	return &App{
		log:     log,
		storage: storage,
		restApp: restApp,
	}
}

func (a *App) MustRun() {
	const op = "app.MustRun"
	a.log.With(slog.String("op", op)).Info("starting application")

	if err := a.restApp.Run(); err != nil {
		panic(err)
	}
}

func (a *App) GracefulShutdown() {
	const op = "app.GracefulShutdown"
	a.log.With(slog.String("op", op)).Info("shutting down application")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.restApp.Stop(ctx); err != nil {
		a.log.Error("failed to stop HTTP server", err)
	}

	if a.storage != nil {
		a.storage.Close()
		a.log.Info("database connection closed")
	}
}
