-- name: CreateTeam :execlastid
INSERT INTO teams (name, created_by)
VALUES (?, ?);

-- name: GetTeamByID :one
SELECT id, name, created_by, created_at
FROM teams
WHERE id = ?;

-- name: AddTeamMember :exec
INSERT INTO team_members (team_id, user_id, role)
VALUES (?, ?, ?);

-- name: UpdateTeamMemberRole :execrows
UPDATE team_members
SET role = ?
WHERE team_id = ? AND user_id = ?;

-- name: RemoveTeamMember :execrows
DELETE FROM team_members
WHERE team_id = ? AND user_id = ?;

-- name: GetTeamMember :one
SELECT team_id, user_id, role, created_at
FROM team_members
WHERE team_id = ? AND user_id = ?;

-- name: GetTeamMemberForUpdate :one
SELECT team_id, user_id, role, created_at
FROM team_members
WHERE team_id = ? AND user_id = ?
FOR UPDATE;

-- name: CountTeamOwners :one
SELECT COUNT(*)
FROM team_members
WHERE team_id = ? AND role = 'owner';

-- name: ListUserTeams :many
SELECT t.id, t.name, t.created_by, t.created_at
FROM teams t
         JOIN team_members tm ON tm.team_id = t.id
WHERE tm.user_id = ?
ORDER BY t.created_at DESC, t.id DESC;

-- name: ListTeamMembers :many
SELECT u.id AS user_id, u.email, u.name, tm.role, tm.created_at
FROM team_members tm
         JOIN users u ON u.id = tm.user_id
WHERE tm.team_id = ?
ORDER BY tm.created_at ASC, u.id ASC;
