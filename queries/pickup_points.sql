-- name: CreatePickupPoint :one
INSERT INTO pickup_points (name, address)
VALUES (sqlc.arg(name), sqlc.arg(address))
RETURNING *;

-- name: GetPickupPointByID :one
SELECT *
FROM pickup_points
WHERE id = sqlc.arg(id);

-- name: ListPickupPoints :many
SELECT *
FROM pickup_points
ORDER BY created_at DESC, id;

-- NULL означает «поле не передали». Пустые значения раньше отсекает service,
-- а CHECK в таблице остаётся последней защитой от обхода HTTP API.
-- name: UpdatePickupPoint :one
UPDATE pickup_points SET
    name    = COALESCE(sqlc.narg(name), name),
    address = COALESCE(sqlc.narg(address), address)
WHERE id = sqlc.arg(id)
RETURNING *;

-- :execrows позволяет отличить успешное удаление от отсутствующего ПВЗ.
-- Будущий items.pickup_point_id должен ссылаться сюда с ON DELETE RESTRICT:
-- repository уже переводит нарушение этого FK в ErrInUse.
-- name: DeletePickupPoint :execrows
DELETE FROM pickup_points
WHERE id = sqlc.arg(id);
