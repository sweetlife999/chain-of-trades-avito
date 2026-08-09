-- Снятие объявления использует тот же advisory-lock, что подтверждение обмена и сдача
-- вещи в ПВЗ. Поэтому параллельные операции не могут одновременно увидеть available и
-- принять несовместимые решения.
-- name: LockItemSearchVisibility :exec
SELECT pg_advisory_xact_lock(hashtextextended((sqlc.arg(item_id)::uuid)::text, 0));

-- После advisory-lock блокируется сама строка. Владелец и текущее состояние читаются
-- внутри транзакции, поэтому предварительная проверка HTTP-сервиса не является источником
-- истины и не создаёт TOCTOU-гонку.
-- name: GetItemSearchVisibilityForUpdate :one
SELECT owner_id, status
FROM items
WHERE id = sqlc.arg(item_id)
FOR UPDATE;

-- name: WithdrawAvailableItem :execrows
UPDATE items
SET status = 'withdrawn'
WHERE id = sqlc.arg(item_id)
  AND owner_id = sqlc.arg(owner_id)
  AND status = 'available';

-- name: PublishWithdrawnItem :execrows
UPDATE items
SET status = 'available'
WHERE id = sqlc.arg(item_id)
  AND owner_id = sqlc.arg(owner_id)
  AND status = 'withdrawn';

-- Закрываются только предложения: confirmed/delivering/delivered уже обещали вещь и
-- не могут быть отменены обычным снятием объявления. Возвращаем остальные объявления
-- затронутых цепочек, чтобы после коммита для них заново запланировать поиск.
-- name: CancelProposedExchangesForItemWithdrawal :many
WITH cancelled AS (
    UPDATE chains AS exchange
    SET status = 'cancelled',
        closed_at = now()
    WHERE exchange.status = 'proposed'
      AND EXISTS (
          SELECT 1
          FROM chain_participants AS participant
          WHERE participant.chain_id = exchange.id
            AND participant.gives_item_id = sqlc.arg(item_id)
      )
    RETURNING exchange.id
), events AS (
    INSERT INTO chain_messages (chain_id, kind)
    SELECT cancelled.id, 'exchange_item_withdrawn'
    FROM cancelled
    RETURNING chain_id
)
SELECT DISTINCT
    participant.gives_item_id AS item_id,
    participant.user_id       AS owner_id
FROM cancelled
JOIN events ON events.chain_id = cancelled.id
JOIN chain_participants AS participant ON participant.chain_id = cancelled.id
WHERE participant.gives_item_id <> sqlc.arg(item_id)
ORDER BY participant.gives_item_id;
