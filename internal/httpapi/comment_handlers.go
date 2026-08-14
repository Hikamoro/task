package httpapi

import "net/http"

type createCommentRequest struct {
	Content string `json:"content"`
}

// CreateComment adds a comment to a task.
// @Summary Add a comment
// @Tags comments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Param body body createCommentRequest true "Comment payload"
// @Success 201 {object} commentResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /tasks/{id}/comments [post]
func (h *handlers) CreateComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	taskID, err := parsePositiveInt64(r.PathValue("id"))
	if err != nil {
		writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "invalid task id"))
		return
	}
	var req createCommentRequest
	if err := decodeBody(w, r, &req, h.maxBodyBytes); err != nil {
		writeError(w, h.logger, err)
		return
	}
	comment, err := h.app.CreateComment(r.Context(), userID, taskID, req.Content)
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, commentResponse{Comment: comment})
}

// ListComments lists comments of a task.
// @Summary List task comments
// @Tags comments
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Success 200 {object} commentsResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /tasks/{id}/comments [get]
func (h *handlers) ListComments(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	taskID, err := parsePositiveInt64(r.PathValue("id"))
	if err != nil {
		writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "invalid task id"))
		return
	}
	comments, err := h.app.ListComments(r.Context(), userID, taskID)
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, commentsResponse{Comments: comments})
}

// TeamStats returns the SQL report for a team. Owner/admin only.
// @Summary Team statistics report
// @Tags stats
// @Produce json
// @Security BearerAuth
// @Param team_id path int true "Team ID"
// @Success 200 {object} statsResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /teams/{team_id}/stats [get]
func (h *handlers) TeamStats(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	teamID, err := parsePositiveInt64(r.PathValue("team_id"))
	if err != nil {
		writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "invalid team_id"))
		return
	}
	stats, err := h.app.GetStats(r.Context(), userID, teamID)
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, statsResponse{Stats: stats})
}