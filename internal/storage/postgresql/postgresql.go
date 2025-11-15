package postgresql

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"log"
	"pull-request-assigner/internal/config"
	"runtime/debug"
)

type Storage struct {
	db *sqlx.DB
}

func Init(cfg config.PostgresConfig) *Storage {
	const op = "storage.postgresql.Init"

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DbName, cfg.SslMode)

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		panic(fmt.Sprintf("%s: failed to open db: %v", op, err))
	}

	if err = db.Ping(); err != nil {
		panic(fmt.Sprintf("%s: failed to ping db: %v", op, err))
	}

	return &Storage{db: db}
}

func (s *Storage) GetDB() *sqlx.DB {
	return s.db
}

func (s *Storage) Close() {
	if s.db != nil {
		log.Printf("Closing DB (caller):\n%s", debug.Stack())
		log.Printf("DB stats before close: InUse=some Idle=some")
		s.db.Close()
	}
}
