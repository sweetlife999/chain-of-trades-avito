-- name: LockExchangeCompletionParticipant :one
SELECT status, completion_confirmed_at
FROM chain_participants
WHERE chain_id = sqlc.arg(exchange_id)
  AND user_id = sqlc.arg(user_id)
FOR UPDATE;

-- name: ConfirmExchangeParticipantCompletion :exec
UPDATE chain_participants
SET completion_confirmed_at = now()
WHERE chain_id = sqlc.arg(exchange_id)
  AND user_id = sqlc.arg(user_id);

-- name: CountIncompleteExchangeParticipants :one
SELECT count(*)
FROM chain_participants
WHERE chain_id = sqlc.arg(exchange_id)
  AND completion_confirmed_at IS NULL;

-- name: MarkExchangeItemsTraded :execrows
UPDATE items
SET status = 'traded'
WHERE id IN (
    SELECT gives_item_id
    FROM chain_participants
    WHERE chain_id = sqlc.arg(exchange_id)
)
  AND status = 'reserved';

-- Вещи разъехались по новым владельцам, поэтому пункт хранения обнуляется вместе с
-- переходом в traded: иначе завершённая вещь навсегда осталась бы «лежащей в ПВЗ» и
-- держала бы его от удаления внешним ключом.
-- name: ClearExchangeItemsPickupPoint :exec
UPDATE items
SET pickup_point_id = NULL
WHERE id IN (
    SELECT gives_item_id
    FROM chain_participants
    WHERE chain_id = sqlc.arg(exchange_id)
);

-- name: CompleteExchange :exec
UPDATE chains
SET status = 'completed',
    closed_at = now()
WHERE id = sqlc.arg(exchange_id);

-- name: IncrementExchangeParticipantsDealsCompleted :execrows
UPDATE users
SET deals_completed = deals_completed + 1
WHERE id IN (
    SELECT user_id
    FROM chain_participants
    WHERE chain_id = sqlc.arg(exchange_id)
);
