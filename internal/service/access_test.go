package service_test

import (
	"context"
	"errors"
	"testing"

	"task/internal/model"
	"task/internal/service"
)

func TestTaskAccessControl(t *testing.T) {
	ctx := context.Background()

	ownerID := seedUser(t, "acl-owner@test.com", "Owner")
	adminID := seedUser(t, "acl-admin@test.com", "Admin")
	memberID := seedUser(t, "acl-member@test.com", "Member")
	outsiderID := seedUser(t, "acl-outsider@test.com", "Outsider")

	teamID := seedTeam(t, ownerID, "ACL Team")
	if err := tsApp.Invite(ctx, ownerID, teamID, "acl-admin@test.com", model.RoleAdmin); err != nil {
		t.Fatalf("invite admin: %v", err)
	}
	if err := tsApp.Invite(ctx, ownerID, teamID, "acl-member@test.com", model.RoleMember); err != nil {
		t.Fatalf("invite member: %v", err)
	}

	// An outsider cannot create a task in a team he does not belong to.
	if _, err := tsApp.CreateTask(ctx, outsiderID, teamID, "sneaky", "x", nil); !errors.Is(err, model.ErrForbidden) {
		t.Errorf("outsider create task err = %v, want ErrForbidden", err)
	}

	// Owner creates a task assigned to the member.
	memberIDp := int64(memberID)
	taskID := createTask(ctx, t, ownerID, teamID, "assigned task", &memberIDp)

	// The assignee may change the status.
	done := model.StatusDone
	updated, err := tsApp.UpdateTask(ctx, memberID, taskID, service.TaskUpdate{Status: &done, Version: 1})
	if err != nil {
		t.Fatalf("assignee change status: %v", err)
	}
	if updated.Version != 2 {
		t.Errorf("task version = %d, want 2 after update", updated.Version)
	}

	// The assignee may NOT reassign the task.
	newAssignee := int64(adminID)
	if _, err := tsApp.UpdateTask(ctx, memberID, taskID, service.TaskUpdate{AssigneeID: &newAssignee, Version: 2}); !errors.Is(err, model.ErrForbidden) {
		t.Errorf("assignee reassign err = %v, want ErrForbidden", err)
	}

	// The assignee may NOT change the title.
	title := "hacked"
	if _, err := tsApp.UpdateTask(ctx, memberID, taskID, service.TaskUpdate{Title: &title, Version: 2}); !errors.Is(err, model.ErrForbidden) {
		t.Errorf("assignee change title err = %v, want ErrForbidden", err)
	}

	// A plain member (neither creator nor assignee) cannot edit the task.
	otherTaskID := createTask(ctx, t, ownerID, teamID, "other task", nil)
	if _, err := tsApp.UpdateTask(ctx, memberID, otherTaskID, service.TaskUpdate{Status: &done, Version: 1}); !errors.Is(err, model.ErrForbidden) {
		t.Errorf("plain member edit err = %v, want ErrForbidden", err)
	}

	// The task creator can reassign to a team member.
	if _, err := tsApp.UpdateTask(ctx, ownerID, taskID, service.TaskUpdate{AssigneeID: &newAssignee, Version: 2}); err != nil {
		t.Errorf("owner reassign err = %v, want nil", err)
	}

	// An admin may edit anything.
	if _, err := tsApp.UpdateTask(ctx, adminID, otherTaskID, service.TaskUpdate{Title: &title, Version: 1}); err != nil {
		t.Errorf("admin edit err = %v, want nil", err)
	}

	// An outsider cannot list tasks or read history/comments of the team.
	if _, err := tsApp.ListTasks(ctx, outsiderID, teamID, service.TaskFilters{Limit: 20}); !errors.Is(err, model.ErrForbidden) {
		t.Errorf("outsider list tasks err = %v, want ErrForbidden", err)
	}
	if _, err := tsApp.GetHistory(ctx, outsiderID, taskID); !errors.Is(err, model.ErrForbidden) {
		t.Errorf("outsider history err = %v, want ErrForbidden", err)
	}
	if _, err := tsApp.ListComments(ctx, outsiderID, taskID); !errors.Is(err, model.ErrForbidden) {
		t.Errorf("outsider comments err = %v, want ErrForbidden", err)
	}

	// A member cannot see members of a team he does not belong to.
	if _, err := tsApp.ListMembers(ctx, outsiderID, teamID); !errors.Is(err, model.ErrForbidden) {
		t.Errorf("outsider list members err = %v, want ErrForbidden", err)
	}
}

