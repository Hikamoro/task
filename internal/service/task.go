package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"task/internal/cache"
	"task/internal/model"
)

type TaskFilters struct {
	Status     *model.TaskStatus
	AssigneeID *int64
	Limit      int32
	Offset     int32
}

func (a *App) CreateTask(ctx context.Context, userID, teamID int64, title, description string, assigneeID *int64) (model.Task, error) {
	if _, err := a.requireTeamMember(ctx, teamID, userID); err != nil {
		return model.Task{}, err
	}
	title = strings.TrimSpace(title)
	if title == "" || len(title) > 255 {
		return model.Task{}, model.ErrInvalidInput
	}
	if len(description) > 65535 {
		return model.Task{}, model.ErrInvalidInput
	}
	if assigneeID != nil {
		if _, err := a.repo.GetMemberRole(ctx, teamID, *assigneeID); err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.Task{}, model.ErrForbidden
			}
			return model.Task{}, err
		}
	}

	task, err := a.repo.CreateTask(ctx, model.Task{
		TeamID:      teamID,
		Title:       title,
		Description: description,
		Status:      model.StatusTodo,
		CreatedBy:   userID,
		AssigneeID:  assigneeID,
	})
	if err != nil {
		return model.Task{}, err
	}
	a.invalidateTeamCache(ctx, teamID)
	return task, nil
}

func (a *App) ListTasks(ctx context.Context, userID, teamID int64, f TaskFilters) (model.TaskList, error) {
	if _, err := a.requireTeamMember(ctx, teamID, userID); err != nil {
		return model.TaskList{}, err
	}

	fp := cache.Fingerprint(f.Status, f.AssigneeID, f.Limit, f.Offset)
	if a.cache != nil {
		if ver, err := a.cache.Version(ctx, teamID); err == nil {
			if entry, ok, err := a.cache.Get(ctx, teamID, ver, fp); err == nil && ok {
				return entry, nil
			}
		}
	}

	tasks, total, err := a.repo.ListTasks(ctx, teamID, f.Status, f.AssigneeID, f.Limit, f.Offset)
	if err != nil {
		return model.TaskList{}, err
	}
	entry := model.TaskList{Tasks: tasks, Total: total}
	if a.cache != nil {
		if ver, err := a.cache.Version(ctx, teamID); err == nil {
			_ = a.cache.Set(ctx, teamID, ver, fp, entry)
		}
	}
	return entry, nil
}

type TaskUpdate struct {
	Title       *string
	Description *string
	Status      *model.TaskStatus
	AssigneeID  *int64
	Version     int64
}

func (a *App) UpdateTask(ctx context.Context, userID, taskID int64, upd TaskUpdate) (model.Task, error) {
	current, err := a.repo.GetTask(ctx, taskID)
	if err != nil {
		return model.Task{}, err
	}
	role, err := a.requireTeamMember(ctx, current.TeamID, userID)
	if err != nil {
		return model.Task{}, err
	}

	canFull := role == model.RoleOwner || role == model.RoleAdmin
	canTitleDesc := canFull || current.CreatedBy == userID
	canAssign := canFull || current.CreatedBy == userID
	canStatus := canFull || current.CreatedBy == userID ||
		(current.AssigneeID != nil && *current.AssigneeID == userID)

	if upd.Title != nil {
		if !canTitleDesc {
			return model.Task{}, model.ErrForbidden
		}
		title := strings.TrimSpace(*upd.Title)
		if title == "" || len(title) > 255 {
			return model.Task{}, model.ErrInvalidInput
		}
		upd.Title = &title
	}
	if upd.Description != nil {
		if !canTitleDesc {
			return model.Task{}, model.ErrForbidden
		}
		if len(*upd.Description) > 65535 {
			return model.Task{}, model.ErrInvalidInput
		}
	}
	if upd.Status != nil {
		if !canStatus {
			return model.Task{}, model.ErrForbidden
		}
		if !upd.Status.Valid() {
			return model.Task{}, model.ErrInvalidInput
		}
	}
	if upd.AssigneeID != nil {
		if !canAssign {
			return model.Task{}, model.ErrForbidden
		}
		if _, err := a.repo.GetMemberRole(ctx, current.TeamID, *upd.AssigneeID); err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.Task{}, model.ErrForbidden
			}
			return model.Task{}, err
		}
	}

	newTask := current
	var changes []model.Change
	if upd.Title != nil {
		if *upd.Title != current.Title {
			changes = append(changes, model.Change{Field: "title", OldValue: current.Title, NewValue: *upd.Title})
		}
		newTask.Title = *upd.Title
	}
	if upd.Description != nil {
		if *upd.Description != current.Description {
			changes = append(changes, model.Change{Field: "description", OldValue: current.Description, NewValue: *upd.Description})
		}
		newTask.Description = *upd.Description
	}
	if upd.Status != nil && *upd.Status != current.Status {
		now := time.Now()
		switch *upd.Status {
		case model.StatusDone:
			if current.ClosedAt == nil {
				closed := now
				newTask.ClosedAt = &closed
			}
		default:
			newTask.ClosedAt = nil
		}
		changes = append(changes, model.Change{Field: "status", OldValue: current.Status, NewValue: *upd.Status})
		newTask.Status = *upd.Status
	}
	if upd.AssigneeID != nil && !samePtr(current.AssigneeID, upd.AssigneeID) {
		changes = append(changes, model.Change{Field: "assignee_id", OldValue: current.AssigneeID, NewValue: *upd.AssigneeID})
		newTask.AssigneeID = upd.AssigneeID
	}

	if len(changes) == 0 {
		return current, nil
	}

	if err := a.repo.UpdateTaskWithHistory(ctx, newTask, upd.Version, userID, changes); err != nil {
		return model.Task{}, err
	}
	a.invalidateTeamCache(ctx, current.TeamID)
	return a.repo.GetTask(ctx, taskID)
}

func (a *App) GetHistory(ctx context.Context, userID, taskID int64) ([]model.TaskHistory, error) {
	task, err := a.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if _, err := a.requireTeamMember(ctx, task.TeamID, userID); err != nil {
		return nil, err
	}
	return a.repo.ListHistory(ctx, taskID)
}

func samePtr(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
