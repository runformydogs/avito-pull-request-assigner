package models

import (
	"database/sql"
	"time"
)

type PullRequest struct {
	PullRequestId   string       `db:"pull_request_id" json:"pull_request_id"`
	PullRequestName string       `db:"pull_request_name" json:"pull_request_name"`
	AuthorID        string       `db:"author_id" json:"author_id"`
	Status          string       `db:"status" json:"status"`
	CreatedAt       time.Time    `db:"created_at" json:"created_at"`
	MergedAt        sql.NullTime `db:"merged_at" json:"merged_at,omitempty"`
}

type PullRequestShort struct {
	PullRequestId   string `db:"pull_request_id" json:"pull_request_id"`
	PullRequestName string `db:"pull_request_name" json:"pull_request_name"`
	AuthorID        string `db:"author_id" json:"author_id"`
	Status          string `db:"status" json:"status"`
}
