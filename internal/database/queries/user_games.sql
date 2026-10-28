-- name: ListWishlistedUserGames :many
SELECT
    sqlc.embed(ug),
    sqlc.embed(g)
FROM
    user_games ug
    JOIN games g ON ug.game_id = g.id
WHERE
    ug.user_id = ?
    AND ug.wishlisted = TRUE
ORDER BY
    g.released_at ASC NULLS LAST;

-- name: ListLibraryGames :many
SELECT
    sqlc.embed(ug),
    sqlc.embed(g),
    p.name AS platform_name
FROM
    user_games ug
    JOIN games g ON g.id = ug.game_id
    LEFT JOIN owned_copies oc ON oc.user_game_id = ug.id
    LEFT JOIN platforms p ON p.id = oc.platform_id
WHERE
    ug.user_id = sqlc.arg(user_id)
    AND NOT ug.wishlisted
    AND g.game_type NOT IN (sqlc.slice(dlc_game_types));

-- name: ListTrackedDLCs :many
SELECT
    sqlc.embed(ug),
    sqlc.embed(g),
    p.name AS platform_name
FROM
    user_games ug
    JOIN games g ON g.id = ug.game_id
    LEFT JOIN owned_dlc od ON od.dlc_game_id = g.id
        AND od.owned_copy_id IN (
            SELECT
                oc.id
            FROM
                owned_copies oc
            JOIN user_games base ON base.id = oc.user_game_id
        WHERE
            base.user_id = ug.user_id)
    LEFT JOIN owned_copies oc ON oc.id = od.owned_copy_id
    LEFT JOIN platforms p ON p.id = oc.platform_id
WHERE
    ug.user_id = sqlc.arg(user_id)
    AND g.game_type IN (sqlc.slice(game_types));

-- name: GetUserGameByIGDBID :one
SELECT
    sqlc.embed(ug),
    sqlc.embed(g)
FROM
    user_games ug
    JOIN games g ON ug.game_id = g.id
WHERE
    ug.user_id = ?
    AND g.igdb_id = ?;

-- name: CreateWishlistedUserGame :execrows
INSERT INTO user_games(user_id, game_id, status, wishlisted, added_at)
    VALUES (?, ?, NULL, TRUE, ?)
ON CONFLICT (user_id, game_id)
    DO NOTHING;

-- name: UpsertOwnedUserGame :one
INSERT INTO user_games(user_id, game_id, status, wishlisted, added_at)
    VALUES (?, ?, ?, FALSE, ?)
ON CONFLICT (user_id, game_id)
    DO UPDATE SET
        status = COALESCE(user_games.status, excluded.status),
        wishlisted = FALSE
    RETURNING
        *;

-- name: CreateTrackedUserGame :exec
INSERT INTO user_games(user_id, game_id, status, wishlisted, added_at)
    VALUES (?, ?, ?, FALSE, ?)
ON CONFLICT (user_id, game_id)
    DO NOTHING;

-- name: UpdateUserGameStatus :exec
UPDATE
    user_games
SET
    status = ?
WHERE
    id = ?;

-- name: UpdateUserGameDetails :execrows
UPDATE
    user_games
SET
    status = ?,
    rating = ?,
    notes = ?
WHERE
    user_id = ?
    AND game_id =(
        SELECT
            id
        FROM
            games
        WHERE
            igdb_id = ?)
    AND NOT wishlisted;

-- name: DeleteTrackedDLCs :exec
DELETE FROM user_games
WHERE user_id = sqlc.arg(user_id)
    AND game_id IN (
        SELECT
            id
        FROM
            games
        WHERE
            parent_igdb_id = sqlc.arg(parent_igdb_id)
            AND game_type IN (sqlc.slice(dlc_game_types)));

-- name: DeleteUserGame :exec
DELETE FROM user_games
WHERE id = ?;

