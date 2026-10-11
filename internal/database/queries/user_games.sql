-- name: UpsertOwnedUserGame :one
INSERT INTO user_games(user_id, game_id, status, wishlisted, added_at)
    VALUES (?, ?, ?, FALSE, ?)
ON CONFLICT (user_id, game_id)
    DO UPDATE SET
        status = COALESCE(user_games.status, excluded.status),
        wishlisted = FALSE
    RETURNING
        *;

