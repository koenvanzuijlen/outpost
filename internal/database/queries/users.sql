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

-- name: UpdateUserForSetup :execrows
UPDATE
    users
SET
    username = ?,
    password_hash = ?,
    role = ?
WHERE
    id = ?;

