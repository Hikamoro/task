-- name: CreateTaskHistory :execlastid
INSERT INTO task_history (task_id, changed_by, changes)
VALUES (?, ?, ?);

-- name: ListTaskHistory :many
SELECT id, task_id, changed_by, changes, created_at
FROM task_history
WHERE task_id = ?
ORDER BY created_at DESC, id DESC;
