package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestTeamCreate(t *testing.T) {
	ts, err := NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	defer ts.Close()

	if err := ts.LoadFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	body := `{
    "team_name": "NewTeam",
    "members": [
        {"user_id": "u10", "username": "Ivan", "is_active": true},
        {"user_id": "u11", "username": "Max", "is_active": true}
    ]
}`

	resp := doPost(t, ts, "/team/add", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		TeamName string `json:"team_name"`
		Members  []struct {
			UserID string `json:"user_id"`
		} `json:"members"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if data.TeamName != "NewTeam" {
		t.Fatalf("wrong team: %s", data.TeamName)
	}

	if len(data.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(data.Members))
	}
}

func TestTeamGet(t *testing.T) {
	ts, err := NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	defer ts.Close()

	if err := ts.LoadFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	resp := doGet(t, ts, "/team/get?team_name=Backend")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		TeamName string `json:"team_name"`
		Members  []struct {
			UserID string `json:"user_id"`
		} `json:"members"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if data.TeamName != "Backend" {
		t.Fatalf("wrong team: %s", data.TeamName)
	}

	if len(data.Members) != 5 {
		t.Fatalf("expected 5 members, got %d", len(data.Members))
	}
}

func TestTeamDeactivate(t *testing.T) {
	ts, err := NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	defer ts.Close()

	if err := ts.LoadFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	resp := doPost(t, ts, "/team/deactivate?team_name=Backend", `{}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		TeamName         string `json:"team_name"`
		DeactivatedUsers int    `json:"deactivated_users"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if data.DeactivatedUsers != 5 {
		t.Fatalf("expected all 5 users to deactivate, got %d", data.DeactivatedUsers)
	}
}

func TestPullRequestCreate(t *testing.T) {
	ts, err := NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	defer ts.Close()

	if err := ts.LoadFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	body := `{
		"pull_request_id": "PR-1",
		"pull_request_name": "Fix API",
		"author_id": "u1"
	}`

	resp := doPost(t, ts, "/pullRequest/create", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		PR struct {
			PullRequestID     string   `json:"pull_request_id"`
			Status            string   `json:"status"`
			AssignedReviewers []string `json:"assigned_reviewers"`
		} `json:"pr"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if data.PR.PullRequestID != "PR-1" {
		t.Fatalf("wrong PR id: %s", data.PR.PullRequestID)
	}

	if data.PR.Status != "OPEN" {
		t.Fatalf("expected OPEN status, got %s", data.PR.Status)
	}

	if len(data.PR.AssignedReviewers) != 2 {
		t.Fatalf("expected 2 reviewers, got %d", len(data.PR.AssignedReviewers))
	}
}

func TestPullRequestMerge(t *testing.T) {
	ts, err := NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	defer ts.Close()

	if err := ts.LoadFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	resp := doPost(t, ts, "/pullRequest/create", `{
		"pull_request_id": "PR-77",
		"pull_request_name": "Refactor",
		"author_id": "u1"
	}`)
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("failed to create PR: %d: %s", resp.StatusCode, string(body))
	}

	resp = doPost(t, ts, "/pullRequest/merge", `{
		"pull_request_id": "PR-77"
	}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		PR struct {
			Status string `json:"status"`
		} `json:"pr"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if data.PR.Status != "MERGED" {
		t.Fatalf("expected MERGED status, got %s", data.PR.Status)
	}
}

func TestPullRequestReassign(t *testing.T) {
	ts, err := NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	defer ts.Close()

	if err := ts.LoadFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	resp := doPost(t, ts, "/pullRequest/create", `{
		"pull_request_id": "PR-200",
		"pull_request_name": "Refactor core",
		"author_id": "u1"
	}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("failed to create PR: %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		PR struct {
			Reviewers []string `json:"assigned_reviewers"`
		} `json:"pr"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode PR response: %v", err)
	}

	if len(data.PR.Reviewers) == 0 {
		t.Fatal("no reviewers assigned")
	}

	old := data.PR.Reviewers[0]

	body := fmt.Sprintf(`{
		"pull_request_id": "PR-200",
		"old_reviewer_id": "%s"
	}`, old)

	resp2 := doPost(t, ts, "/pullRequest/reassign", body)
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("expected 200, got %d: %s", resp2.StatusCode, string(body))
	}

	var out struct {
		ReplacedBy string `json:"replaced_by"`
	}

	if err := json.NewDecoder(resp2.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode reassign response: %v", err)
	}

	if out.ReplacedBy == old {
		t.Fatalf("reviewer was NOT replaced: %s", out.ReplacedBy)
	}
}

func TestUserSetIsActive(t *testing.T) {
	ts, err := NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	defer ts.Close()

	if err := ts.LoadFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	resp := doPost(t, ts, "/users/setIsActive", `{"user_id":"u2","is_active":false}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		User struct {
			UserID   string `json:"user_id"`
			IsActive bool   `json:"is_active"`
		} `json:"user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if data.User.IsActive {
		t.Fatalf("user should be inactive")
	}
}

func TestUserGetReview(t *testing.T) {
	ts, err := NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	defer ts.Close()

	if err := ts.LoadFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	resp := doPost(t, ts, "/pullRequest/create", `{
		"pull_request_id": "PR-999",
		"pull_request_name": "Fix",
		"author_id": "u1"
	}`)
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("failed to create PR: %d: %s", resp.StatusCode, string(body))
	}

	resp = doGet(t, ts, "/users/getReview?user_id=u2")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		PullRequests []struct {
			PullRequestID string `json:"pull_request_id"`
		} `json:"pull_requests"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(data.PullRequests) == 0 {
		t.Fatalf("expected at least 1 PR in review list")
	}
}

func doPost(t *testing.T, ts *TestServer, path string, body string) *http.Response {
	resp, err := http.Post(ts.Server.URL+path, "application/json", bytes.NewBuffer([]byte(body)))
	if err != nil {
		t.Fatalf("POST %s failed: %v", path, err)
	}
	return resp
}

func doGet(t *testing.T, ts *TestServer, path string) *http.Response {
	resp, err := http.Get(ts.Server.URL + path)
	if err != nil {
		t.Fatalf("GET %s failed: %v", path, err)
	}
	return resp
}
