package service_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"task/internal/auth"
	"task/internal/config"
	"task/internal/httpapi"
	"task/internal/service"
)

// ---- response shapes (mirror the API) ----

type apiUser struct {
	ID int64 `json:"id"`
}

type apiAuth struct {
	Token string  `json:"token"`
	User  apiUser `json:"user"`
}

type apiEnvelope struct {
	Team  *struct{ ID int64 `json:"id"` }  `json:"team"`
	Task  *struct {
		ID      int64  `json:"id"`
		Title   string `json:"title"`
		Status  string `json:"status"`
		Version int64  `json:"version"`
	} `json:"task"`
	Tasks []struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	} `json:"tasks"`
	Total int64 `json:"total"`
	Stats *struct {
		TodoCount       int64 `json:"todo_count"`
		InProgressCount int64 `json:"in_progress_count"`
		DoneCount       int64 `json:"done_count"`
	} `json:"stats"`
	History []json.RawMessage `json:"history"`
}

func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	app := service.New(tsRepo, tsCache, auth.NewManager("test-secret-for-jwt", time.Hour), testLog)
	cfg := &config.Config{
		HTTPAddr:       ":0",
		RateLimitRPS:   1000,
		RateLimitBurst: 1000,
		MaxBodyBytes:   1 << 20,
	}
	srv := httptest.NewServer(httpapi.NewRouter(app, cfg, testLog))
	t.Cleanup(srv.Close)
	return srv, ""
}

func apiCall(t *testing.T, srv *httptest.Server, method, path, token string, body any) (int, []byte) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, srv.URL+path, rd)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, data
}

func registerAndLogin(t *testing.T, srv *httptest.Server, email string) string {
	t.Helper()
	status, body := apiCall(t, srv, "POST", "/api/v1/register", email, map[string]any{
		"email":    email,
		"password": "password123",
		"name":     email,
	})
	if status != http.StatusCreated {
		t.Fatalf("register %s: status %d body %s", email, status, body)
	}
	status, body = apiCall(t, srv, "POST", "/api/v1/login", "", map[string]any{
		"email":    email,
		"password": "password123",
	})
	if status != http.StatusOK {
		t.Fatalf("login %s: status %d body %s", email, status, body)
	}
	var auth apiAuth
	if err := json.Unmarshal(body, &auth); err != nil {
		t.Fatalf("decode login: %v (%s)", err, body)
	}
	return auth.Token
}

func TestAPIEndToEnd(t *testing.T) {
	srv, _ := newTestServer(t)

	// auth
	ownerToken := registerAndLogin(t, srv, "e2e-owner@test.com")
	memberToken := registerAndLogin(t, srv, "e2e-member@test.com")

	// unauthenticated access is rejected
	if status, _ := apiCall(t, srv, "GET", "/api/v1/teams", "", nil); status != http.StatusUnauthorized {
		t.Errorf("GET /teams without token: status %d, want 401", status)
	}

	// create team
	status, body := apiCall(t, srv, "POST", "/api/v1/teams", ownerToken, map[string]any{"name": "E2E Team"})
	if status != http.StatusCreated {
		t.Fatalf("create team: status %d body %s", status, body)
	}
	var envelope apiEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode team: %v", err)
	}
	teamID := envelope.Team.ID

	// invite member
	status, body = apiCall(t, srv, "POST", "/api/v1/teams/"+itoa(teamID)+"/invite", ownerToken,
		map[string]any{"email": "e2e-member@test.com", "role": "member"})
	if status != http.StatusOK {
		t.Fatalf("invite: status %d body %s", status, body)
	}

	// create two tasks
	task1 := createTaskViaAPI(t, srv, ownerToken, teamID, "first task")
	createTaskViaAPI(t, srv, ownerToken, teamID, "second task")

	// list -> both tasks, proves the cache was invalidated on the second create
	status, body = apiCall(t, srv, "GET", "/api/v1/tasks?team_id="+itoa(teamID), ownerToken, nil)
	if status != http.StatusOK {
		t.Fatalf("list tasks: status %d body %s", status, body)
	}
	var list apiEnvelope
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if list.Total != 2 || len(list.Tasks) != 2 {
		t.Errorf("list: total=%d len=%d, want 2/2", list.Total, len(list.Tasks))
	}

	// member sees the team's tasks too
	if status, body = apiCall(t, srv, "GET", "/api/v1/tasks?team_id="+itoa(teamID), memberToken, nil); status != http.StatusOK {
		t.Errorf("member list tasks: status %d body %s", status, body)
	}

	// update with fresh version
	status, body = apiCall(t, srv, "PUT", "/api/v1/tasks/"+itoa(task1), ownerToken, map[string]any{
		"title":   "first task renamed",
		"status":  "done",
		"version": 1,
	})
	if status != http.StatusOK {
		t.Fatalf("update task: status %d body %s", status, body)
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if envelope.Task.Status != "done" {
		t.Errorf("task status = %s, want done", envelope.Task.Status)
	}

	// stale version -> 409
	status, _ = apiCall(t, srv, "PUT", "/api/v1/tasks/"+itoa(task1), ownerToken, map[string]any{
		"title":   "conflicting write",
		"version": 1,
	})
	if status != http.StatusConflict {
		t.Errorf("stale update: status %d, want 409", status)
	}

	// history recorded
	status, body = apiCall(t, srv, "GET", "/api/v1/tasks/"+itoa(task1)+"/history", ownerToken, nil)
	if status != http.StatusOK {
		t.Fatalf("history: status %d body %s", status, body)
	}
	var hist apiEnvelope
	if err := json.Unmarshal(body, &hist); err != nil {
		t.Fatalf("decode history: %v", err)
	}
	if len(hist.History) != 1 {
		t.Errorf("history len = %d, want 1", len(hist.History))
	}

	// comments
	status, body = apiCall(t, srv, "POST", "/api/v1/tasks/"+itoa(task1)+"/comments", memberToken,
		map[string]any{"content": "looks good"})
	if status != http.StatusCreated {
		t.Fatalf("comment: status %d body %s", status, body)
	}

	// stats: owner allowed, member forbidden
	status, body = apiCall(t, srv, "GET", "/api/v1/teams/"+itoa(teamID)+"/stats", ownerToken, nil)
	if status != http.StatusOK {
		t.Fatalf("stats owner: status %d body %s", status, body)
	}
	var st apiEnvelope
	if err := json.Unmarshal(body, &st); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if st.Stats.DoneCount != 1 {
		t.Errorf("stats done_count = %d, want 1", st.Stats.DoneCount)
	}
	if status, _ := apiCall(t, srv, "GET", "/api/v1/teams/"+itoa(teamID)+"/stats", memberToken, nil); status != http.StatusForbidden {
		t.Errorf("stats member: status %d, want 403", status)
	}

	// swagger is served
	status, body = apiCall(t, srv, "GET", "/swagger/index.html", "", nil)
	if status != http.StatusOK || !strings.Contains(string(body), "swagger") {
		t.Errorf("swagger: status %d, want 200 with swagger content", status)
	}
}

func createTaskViaAPI(t *testing.T, srv *httptest.Server, token string, teamID int64, title string) int64 {
	t.Helper()
	status, body := apiCall(t, srv, "POST", "/api/v1/tasks?team_id="+itoa(teamID), token,
		map[string]any{"title": title, "description": "desc"})
	if status != http.StatusCreated {
		t.Fatalf("create task %s: status %d body %s", title, status, body)
	}
	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode task: %v", err)
	}
	return env.Task.ID
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
