-- Блокировка и INSERT намеренно разделены. В READ COMMITTED запрос, ожидавший
-- advisory lock, сохранил бы старый snapshot и мог не увидеть только что
-- зафиксированный запрет. Следующий запрос получает свежий snapshot.
-- name: LockExchangeComposition :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(composition_key), 1));

-- Направленная подпись сохраняется для диагностики, а composition_key защищает
-- фактический набор объявлений. Безымянный ON CONFLICT учитывает оба частичных
-- уникальных индекса и атомарно гасит гонку параллельных DFS.
-- name: CreateExchange :one
INSERT INTO chains (signature, composition_key)
SELECT sqlc.arg(signature), sqlc.arg(composition_key)
WHERE NOT EXISTS (
    SELECT 1
    FROM broken_exchange_compositions AS broken
    WHERE broken.composition_key = sqlc.arg(composition_key)
)
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
