package service

import (
	"context"
	"log/slog"

	"task/internal/auth"
	"task/internal/cache"
	"task/internal/model"
	"task/internal/repository"
)

type App struct {
	repo   *repository.Repository
	cache  *cache.TaskCache
	auth   *auth.Manager
	logger *slog.Logger
}

func New(repo *repository.Repository, c *cache.TaskCache, m *auth.Manager, logger *slog.Logger) *App {
	return &App{repo: repo, cache: c, auth: m, logger: logger}
}

func (a *App) AuthManager() *auth.Manager {
	return a.auth
}

// teamRole returns the actor's role in a team or model.ErrNotFound if not a member.
func (a *App) teamRole(ctx context.Context, teamID, userID int64) (model.Role, error) {
	if _, err := a.repo.GetTeam(ctx, teamID); err != nil {
		return "", err
	}
	return a.repo.GetMemberRole(ctx, teamID, userID)
}

// requireTeamMember ensures the actor belongs to the team.
func (a *App) requireTeamMember(ctx context.Context, teamID, userID int64) (model.Role, error) {
	role, err := a.repo.GetMemberRole(ctx, teamID, userID)
	if err != nil {
		if err == model.ErrNotFound {
			return "", model.ErrForbidden
		}
		return "", err
	}
	return role, nil
}

// requireTeamOwnerOrAdmin ensures the actor manages the team.
func (a *App) requireTeamOwnerOrAdmin(ctx context.Context, teamID, userID int64) error {
	role, err := a.requireTeamMember(ctx, teamID, userID)
	if err != nil {
		return err
	}
	if role != model.RoleOwner && role != model.RoleAdmin {
		return model.ErrForbidden
	}
	return nil
}

func (a *App) invalidateTeamCache(ctx context.Context, teamID int64) {
	if a.cache == nil {
		return
	}
	if err := a.cache.Bump(ctx, teamID); err != nil {
		a.logger.Warn("failed to invalidate team cache", "team_id", teamID, "error", err)
	}
}
