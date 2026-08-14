package service_test

import (
	"context"
	"errors"
	"testing"

	"task/internal/model"
	"task/internal/service"
)

func TestTeamStatsReport(t *testing.T) {
	ctx := context.Background()

	ownerID := seedUser(t, "stats-owner@test.com", "Owner Name")
	memberID := seedUser(t, "stats-member@test.com", "Member Name")
	assigneeA := seedUser(t, "stats-assignee-a@test.com", "Assignee A")
	assigneeB := seedUser(t, "stats-assignee-b@test.com", "Assignee B")

	teamID := seedTeam(t, ownerID, "Stats Team")
	if err := tsApp.Invite(ctx, ownerID, teamID, "stats-member@test.com", model.RoleMember); err != nil {
		t.Fatalf("invite member: %v", err)
	}
	if err := tsApp.Invite(ctx, ownerID, teamID, "stats-assignee-a@test.com", model.RoleAdmin); err != nil {
		t.Fatalf("invite assignee a: %v", err)
	}
	if err := tsApp.Invite(ctx, ownerID, teamID, "stats-assignee-b@test.com", model.RoleMember); err != nil {
		t.Fatalf("invite assignee b: %v", err)
	}

	aID := int64(assigneeA)
	bID := int64(assigneeB)

	taskTodo := createTask(ctx, t, ownerID, teamID, "todo task", nil)
	taskProgress := createTask(ctx, t, memberID, teamID, "in progress task", nil)
	taskDone1 := createTask(ctx, t, ownerID, teamID, "done task 1", &aID)
	taskDone2 := createTask(ctx, t, ownerID, teamID, "done task 2", &aID)
	taskDone3 := createTask(ctx, t, memberID, teamID, "done task 3", &bID)
	taskOld := createTask(ctx, t, memberID, teamID, "old done task", &bID)

	progress := model.StatusInProgress
	if _, err := tsApp.UpdateTask(ctx, ownerID, taskProgress, service.TaskUpdate{Status: &progress, Version: 1}); err != nil {
		t.Fatalf("set in_progress: %v", err)
	}

	setTaskDone(t, taskDone1, 0, 2)
	setTaskDone(t, taskDone2, 0, 2)
	setTaskDone(t, taskDone3, 0, 2)
	setTaskDone(t, taskOld, 100, 2)

	if _, err := tsApp.CreateComment(ctx, memberID, taskTodo, "comment one"); err != nil {
		t.Fatalf("comment 1: %v", err)
	}
	if _, err := tsApp.CreateComment(ctx, memberID, taskTodo, "comment two"); err != nil {
		t.Fatalf("comment 2: %v", err)
	}
	if _, err := tsApp.CreateComment(ctx, memberID, taskDone1, "comment three"); err != nil {
		t.Fatalf("comment 3: %v", err)
	}

	stats, err := tsApp.GetStats(ctx, ownerID, teamID)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}

	if stats.TodoCount != 1 {
		t.Errorf("todo_count = %d, want 1", stats.TodoCount)
	}
	if stats.InProgressCount != 1 {
		t.Errorf("in_progress_count = %d, want 1", stats.InProgressCount)
	}
	if stats.DoneCount != 4 {
		t.Errorf("done_count = %d, want 4", stats.DoneCount)
	}
	if stats.CommentsCount != 3 {
		t.Errorf("comments_count = %d, want 3", stats.CommentsCount)
	}
	if stats.AvgCloseSeconds <= 0 {
		t.Errorf("avg_close_seconds = %v, want > 0", stats.AvgCloseSeconds)
	}
	if stats.TopAssigneesDone30 != 3 {
		t.Errorf("top_assignees_done_30d = %d, want 3", stats.TopAssigneesDone30)
	}
	if len(stats.TopAssignees) != 2 {
		t.Fatalf("top_assignees len = %d, want 2: %+v", len(stats.TopAssignees), stats.TopAssignees)
	}
	if stats.TopAssignees[0].ClosedCount != 2 {
		t.Errorf("top assignee #1 closed = %d, want 2", stats.TopAssignees[0].ClosedCount)
	}
	if stats.TopAssignees[1].ClosedCount != 1 {
		t.Errorf("top assignee #2 closed = %d, want 1", stats.TopAssignees[1].ClosedCount)
	}

	// Access control: a plain member must NOT see the report.
	if _, err := tsApp.GetStats(ctx, memberID, teamID); !errors.Is(err, model.ErrForbidden) {
		t.Errorf("member stats err = %v, want ErrForbidden", err)
	}
	// An admin may see the report.
	if _, err := tsApp.GetStats(ctx, assigneeA, teamID); err != nil {
		t.Errorf("admin stats err = %v, want nil", err)
	}
}
