-- Направленная подпись сохраняется для диагностики, а composition_key защищает
-- фактический набор объявлений. Безымянный ON CONFLICT учитывает оба частичных
-- уникальных индекса и атомарно гасит гонку параллельных DFS.
-- name: CreateExchange :one
INSERT INTO chains (signature, composition_key)
VALUES (sqlc.arg(signature), sqlc.arg(composition_key))
ON CONFLICT DO NOTHING
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
