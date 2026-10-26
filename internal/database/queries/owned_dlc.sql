-- name: ListOwnedCopyDLCs :many
SELECT
    p.igdb_id AS platform_igdb_id,
    sqlc.embed(g),
    od.dlc_game_id IS NOT NULL AS owned
FROM
    owned_copies oc
    JOIN platforms p ON p.id = oc.platform_id
    JOIN game_platforms gp ON gp.platform_id = oc.platform_id
    JOIN games g ON g.id = gp.game_id
    LEFT JOIN owned_dlc od ON od.owned_copy_id = oc.id
        AND od.dlc_game_id = g.id
WHERE
    oc.user_game_id = ?
    AND g.parent_igdb_id = ?
    AND g.game_type IN (sqlc.slice(game_types))
ORDER BY
    g.released_at ASC NULLS LAST,
    g.title;

