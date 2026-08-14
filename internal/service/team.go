package service

import (
	"context"
	"strings"

	"task/internal/model"
)

func (a *App) CreateTeam(ctx context.Context, userID int64, name string) (model.Team, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return model.Team{}, model.ErrInvalidInput
	}
	return a.repo.CreateTeam(ctx, name, userID)
}

func (a *App) ListTeams(ctx context.Context, userID int64) ([]model.Team, error) {
	return a.repo.ListUserTeams(ctx, userID)
}

func (a *App) ListMembers(ctx context.Context, actorID, teamID int64) ([]model.TeamMember, error) {
	if _, err := a.requireTeamMember(ctx, teamID, actorID); err != nil {
		return nil, err
	}
	return a.repo.ListMembers(ctx, teamID)
}

// Invite adds an existing user to the team. Only the team owner or admin may invite.
// The owner role can never be granted through an invitation.
func (a *App) Invite(ctx context.Context, actorID, teamID int64, email string, role model.Role) error {
	if !role.Valid() || role == model.RoleOwner {
		return model.ErrInvalidInput
	}
	if err := a.requireTeamOwnerOrAdmin(ctx, teamID, actorID); err != nil {
		return err
	}
	user, err := a.repo.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return err
	}
	if err := a.repo.AddMember(ctx, teamID, user.ID, role); err != nil {
		return err
	}
	return nil
}

// UpdateMemberRole changes a member's role. Only owner/admin may do this, and
// the owner's role is immutable.
func (a *App) UpdateMemberRole(ctx context.Context, actorID, teamID, targetUserID int64, role model.Role) error {
	if !role.Valid() || role == model.RoleOwner {
		return model.ErrInvalidInput
	}
	if err := a.requireTeamOwnerOrAdmin(ctx, teamID, actorID); err != nil {
		return err
	}
	targetRole, err := a.repo.GetMemberRole(ctx, teamID, targetUserID)
	if err != nil {
		return err
	}
	if targetRole == model.RoleOwner {
		return model.ErrForbidden
	}
	return a.repo.UpdateMemberRole(ctx, teamID, targetUserID, role)
}

// RemoveMember removes a user from the team. The owner cannot be removed.
func (a *App) RemoveMember(ctx context.Context, actorID, teamID, targetUserID int64) error {
	if err := a.requireTeamOwnerOrAdmin(ctx, teamID, actorID); err != nil {
		return err
	}
	targetRole, err := a.repo.GetMemberRole(ctx, teamID, targetUserID)
	if err != nil {
		return err
	}
	if targetRole == model.RoleOwner {
		return model.ErrForbidden
	}
	if err := a.repo.RemoveMember(ctx, teamID, targetUserID); err != nil {
		return err
	}
	a.invalidateTeamCache(ctx, teamID)
	return nil
}
