-- name: GetSessionUser :one
SELECT
    sqlc.embed(u),
    s.expires_at
FROM
    sessions s
    JOIN users u ON s.user_id = u.id
WHERE
    s.token_hash = sqlc.arg(token_hash)
    AND s.expires_at > sqlc.arg(now);

-- name: InsertSession :exec
INSERT INTO sessions(token_hash, user_id, created_at, expires_at)
    VALUES (?, ?, ?, ?);

-- name: UpdateSessionExpiry :exec
UPDATE
    sessions
SET
    expires_at = ?
WHERE
    token_hash = ?;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE token_hash = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at <= ?;

