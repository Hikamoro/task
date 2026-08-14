package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"

	"task/internal/model"
	"task/internal/sqlc"
)

const mysqlDuplicateEntry = 1062

type Repository struct {
	db *sql.DB
}

func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == mysqlDuplicateEntry
}

func toUser(u sqlc.User) model.User {
	return model.User{
		ID:           int64(u.ID),
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Name:         u.Name,
		CreatedAt:    u.CreatedAt,
	}
}

func toTeam(t sqlc.Team) model.Team {
	return model.Team{
		ID:        int64(t.ID),
		Name:      t.Name,
		CreatedBy: int64(t.CreatedBy),
		CreatedAt: t.CreatedAt,
	}
}

func toRole(r sqlc.TeamMembersRole) model.Role {
	return model.Role(string(r))
}

func toTask(t sqlc.Task) model.Task {
	res := model.Task{
		ID:          int64(t.ID),
		TeamID:      int64(t.TeamID),
		Title:       t.Title,
		Description: t.Description,
		Status:      model.TaskStatus(t.Status),
		CreatedBy:   int64(t.CreatedBy),
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		Version:     int64(t.Version),
	}
	if t.AssigneeID.Valid {
		v := t.AssigneeID.Int64
		res.AssigneeID = &v
	}
	if t.ClosedAt.Valid {
		v := t.ClosedAt.Time
		res.ClosedAt = &v
	}
	return res
}

func nullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func nullTime(v *time.Time) sql.NullTime {
	if v == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *v, Valid: true}
}

// ---- Users ----

func (r *Repository) CreateUser(ctx context.Context, email, passwordHash, name string) (model.User, error) {
	id, err := sqlc.New(r.db).CreateUser(ctx, sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
	})
	if err != nil {
		if isDuplicate(err) {
			return model.User{}, model.ErrDuplicate
		}
		return model.User{}, err
	}
	return model.User{ID: id, Email: email, Name: name, CreatedAt: time.Now()}, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	u, err := sqlc.New(r.db).GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, model.ErrNotFound
		}
		return model.User{}, err
	}
	return toUser(u), nil
}

func (r *Repository) GetUserByID(ctx context.Context, id int64) (model.User, error) {
	u, err := sqlc.New(r.db).GetUserByID(ctx, uint64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, model.ErrNotFound
		}
		return model.User{}, err
	}
	return toUser(u), nil
}

// ---- Teams ----

func (r *Repository) CreateTeam(ctx context.Context, name string, ownerID int64) (model.Team, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Team{}, err
	}
	defer func() { _ = tx.Rollback() }()

	q := sqlc.New(tx)
	teamID, err := q.CreateTeam(ctx, sqlc.CreateTeamParams{
		Name:      name,
		CreatedBy: uint64(ownerID),
	})
	if err != nil {
		return model.Team{}, err
	}
	if err := q.AddTeamMember(ctx, sqlc.AddTeamMemberParams{
		TeamID: uint64(teamID),
		UserID: uint64(ownerID),
		Role:   sqlc.TeamMembersRoleOwner,
	}); err != nil {
		return model.Team{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.Team{}, err
	}
	return r.GetTeam(ctx, teamID)
}

func (r *Repository) GetTeam(ctx context.Context, id int64) (model.Team, error) {
	t, err := sqlc.New(r.db).GetTeamByID(ctx, uint64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Team{}, model.ErrNotFound
		}
		return model.Team{}, err
	}
	return toTeam(t), nil
}

func (r *Repository) ListUserTeams(ctx context.Context, userID int64) ([]model.Team, error) {
	rows, err := sqlc.New(r.db).ListUserTeams(ctx, uint64(userID))
	if err != nil {
		return nil, err
	}
	out := make([]model.Team, 0, len(rows))
	for _, t := range rows {
		out = append(out, toTeam(t))
	}
	return out, nil
}

