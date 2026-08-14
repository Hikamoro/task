package service

import (
	"context"

	"task/internal/model"
)

// GetStats returns the team report. Only the team owner or admin has access.
func (a *App) GetStats(ctx context.Context, userID, teamID int64) (model.TeamStats, error) {
	if err := a.requireTeamOwnerOrAdmin(ctx, teamID, userID); err != nil {
		return model.TeamStats{}, err
	}
	return a.repo.TeamStats(ctx, teamID)
}
