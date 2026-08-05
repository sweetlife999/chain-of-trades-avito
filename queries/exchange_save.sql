-- name: CreateExchange :one
INSERT INTO chains DEFAULT VALUES
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
