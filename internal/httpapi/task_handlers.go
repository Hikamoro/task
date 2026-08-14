package httpapi

import (
	"net/http"

	"task/internal/model"
	"task/internal/service"
)

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	AssigneeID  *int64 `json:"assignee_id"`
}

// CreateTask creates a task in a team.
// @Summary Create a task
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param team_id query int true "Team ID"
// @Param body body createTaskRequest true "Task payload"
// @Success 201 {object} taskResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Router /tasks [post]
func (h *handlers) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	teamID, err := parsePositiveInt64(r.URL.Query().Get("team_id"))
	if err != nil {
		writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "team_id is required and must be positive"))
		return
	}
	var req createTaskRequest
	if err := decodeBody(w, r, &req, h.maxBodyBytes); err != nil {
		writeError(w, h.logger, err)
		return
	}
	task, err := h.app.CreateTask(r.Context(), userID, teamID, req.Title, req.Description, req.AssigneeID)
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, taskResponse{Task: task})
}

// ListTasks lists tasks of a team with filters and pagination.
// @Summary List tasks
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param team_id query int true "Team ID"
// @Param status query string false "Filter by status: todo, in_progress, done"
// @Param assignee_id query int false "Filter by assignee"
// @Param limit query int false "Page size (1..100, default 20)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {object} taskListResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Router /tasks [get]
func (h *handlers) ListTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	teamID, err := parsePositiveInt64(r.URL.Query().Get("team_id"))
	if err != nil {
		writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "team_id is required and must be positive"))
		return
	}

	q := r.URL.Query()
	var status *model.TaskStatus
	if s := q.Get("status"); s != "" {
		parsed, ok := model.ParseTaskStatus(s)
		if !ok {
			writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "invalid status"))
			return
		}
		status = &parsed
	}
	var assigneeID *int64
	if s := q.Get("assignee_id"); s != "" {
		n, perr := parsePositiveInt64(s)
		if perr != nil {
			writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "invalid assignee_id"))
			return
		}
		assigneeID = &n
	}
	limit, offset, err := parsePagination(q)
	if err != nil {
		writeError(w, h.logger, err)
		return
	}

	res, err := h.app.ListTasks(r.Context(), userID, teamID, service.TaskFilters{
		Status:     status,
		AssigneeID: assigneeID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, taskListResponse{
		Tasks:  res.Tasks,
		Total:  res.Total,
		Limit:  limit,
		Offset: offset,
	})
}

type updateTaskRequest struct {
	Title       *string           `json:"title"`
	Description *string           `json:"description"`
	Status      *model.TaskStatus `json:"status"`
	AssigneeID  *int64            `json:"assignee_id"`
	Version     int64             `json:"version"`
}

// UpdateTask updates a task (optimistic concurrency via version).
// @Summary Update a task
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Param body body updateTaskRequest true "Update payload (only provided fields change; version is required)"
// @Success 200 {object} taskResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Router /tasks/{id} [put]
func (h *handlers) UpdateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	taskID, err := parsePositiveInt64(r.PathValue("id"))
	if err != nil {
		writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "invalid task id"))
		return
	}
	var req updateTaskRequest
	if err := decodeBody(w, r, &req, h.maxBodyBytes); err != nil {
		writeError(w, h.logger, err)
		return
	}
	if req.Version < 1 {
		writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "version is required and must be positive"))
		return
	}
	task, err := h.app.UpdateTask(r.Context(), userID, taskID, service.TaskUpdate{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		AssigneeID:  req.AssigneeID,
		Version:     req.Version,
	})
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, taskResponse{Task: task})
}

// TaskHistory returns the change history of a task.
// @Summary Get task history
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Success 200 {object} historyResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /tasks/{id}/history [get]
func (h *handlers) TaskHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	taskID, err := parsePositiveInt64(r.PathValue("id"))
	if err != nil {
		writeError(w, h.logger, newAPIError(http.StatusBadRequest, "invalid_request", "invalid task id"))
		return
	}
	history, err := h.app.GetHistory(r.Context(), userID, taskID)
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, historyResponse{History: history})
}