-- name: ListPlaythroughsForUserGameID :many
SELECT
    sqlc.embed(pt),
    p.name AS platform_name
FROM
    playthroughs pt
    LEFT JOIN platforms p ON pt.platform_id = p.id
WHERE
    pt.user_game_id = ?
ORDER BY
    pt.started_at DESC,
    pt.id DESC;

-- name: GetPlaythrough :one
SELECT
    *
FROM
    playthroughs
WHERE
    id = ?
    AND user_game_id = ?;

-- name: CountPlayingPlaythroughs :one
SELECT
    COUNT(*)
FROM
    playthroughs
WHERE
    user_game_id = ?
    AND outcome = 'playing';

-- name: CreatePlaythrough :one
INSERT INTO playthroughs(user_game_id, platform_id, name, started_at, outcome)
    VALUES (?, ?, ?, ?, 'playing')
RETURNING
    *;

-- name: UpdatePlaythroughOutcome :exec
UPDATE
    playthroughs
SET
    outcome = ?,
    finished_at = ?
WHERE
    id = ?;

-- name: UpdatePlaythroughDetails :exec
UPDATE
    playthroughs
SET
    platform_id = ?,
    name = ?,
    started_at = ?,
    finished_at = ?
WHERE
    id = ?;

-- name: DeletePlaythrough :execrows
DELETE FROM playthroughs
WHERE id = ?
    AND user_game_id = ?;

