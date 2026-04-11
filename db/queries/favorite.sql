-- db/queries/favorite.sql

-- name: FindFavoritesByUserID :many
SELECT l.*
FROM user_favorites uf
JOIN locations l ON uf.location_id = l.id
WHERE uf.user_id = $1
ORDER BY uf.created_at DESC;

-- name: AddFavorite :exec
INSERT INTO user_favorites (id, user_id, location_id)
VALUES ($1, $2, $3);

-- name: DeleteFavorite :exec
DELETE FROM user_favorites
WHERE user_id = $1 AND location_id = $2;

-- name: FavoriteExists :one
SELECT EXISTS(
    SELECT 1 FROM user_favorites
    WHERE user_id = $1 AND location_id = $2
) AS "exists";
