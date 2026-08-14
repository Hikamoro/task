-- name: CreateTask :execlastid
INSERT INTO tasks (team_id, title, description, status, created_by, assignee_id)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetTaskByID :one
SELECT id, team_id, title, description, status, created_by, assignee_id,
       created_at, updated_at, closed_at, version
FROM tasks
WHERE id = ?;

-- name: UpdateTask :execrows
UPDATE tasks
SET title       = ?,
    description = ?,
    status      = ?,
    assignee_id = ?,
    closed_at   = ?,
    updated_at  = CURRENT_TIMESTAMP,
    version     = version + 1
WHERE id = ? AND version = ?;

-- name: ListTasks :many
SELECT id, team_id, title, description, status, created_by, assignee_id,
       created_at, updated_at, closed_at, version
FROM tasks
WHERE team_id = ?
  AND (sqlc.narg('status') IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('assignee_id') IS NULL OR assignee_id = sqlc.narg('assignee_id'))
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: CountTasks :one
SELECT COUNT(*)
FROM tasks
WHERE team_id = ?
  AND (sqlc.narg('status') IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('assignee_id') IS NULL OR assignee_id = sqlc.narg('assignee_id'));
