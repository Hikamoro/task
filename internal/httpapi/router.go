package httpapi

import (
	"log/slog"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"

	_ "task/docs"

	"task/internal/config"
	"task/internal/httpapi/middleware"
	"task/internal/service"
)

func NewRouter(app *service.App, cfg *config.Config, logger *slog.Logger) http.Handler {
	h := &handlers{app: app, logger: logger, maxBodyBytes: cfg.MaxBodyBytes}

	auth := func(next http.HandlerFunc) http.Handler {
		return middleware.RequireAuth(app.AuthManager(), next)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", h.health)

	mux.HandleFunc("POST /api/v1/register", h.Register)
	mux.HandleFunc("POST /api/v1/login", h.Login)

	mux.Handle("POST /api/v1/teams", auth(h.CreateTeam))
	mux.Handle("GET /api/v1/teams", auth(h.ListTeams))
	mux.Handle("GET /api/v1/teams/{team_id}/members", auth(h.ListMembers))
	mux.Handle("POST /api/v1/teams/{team_id}/invite", auth(h.InviteMember))
	mux.Handle("PATCH /api/v1/teams/{team_id}/members/{user_id}", auth(h.UpdateMemberRole))
	mux.Handle("DELETE /api/v1/teams/{team_id}/members/{user_id}", auth(h.RemoveMember))
	mux.Handle("GET /api/v1/teams/{team_id}/stats", auth(h.TeamStats))

	mux.Handle("POST /api/v1/tasks", auth(h.CreateTask))
	mux.Handle("GET /api/v1/tasks", auth(h.ListTasks))
	mux.Handle("PUT /api/v1/tasks/{id}", auth(h.UpdateTask))
	mux.Handle("GET /api/v1/tasks/{id}/history", auth(h.TaskHistory))
	mux.Handle("POST /api/v1/tasks/{id}/comments", auth(h.CreateComment))
	mux.Handle("GET /api/v1/tasks/{id}/comments", auth(h.ListComments))

	mux.Handle("GET /swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))

	var handler http.Handler = mux
	handler = middleware.Recover(logger, handler)
	handler = middleware.RequestLog(logger, handler)
	handler = middleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst, handler)
	return handler
}