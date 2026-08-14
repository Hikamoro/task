package service

import (
	"context"
	"strings"

	"task/internal/model"
)

func (a *App) CreateComment(ctx context.Context, userID, taskID int64, content string) (model.Comment, error) {
	task, err := a.repo.GetTask(ctx, taskID)
	if err != nil {
		return model.Comment{}, err
	}
	if _, err := a.requireTeamMember(ctx, task.TeamID, userID); err != nil {
		return model.Comment{}, err
	}
	content = strings.TrimSpace(content)
	if content == "" || len(content) > 65535 {
		return model.Comment{}, model.ErrInvalidInput
	}
	return a.repo.CreateComment(ctx, model.Comment{TaskID: taskID, UserID: userID, Content: content})
}

func (a *App) ListComments(ctx context.Context, userID, taskID int64) ([]model.Comment, error) {
	task, err := a.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if _, err := a.requireTeamMember(ctx, task.TeamID, userID); err != nil {
		return nil, err
	}
	return a.repo.ListComments(ctx, taskID)
}
