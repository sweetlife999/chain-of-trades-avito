-- Подпись уникальна только среди открытых обменов, поэтому арбитром выступает
-- частичный индекс. Без повторения его предиката Postgres не выводит индекс как
-- арбитра и падает на ON CONFLICT.
-- name: CreateExchange :one
INSERT INTO chains (signature)
VALUES (sqlc.arg(signature))
ON CONFLICT (signature) WHERE status IN ('proposed', 'confirmed') DO NOTHING
RETURNING id;

-- name: CreateExchangeParticipant :exec
INSERT INTO chain_participants (
    chain_id,
    user_id,
    gives_item_id,
    receives_item_id,
    position
)
VALUES (
    sqlc.arg(chain_id),
    sqlc.arg(user_id),
    sqlc.arg(gives_item_id),
    sqlc.arg(receives_item_id),
    sqlc.arg(position)
);
