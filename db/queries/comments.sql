-- name: CreateComment :execlastid
INSERT INTO task_comments (task_id, user_id, content)
VALUES (?, ?, ?);

-- name: ListTaskComments :many
SELECT id, task_id, user_id, content, created_at
FROM task_comments
WHERE task_id = ?
ORDER BY created_at DESC, id DESC;
