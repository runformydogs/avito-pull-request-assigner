package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"time"
)

type Config struct {
	Env      string         `env:"ENV" env-default:"dev"`
	Server   HTTPServer     `env-prefix:"SERVER_"`
	Postgres PostgresConfig `env-prefix:"PG_"`
}

type HTTPServer struct {
	Port    string        `env:"PORT" env-default:"8080"`
	Timeout time.Duration `env:"TIMEOUT" env-default:"5s"`
}

type PostgresConfig struct {
	Host     string `env:"HOST" env-default:"localhost"`
	Port     string `env:"PORT" env-default:"5432"`
	User     string `env:"USER" env-default:"postgres"`
	Password string `env:"PASSWORD" env-default:"postgres"`
	DbName   string `env:"DBNAME" env-default:"pullrequest_db"`
	SslMode  string `env:"SSLMODE" env-default:"disable"`
}

func MustLoad() *Config {
	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		panic("failed to read config from environment: " + err.Error())
	}

	return &cfg
}
