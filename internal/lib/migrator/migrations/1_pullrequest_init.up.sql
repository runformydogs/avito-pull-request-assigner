CREATE TABLE IF NOT EXISTS teams
(
    team_name VARCHAR(255) PRIMARY KEY
    );

CREATE TABLE IF NOT EXISTS users
(
    user_id   INTEGER PRIMARY KEY,
    username  VARCHAR(255) NOT NULL,
    team_name VARCHAR(255) NOT NULL,
    is_active BOOLEAN      NOT NULL DEFAULT true,
    FOREIGN KEY (team_name) REFERENCES teams (team_name) ON DELETE RESTRICT
    );

CREATE INDEX idx_users_team_active ON users(team_name, is_active) WHERE is_active = true;

CREATE TABLE IF NOT EXISTS team_members
(
    team_name VARCHAR(255) NOT NULL,
    user_id   INTEGER NOT NULL,
    PRIMARY KEY (team_name, user_id),
    FOREIGN KEY (team_name) REFERENCES teams (team_name) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (user_id) ON DELETE CASCADE
    );

CREATE TABLE IF NOT EXISTS pull_requests
(
    pull_request_id   VARCHAR(255) PRIMARY KEY,
    pull_request_name VARCHAR(255) NOT NULL,
    author_id         INTEGER NOT NULL,
    status            VARCHAR(50)  NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'MERGED')),
    created_at        TIMESTAMP DEFAULT NOW(),
    merged_at         TIMESTAMP NULL,
    FOREIGN KEY (author_id) REFERENCES users(user_id) ON DELETE RESTRICT
    );

CREATE INDEX idx_pull_requests_author_id ON pull_requests(author_id);
CREATE INDEX idx_pull_requests_status ON pull_requests(status);

CREATE TABLE IF NOT EXISTS pr_reviewers
(
    pull_request_id VARCHAR(255) NOT NULL,
    reviewer_id     INTEGER NOT NULL,
    PRIMARY KEY (pull_request_id, reviewer_id),
    FOREIGN KEY (pull_request_id) REFERENCES pull_requests (pull_request_id) ON DELETE CASCADE,
    FOREIGN KEY (reviewer_id) REFERENCES users (user_id) ON DELETE CASCADE
    );

CREATE INDEX idx_pr_reviewers_reviewer_id ON pr_reviewers(reviewer_id);