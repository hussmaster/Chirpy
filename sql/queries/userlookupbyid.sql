-- name: UserLookupByID :one
SELECT * FROM users
WHERE id = $1;