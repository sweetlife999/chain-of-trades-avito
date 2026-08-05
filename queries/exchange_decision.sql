-- Строка обмена блокируется первой. Поэтому два одновременных решения по одному
-- обмену выполняются последовательно и не могут потерять обновления друг друга.
-- name: LockExchange :one
SELECT status
FROM chains
WHERE id = sqlc.arg(exchange_id)
FOR UPDATE;

-- name: LockExchangeParticipant :one
SELECT status
FROM chain_participants
WHERE chain_id = sqlc.arg(exchange_id)
  AND user_id = sqlc.arg(user_id)
FOR UPDATE;

-- name: AcceptExchangeParticipant :exec
UPDATE chain_participants
SET status = 'accepted',
    decided_at = now()
WHERE chain_id = sqlc.arg(exchange_id)
  AND user_id = sqlc.arg(user_id);

-- name: DeclineExchangeParticipant :exec
UPDATE chain_participants
SET status = 'declined',
    decided_at = now()
WHERE chain_id = sqlc.arg(exchange_id)
  AND user_id = sqlc.arg(user_id);

-- name: CountPendingExchangeParticipants :one
SELECT count(*)
FROM chain_participants
WHERE chain_id = sqlc.arg(exchange_id)
  AND status = 'pending';

-- Вещи блокируются в одинаковом порядке UUID. Это защищает конкурирующие обмены
-- от двойного резервирования и уменьшает риск взаимной блокировки транзакций.
-- name: LockExchangeItems :many
SELECT item.id, item.status
FROM items AS item
JOIN chain_participants AS participant
  ON participant.gives_item_id = item.id
WHERE participant.chain_id = sqlc.arg(exchange_id)
ORDER BY item.id
FOR UPDATE OF item;

-- name: ReserveExchangeItems :execrows
UPDATE items
SET status = 'reserved'
WHERE id IN (
    SELECT gives_item_id
    FROM chain_participants
    WHERE chain_id = sqlc.arg(exchange_id)
)
  AND status = 'available';

-- name: ConfirmExchange :exec
UPDATE chains
SET status = 'confirmed'
WHERE id = sqlc.arg(exchange_id);

-- name: CancelExchange :exec
UPDATE chains
SET status = 'cancelled',
    closed_at = now()
WHERE id = sqlc.arg(exchange_id);
