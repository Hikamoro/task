package httpapi

import (
	"net/http"

	"task/internal/model"
)

type createTeamRequest struct {
	Name string `json:"name"`
}

// CreateTeam creates a team; the current user becomes its owner.
// @Summary Create a team
// @Tags teams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body createTeamRequest true "Team payload"
// @Success 201 {object} teamResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Router /teams [post]
func (h *handlers) CreateTeam(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	var req createTeamRequest
	if err := decodeBody(w, r, &req, h.maxBodyBytes); err != nil {
		writeError(w, h.logger, err)
		return
	}
	team, err := h.app.CreateTeam(r.Context(), userID, req.Name)
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, teamResponse{Team: team})
}

// ListTeams lists teams the current user belongs to.
// @Summary List my teams
// @Tags teams
// @Produce json
// @Security BearerAuth
// @Success 200 {object} teamsResponse
// @Failure 401 {object} errorResponse
// @Router /teams [get]
func (h *handlers) ListTeams(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	teams, err := h.app.ListTeams(r.Context(), userID)
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, teamsResponse{Teams: teams})
}

// ListMembers lists members of a team (any team member may view).
// @Summary List team members
// @Tags teams
// @Produce json
// @Security BearerAuth
// @Param team_id path int true "Team ID"
// @Success 200 {object} membersResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Router /teams/{team_id}/members [get]
func (h *handlers) ListMembers(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	teamID, err := parsePositiveInt64(r.PathValue("team_id"))
	if err != nil {
		writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "invalid team_id"))
		return
	}
	members, err := h.app.ListMembers(r.Context(), userID, teamID)
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, membersResponse{Members: members})
}

type inviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// InviteMember adds an existing user to a team. Only owner/admin may invite.
// The owner role is never granted through an invitation.
// @Summary Invite a user to a team
// @Tags teams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param team_id path int true "Team ID"
// @Param body body inviteRequest true "Invite payload"
// @Success 200 {object} statusResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Router /teams/{team_id}/invite [post]
func (h *handlers) InviteMember(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	teamID, err := parsePositiveInt64(r.PathValue("team_id"))
	if err != nil {
		writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "invalid team_id"))
		return
	}
	var req inviteRequest
	if err := decodeBody(w, r, &req, h.maxBodyBytes); err != nil {
		writeError(w, h.logger, err)
		return
	}
	role, ok := model.ParseRole(req.Role)
	if !ok {
		role = model.RoleMember
	}
	if err := h.app.Invite(r.Context(), userID, teamID, req.Email, role); err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, statusResponse{Status: "ok"})
}

type updateMemberRoleRequest struct {
	Role string `json:"role"`
}

// UpdateMemberRole changes a member's role. The owner role is immutable.
// @Summary Change a member's role
// @Tags teams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param team_id path int true "Team ID"
// @Param user_id path int true "Target user ID"
// @Param body body updateMemberRoleRequest true "New role (admin | member)"
// @Success 200 {object} statusResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /teams/{team_id}/members/{user_id} [patch]
func (h *handlers) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	teamID, err := parsePositiveInt64(r.PathValue("team_id"))
	if err != nil {
		writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "invalid team_id"))
		return
	}
	targetUserID, err := parsePositiveInt64(r.PathValue("user_id"))
	if err != nil {
		writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "invalid user_id"))
		return
	}
	var req updateMemberRoleRequest
	if err := decodeBody(w, r, &req, h.maxBodyBytes); err != nil {
		writeError(w, h.logger, err)
		return
	}
	role, ok := model.ParseRole(req.Role)
	if !ok {
		writeError(w, h.logger, model.ErrInvalidInput)
		return
	}
	if err := h.app.UpdateMemberRole(r.Context(), userID, teamID, targetUserID, role); err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, statusResponse{Status: "ok"})
}

// RemoveMember removes a user from a team. The owner cannot be removed.
// @Summary Remove a team member
// @Tags teams
// @Produce json
// @Security BearerAuth
// @Param team_id path int true "Team ID"
// @Param user_id path int true "Target user ID"
// @Success 200 {object} statusResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /teams/{team_id}/members/{user_id} [delete]
func (h *handlers) RemoveMember(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	teamID, err := parsePositiveInt64(r.PathValue("team_id"))
	if err != nil {
		writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "invalid team_id"))
		return
	}
	targetUserID, err := parsePositiveInt64(r.PathValue("user_id"))
	if err != nil {
		writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "invalid user_id"))
		return
	}
	if err := h.app.RemoveMember(r.Context(), userID, teamID, targetUserID); err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, statusResponse{Status: "ok"})
}