package integration

import (
	"fmt"
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log/slog"
	"net/http/httptest"
	"os"
	"pull-request-assigner/internal/http/v1/router"
	"pull-request-assigner/internal/repo"
	"pull-request-assigner/internal/service"
)

type TestServer struct {
	DB     *sqlx.DB
	Server *httptest.Server
}

func NewTestServer() (*TestServer, error) {
	dbURL := "host=localhost port=5432 user=postgres password=postgres dbname=pullrequest_db sslmode=disable"

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	prRepo := repo.NewPullRequestRepo(db)
	teamRepo := repo.NewTeamRepo(db)
	userRepo := repo.NewUserRepo(db)

	prService := service.NewPullRequestService(log, prRepo, teamRepo)
	teamService := service.NewTeamService(log, teamRepo)
	userService := service.NewUserService(log, userRepo)

	r := chi.NewRouter()
	router.NewPullRequestRouter(prService, log).SetupRoutes(r)
	router.NewTeamRouter(teamService, log).SetupRoutes(r)
	router.NewUserRouter(userService, log).SetupRoutes(r)

	ts := httptest.NewServer(r)

	return &TestServer{
		DB:     db,
		Server: ts,
	}, nil
}

func (s *TestServer) LoadFixtures() error {
	tables := []string{"pr_reviewers", "pull_requests", "team_members", "users", "teams"}
	for _, table := range tables {
		_, err := s.DB.Exec(fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}

	fixtures := `
		INSERT INTO teams(team_name) VALUES 
			('Backend'),
			('QA');

		INSERT INTO users(user_id, username, team_name, is_active) VALUES
			(1, 'Alice', 'Backend', true),
			(2, 'Bob', 'Backend', true),
			(3, 'Carol', 'Backend', true),
			(4, 'David', 'Backend', true),
			(5, 'Eve', 'Backend', true),
			(10, 'Ivan', 'QA', true),
			(11, 'Max', 'QA', true);

		INSERT INTO team_members(team_name, user_id) VALUES
			('Backend', 1),
			('Backend', 2),
			('Backend', 3),
			('Backend', 4),
			('Backend', 5),
			('QA', 10),
			('QA', 11);
	`

	_, err := s.DB.Exec(fixtures)
	if err != nil {
		return fmt.Errorf("failed to load fixtures: %w", err)
	}

	log.Println("Fixtures loaded successfully")
	return nil
}

func (s *TestServer) Close() {
	s.Server.Close()
	s.DB.Close()
}
