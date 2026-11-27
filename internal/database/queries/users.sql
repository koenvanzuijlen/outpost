-- name: GetUserByID :one
SELECT
    *
FROM
    users
WHERE
    id = ?;

-- name: GetUserByUsername :one
SELECT
    *
FROM
    users
WHERE
    username = ?;

-- name: CountUsersWithRole :one
SELECT
    COUNT(*)
FROM
    users
WHERE
    role = ?;

-- name: ListUsers :many
SELECT
    *
FROM
    users
ORDER BY
    username ASC;

-- name: InsertUser :one
INSERT INTO users(username, password_hash, role)
    VALUES (?, ?, ?)
RETURNING
    *;

-- name: UpdateUsername :execrows
UPDATE
    users
SET
    username = ?
WHERE
    id = ?;

-- name: UpdatePasswordHash :execrows
UPDATE
    users
SET
    password_hash = ?
WHERE
    id = ?;

-- name: UpdateUserRole :execrows
UPDATE
    users
SET
    role = ?
WHERE
    id = ?;

-- name: UpdateUserTimeZone :execrows
UPDATE
    users
SET
    time_zone = ?
WHERE
    id = ?;

-- name: UpdateUserTheme :execrows
UPDATE
    users
SET
    theme = ?
WHERE
    id = ?;

-- name: UpdateUserForSetup :execrows
UPDATE
    users
SET
    username = ?,
    password_hash = ?,
    role = ?
WHERE
    id = ?;

-- name: DeleteUser :execrows
DELETE FROM users
WHERE id = ?;

