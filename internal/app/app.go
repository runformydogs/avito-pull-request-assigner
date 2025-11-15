package app

import (
	"log/slog"
	"pull-request-assigner/internal/app/rest"
)

type App struct {
	log     *slog.Logger
	storage *psql.Storage
	restApp *rest.App
}
