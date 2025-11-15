package migrator

import (
	"embed"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"log/slog"
	"pull-request-assigner/internal/config"
)

//go:embed migrations/*.sql
var fs embed.FS

// RunMigrations up migrations files from embed.FS - fs
func RunMigrations(cfg config.PostgresConfig, log *slog.Logger) error {
	const op = "migrator.RunMigrations"

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DbName, cfg.SslMode)

	migrationDB, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return fmt.Errorf("%s: failed to connect: %w", op, err)
	}
	defer migrationDB.Close()

	driver, err := postgres.WithInstance(migrationDB.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("%s: failed to create driver: %w", op, err)
	}

	source, err := iofs.New(fs, "migrations")
	if err != nil {
		return fmt.Errorf("%s: failed to create source: %w", op, err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("%s: failed to create migrate instance: %w", op, err)
	}
	defer m.Close()

	log.Info("applying database migrations")
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("%s: migration failed: %w", op, err)
	}

	return nil
}