func TestOptimisticLockConflict(t *testing.T) {
	ctx := context.Background()
	ownerID := seedUser(t, "lock-owner@test.com", "Lock Owner")
	teamID := seedTeam(t, ownerID, "Lock Team")

	taskID := createTask(ctx, t, ownerID, teamID, "contended task", nil)

	t1 := "first write"
	t2 := "second write"
	if _, err := tsApp.UpdateTask(ctx, ownerID, taskID, service.TaskUpdate{Title: &t1, Version: 1}); err != nil {
		t.Fatalf("first update: %v", err)
	}
	// Both updates were issued against version 1, so the second must conflict.
	if _, err := tsApp.UpdateTask(ctx, ownerID, taskID, service.TaskUpdate{Title: &t2, Version: 1}); !errors.Is(err, model.ErrConflict) {
		t.Errorf("stale update err = %v, want ErrConflict", err)
	}

	// Retrying with the fresh version succeeds.
	task, err := tsRepo.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if _, err := tsApp.UpdateTask(ctx, ownerID, taskID, service.TaskUpdate{Title: &t2, Version: task.Version}); err != nil {
		t.Errorf("retry update err = %v, want nil", err)
	}
}

func TestRoleManagement(t *testing.T) {
	ctx := context.Background()
	ownerID := seedUser(t, "role-owner@test.com", "Role Owner")
	adminID := seedUser(t, "role-admin@test.com", "Role Admin")
	memberID := seedUser(t, "role-member@test.com", "Role Member")
	inviteeID := seedUser(t, "role-invitee@test.com", "Role Invitee")

	teamID := seedTeam(t, ownerID, "Role Team")
	if err := tsApp.Invite(ctx, ownerID, teamID, "role-admin@test.com", model.RoleAdmin); err != nil {
		t.Fatalf("invite admin: %v", err)
	}
	if err := tsApp.Invite(ctx, ownerID, teamID, "role-member@test.com", model.RoleMember); err != nil {
		t.Fatalf("invite member: %v", err)
	}

	// A plain member cannot invite.
	if err := tsApp.Invite(ctx, memberID, teamID, "role-invitee@test.com", model.RoleMember); !errors.Is(err, model.ErrForbidden) {
		t.Errorf("member invite err = %v, want ErrForbidden", err)
	}

	// An admin can invite but cannot grant the owner role.
	if err := tsApp.Invite(ctx, adminID, teamID, "role-invitee@test.com", model.RoleOwner); !errors.Is(err, model.ErrInvalidInput) {
		t.Errorf("invite as owner err = %v, want ErrInvalidInput", err)
	}
	if err := tsApp.Invite(ctx, adminID, teamID, "role-invitee@test.com", model.RoleMember); err != nil {
		t.Errorf("admin invite err = %v, want nil", err)
	}

	// Admin cannot change the owner's role.
	if err := tsApp.UpdateMemberRole(ctx, adminID, teamID, ownerID, model.RoleAdmin); !errors.Is(err, model.ErrForbidden) {
		t.Errorf("admin change owner role err = %v, want ErrForbidden", err)
	}
	// Owner cannot change his own owner role (immutable owner).
	if err := tsApp.UpdateMemberRole(ctx, ownerID, teamID, ownerID, model.RoleAdmin); !errors.Is(err, model.ErrForbidden) {
		t.Errorf("owner self-demote err = %v, want ErrForbidden", err)
	}

	// Owner can promote the member to admin, then demote back.
	if err := tsApp.UpdateMemberRole(ctx, ownerID, teamID, memberID, model.RoleAdmin); err != nil {
		t.Errorf("promote to admin err = %v, want nil", err)
	}
	if err := tsApp.UpdateMemberRole(ctx, ownerID, teamID, memberID, model.RoleMember); err != nil {
		t.Errorf("demote to member err = %v, want nil", err)
	}

	// The owner cannot be removed.
	if err := tsApp.RemoveMember(ctx, adminID, teamID, ownerID); !errors.Is(err, model.ErrForbidden) {
		t.Errorf("remove owner err = %v, want ErrForbidden", err)
	}
	// A member can be removed by an admin.
	if err := tsApp.RemoveMember(ctx, adminID, teamID, inviteeID); err != nil {
		t.Errorf("admin remove member err = %v, want nil", err)
	}
}

func TestTaskHistoryRecording(t *testing.T) {
	ctx := context.Background()
	ownerID := seedUser(t, "hist-owner@test.com", "Hist Owner")
	teamID := seedTeam(t, ownerID, "Hist Team")

	done := model.StatusDone
	taskID := createTask(ctx, t, ownerID, teamID, "task to edit", nil)
	if _, err := tsApp.UpdateTask(ctx, ownerID, taskID, service.TaskUpdate{
		Title:  stringPtr("renamed task"),
		Status: &done,
		Version: 1,
	}); err != nil {
		t.Fatalf("update task: %v", err)
	}

	history, err := tsApp.GetHistory(ctx, ownerID, taskID)
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1", len(history))
	}
	fields := map[string]bool{}
	for _, c := range history[0].Changes {
		fields[c.Field] = true
	}
	if !fields["title"] || !fields["status"] {
		t.Errorf("history changes = %+v, want title and status", history[0].Changes)
	}
	if history[0].ChangedBy != ownerID {
		t.Errorf("changed_by = %d, want %d", history[0].ChangedBy, ownerID)
	}
}

func stringPtr(s string) *string { return &s }