func (r *Repository) AddMember(ctx context.Context, teamID, userID int64, role model.Role) error {
	err := sqlc.New(r.db).AddTeamMember(ctx, sqlc.AddTeamMemberParams{
		TeamID: uint64(teamID),
		UserID: uint64(userID),
		Role:   sqlc.TeamMembersRole(role),
	})
	if err != nil {
		if isDuplicate(err) {
			return model.ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *Repository) GetMemberRole(ctx context.Context, teamID, userID int64) (model.Role, error) {
	m, err := sqlc.New(r.db).GetTeamMember(ctx, sqlc.GetTeamMemberParams{
		TeamID: uint64(teamID),
		UserID: uint64(userID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", model.ErrNotFound
		}
		return "", err
	}
	return toRole(m.Role), nil
}

func (r *Repository) UpdateMemberRole(ctx context.Context, teamID, userID int64, role model.Role) error {
	rows, err := sqlc.New(r.db).UpdateTeamMemberRole(ctx, sqlc.UpdateTeamMemberRoleParams{
		Role:   sqlc.TeamMembersRole(role),
		TeamID: uint64(teamID),
		UserID: uint64(userID),
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *Repository) RemoveMember(ctx context.Context, teamID, userID int64) error {
	rows, err := sqlc.New(r.db).RemoveTeamMember(ctx, sqlc.RemoveTeamMemberParams{
		TeamID: uint64(teamID),
		UserID: uint64(userID),
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *Repository) ListMembers(ctx context.Context, teamID int64) ([]model.TeamMember, error) {
	rows, err := sqlc.New(r.db).ListTeamMembers(ctx, uint64(teamID))
	if err != nil {
		return nil, err
	}
	out := make([]model.TeamMember, 0, len(rows))
	for _, m := range rows {
		out = append(out, model.TeamMember{
			UserID: int64(m.UserID),
			Email:  m.Email,
			Name:   m.Name,
			Role:   toRole(m.Role),
			Joined: m.CreatedAt,
		})
	}
	return out, nil
}

// ---- Tasks ----

func (r *Repository) CreateTask(ctx context.Context, t model.Task) (model.Task, error) {
	id, err := sqlc.New(r.db).CreateTask(ctx, sqlc.CreateTaskParams{
		TeamID:      uint64(t.TeamID),
		Title:       t.Title,
		Description: t.Description,
		Status:      sqlc.TasksStatus(t.Status),
		CreatedBy:   uint64(t.CreatedBy),
		AssigneeID:  nullInt64(t.AssigneeID),
	})
	if err != nil {
		return model.Task{}, err
	}
	return r.GetTask(ctx, id)
}

func (r *Repository) GetTask(ctx context.Context, id int64) (model.Task, error) {
	t, err := sqlc.New(r.db).GetTaskByID(ctx, uint64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Task{}, model.ErrNotFound
		}
		return model.Task{}, err
	}
	return toTask(t), nil
}

func (r *Repository) ListTasks(ctx context.Context, teamID int64, status *model.TaskStatus, assigneeID *int64, limit, offset int32) ([]model.Task, int64, error) {
	q := sqlc.New(r.db)
	statusArg := sqlc.NullTasksStatus{}
	if status != nil {
		statusArg = sqlc.NullTasksStatus{TasksStatus: sqlc.TasksStatus(*status), Valid: true}
	}
	rows, err := q.ListTasks(ctx, sqlc.ListTasksParams{
		TeamID:     uint64(teamID),
		Status:     statusArg,
		AssigneeID: nullInt64(assigneeID),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountTasks(ctx, sqlc.CountTasksParams{
		TeamID:     uint64(teamID),
		Status:     statusArg,
		AssigneeID: nullInt64(assigneeID),
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]model.Task, 0, len(rows))
	for _, t := range rows {
		out = append(out, toTask(t))
	}
	return out, total, nil
}

// UpdateTaskWithHistory atomically applies an optimistic-locked task update and
// records the history entry inside a single transaction.
func (r *Repository) UpdateTaskWithHistory(ctx context.Context, t model.Task, expectedVersion int64, changedBy int64, changes []model.Change) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := sqlc.New(tx)
	rows, err := q.UpdateTask(ctx, sqlc.UpdateTaskParams{
		Title:       t.Title,
		Description: t.Description,
		Status:      sqlc.TasksStatus(t.Status),
		AssigneeID:  nullInt64(t.AssigneeID),
		ClosedAt:    nullTime(t.ClosedAt),
		ID:          uint64(t.ID),
		Version:     uint32(expectedVersion),
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return model.ErrConflict
	}
	changesJSON, err := json.Marshal(changes)
	if err != nil {
		return err
	}
	if _, err := q.CreateTaskHistory(ctx, sqlc.CreateTaskHistoryParams{
		TaskID:    uint64(t.ID),
		ChangedBy: uint64(changedBy),
		Changes:   string(changesJSON),
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) ListHistory(ctx context.Context, taskID int64) ([]model.TaskHistory, error) {
	rows, err := sqlc.New(r.db).ListTaskHistory(ctx, uint64(taskID))
	if err != nil {
		return nil, err
	}
	out := make([]model.TaskHistory, 0, len(rows))
	for _, h := range rows {
		var changes []model.Change
		_ = json.Unmarshal([]byte(h.Changes), &changes)
		out = append(out, model.TaskHistory{
			ID:        int64(h.ID),
			TaskID:    int64(h.TaskID),
			ChangedBy: int64(h.ChangedBy),
			Changes:   changes,
			CreatedAt: h.CreatedAt,
		})
	}
	return out, nil
}

// ---- Comments ----

func (r *Repository) CreateComment(ctx context.Context, c model.Comment) (model.Comment, error) {
	id, err := sqlc.New(r.db).CreateComment(ctx, sqlc.CreateCommentParams{
		TaskID:  uint64(c.TaskID),
		UserID:  uint64(c.UserID),
		Content: c.Content,
	})
	if err != nil {
		return model.Comment{}, err
	}
	c.ID = id
	c.CreatedAt = time.Now()
	return c, nil
}

func (r *Repository) ListComments(ctx context.Context, taskID int64) ([]model.Comment, error) {
	rows, err := sqlc.New(r.db).ListTaskComments(ctx, uint64(taskID))
	if err != nil {
		return nil, err
	}
	out := make([]model.Comment, 0, len(rows))
	for _, c := range rows {
		out = append(out, model.Comment{
			ID:        int64(c.ID),
			TaskID:    int64(c.TaskID),
			UserID:    int64(c.UserID),
			Content:   c.Content,
			CreatedAt: c.CreatedAt,
		})
	}
	return out, nil
}

// ---- Stats ----

func (r *Repository) TeamStats(ctx context.Context, teamID int64) (model.TeamStats, error) {
	row, err := sqlc.New(r.db).TeamStats(ctx, uint64(teamID))
	if err != nil {
		return model.TeamStats{}, err
	}
	var assignees []model.AssigneeStat
	_ = json.Unmarshal([]byte(row.TopAssignees), &assignees)
	if assignees == nil {
		assignees = []model.AssigneeStat{}
	}
	return model.TeamStats{
		TodoCount:          row.TodoCount,
		InProgressCount:    row.InProgressCount,
		DoneCount:          row.DoneCount,
		CommentsCount:      row.CommentsCount,
		AvgCloseSeconds:    row.AvgCloseSeconds,
		TopAssigneesDone30: row.TopAssigneesDone30d,
		TopAssignees:       assignees,
	}, nil
}
