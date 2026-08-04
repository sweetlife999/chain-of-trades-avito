-- name: CreateUser :one
INSERT INTO users (nickname, password_hash, photo_url, description)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- Логин: nickname уникален регистронезависимо, запрос обязан идти через lower(),
-- иначе не попадёт в users_nickname_lower_key.
-- name: GetUserByNickname :one
SELECT * FROM users WHERE lower(nickname) = lower(sqlc.arg(nickname));

-- TODO: NULL в аргументе означает «не менять поле», поэтому сбросить photo_url обратно
-- в NULL этим запросом нельзя. Понадобится отдельный ClearUserPhoto.
-- name: UpdateUserProfile :one
UPDATE users SET
    nickname    = COALESCE(sqlc.narg(nickname), nickname),
    photo_url   = COALESCE(sqlc.narg(photo_url), photo_url),
    description = COALESCE(sqlc.narg(description), description)
WHERE id = sqlc.arg(id)
RETURNING *;
