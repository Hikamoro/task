package model

import "time"

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

func ParseRole(s string) (Role, bool) {
	switch Role(s) {
	case RoleOwner, RoleAdmin, RoleMember:
		return Role(s), true
	}
	return "", false
}

func (r Role) Valid() bool {
	_, ok := ParseRole(string(r))
	return ok
}

type TaskStatus string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

func ParseTaskStatus(s string) (TaskStatus, bool) {
	switch TaskStatus(s) {
	case StatusTodo, StatusInProgress, StatusDone:
		return TaskStatus(s), true
	}
	return "", false
}

func (s TaskStatus) Valid() bool {
	_, ok := ParseTaskStatus(string(s))
	return ok
}

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
}

type Team struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type TeamMember struct {
	UserID int64     `json:"user_id"`
	Email  string    `json:"email"`
	Name   string    `json:"name"`
	Role   Role      `json:"role"`
	Joined time.Time `json:"joined_at"`
}

type Task struct {
	ID          int64      `json:"id"`
	TeamID      int64      `json:"team_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	CreatedBy   int64      `json:"created_by"`
	AssigneeID  *int64     `json:"assignee_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at"`
	Version     int64      `json:"version"`
}

type Change struct {
	Field    string `json:"field"`
	OldValue any    `json:"old_value"`
	NewValue any    `json:"new_value"`
}

type TaskHistory struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	ChangedBy int64     `json:"changed_by"`
	Changes   []Change  `json:"changes"`
	CreatedAt time.Time `json:"created_at"`
}

type Comment struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	UserID    int64     `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type AssigneeStat struct {
	Name        string `json:"name"`
	ClosedCount int64  `json:"closed_count"`
}

type TeamStats struct {
	TodoCount          int64          `json:"todo_count"`
	InProgressCount    int64          `json:"in_progress_count"`
	DoneCount          int64          `json:"done_count"`
	CommentsCount      int64          `json:"comments_count"`
	AvgCloseSeconds    float64        `json:"avg_close_seconds"`
	TopAssigneesDone30 int64          `json:"top_assignees_done_30d"`
	TopAssignees       []AssigneeStat `json:"top_assignees"`
}

type TaskList struct {
	Tasks []Task `json:"tasks"`
	Total int64  `json:"total"`
}