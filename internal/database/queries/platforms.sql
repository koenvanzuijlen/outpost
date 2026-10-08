-- name: UpsertPlatform :one
INSERT INTO platforms(igdb_id, fetched_at, name)
    VALUES (?, ?, ?)
ON CONFLICT (igdb_id)
    DO UPDATE SET
        fetched_at = excluded.fetched_at,
        name = excluded.name
    RETURNING
        *;

