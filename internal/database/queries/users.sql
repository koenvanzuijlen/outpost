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

-- name: UpdateUserForSetup :execrows
UPDATE
    users
SET
    username = ?,
    password_hash = ?,
    role = ?
WHERE
    id = ?;

