-- Все операции принятия решения сначала блокируют объявления обмена в одном и том
-- же порядке. Advisory-lock не меняет строку, но сериализует два обмена, которые
-- используют хотя бы одно общее объявление. Коллизия хэша безопасна: она только
-- заставит два независимых запроса ненадолго выполняться последовательно.
-- name: LockExchangeDecisionItems :exec
SELECT pg_advisory_xact_lock(hashtextextended(exchange_item.item_id::text, 0))
FROM (
    SELECT participant.gives_item_id AS item_id
    FROM chain_participants AS participant
    WHERE participant.chain_id = sqlc.arg(exchange_id)
    UNION
    SELECT participant.receives_item_id AS item_id
    FROM chain_participants AS participant
    WHERE participant.chain_id = sqlc.arg(exchange_id)
) AS exchange_item
ORDER BY exchange_item.item_id;

-- Строка обмена блокируется первой. Поэтому два одновременных решения по одному
-- обмену выполняются последовательно и не могут потерять обновления друг друга.
-- name: LockExchange :one
SELECT status, signature, composition_key
FROM chains
WHERE id = sqlc.arg(exchange_id)
FOR UPDATE;

-- name: LockExchangeParticipant :one
SELECT status, completion_confirmed_at
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

-- Отказ от предложения — «не хочу эту вещь», поэтому запоминается вырезанным ребром:
-- вещь берётся из самой цепочки, отдельного чтения в Go не нужно. Повторный отказ от
-- той же вещи в другом обмене — нормальный случай, а не ошибка.
-- name: RecordItemRefusal :exec
INSERT INTO item_refusals (user_id, item_id)
SELECT participant.user_id, participant.receives_item_id
FROM chain_participants AS participant
WHERE participant.chain_id = sqlc.arg(exchange_id)
  AND participant.user_id = sqlc.arg(user_id)
ON CONFLICT DO NOTHING;

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
SELECT item.id, item.owner_id, item.status
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

-- Вещь может лежать в нескольких предложенных обменах, но подтверждённый у неё ровно
-- один: подтверждение резервирует её вещи и гасит конкурентов. Поэтому :one, а pgx.ErrNoRows
-- здесь — нормальный ответ «вещь ни в каком подтверждённом обмене не участвует».
-- name: FindConfirmedExchangeForItem :one
SELECT exchange.id
FROM chains AS exchange
JOIN chain_participants AS participant ON participant.chain_id = exchange.id
WHERE participant.gives_item_id = sqlc.arg(item_id)
  AND exchange.status = 'confirmed';

-- Переход в доставку — одно выражение, а не «прочитать вещи в Go и потом записать статус»:
-- иначе проверка гонялась бы с параллельной сдачей вещи соседом. Условие status = 'confirmed'
-- делает вызов идемпотентным, поэтому его безопасно дёргать и из сдачи вещи, и из последнего
-- подтверждения участия. closed_at не трогаем: обмен не закрыт, он едет.
-- name: PromoteExchangeToDelivering :execrows
UPDATE chains
SET status = 'delivering'
WHERE chains.id = sqlc.arg(exchange_id)
  AND chains.status = 'confirmed'
  AND NOT EXISTS (
      SELECT 1
      FROM chain_participants AS participant
      JOIN items AS item ON item.id = participant.gives_item_id
      WHERE participant.chain_id = sqlc.arg(exchange_id)
        AND item.pickup_point_id IS NULL
  );

-- Администратор завершает физическую доставку всей цепочки одной командой. Проверка
-- пунктов продублирована намеренно: status = 'delivering' уже означает, что все вещи
-- сданы, но это условие не даст некорректным старым данным перейти к получению.
-- name: MarkExchangeDelivered :execrows
UPDATE chains
SET status = 'delivered'
WHERE chains.id = sqlc.arg(exchange_id)
  AND chains.status = 'delivering'
  AND NOT EXISTS (
      SELECT 1
      FROM chain_participants AS participant
      JOIN items AS item ON item.id = participant.gives_item_id
      WHERE participant.chain_id = sqlc.arg(exchange_id)
        AND item.pickup_point_id IS NULL
  );

-- После победы одного обмена все ещё открытые предложения с любым общим
-- объявлением больше не выполнимы и сразу закрываются для frontend. Отмена и запись
-- события в тред каждой отменённой цепочки — одно выражение: участник не может
-- увидеть закрытый обмен без объяснения, почему он закрылся.
-- name: CancelCompetingProposedExchanges :execrows
WITH current_items AS (
    SELECT participant.gives_item_id AS item_id
    FROM chain_participants AS participant
    WHERE participant.chain_id = sqlc.arg(exchange_id)
    UNION
    SELECT participant.receives_item_id AS item_id
    FROM chain_participants AS participant
    WHERE participant.chain_id = sqlc.arg(exchange_id)
), cancelled AS (
    UPDATE chains AS competing_exchange
    SET status = 'cancelled',
        closed_at = now()
    WHERE competing_exchange.id <> sqlc.arg(exchange_id)
      AND competing_exchange.status = 'proposed'
      AND EXISTS (
          SELECT 1
          FROM chain_participants AS competing_participant
          JOIN current_items
            ON current_items.item_id = competing_participant.gives_item_id
            OR current_items.item_id = competing_participant.receives_item_id
          WHERE competing_participant.chain_id = competing_exchange.id
      )
    RETURNING competing_exchange.id
)
INSERT INTO chain_messages (chain_id, kind)
SELECT cancelled.id, 'exchange_superseded'
FROM cancelled;

-- name: CancelExchange :exec
UPDATE chains
SET status = 'cancelled',
    closed_at = now()
WHERE id = sqlc.arg(exchange_id);

-- Подтверждённый обмен уже зарезервировал вещи. При его отмене освобождаем только
-- вещи этого обмена и только из reserved: чужое параллельное изменение не затирается.
-- name: ReleaseExchangeItems :execrows
UPDATE items
SET status = 'available'
WHERE id IN (
    SELECT gives_item_id
    FROM chain_participants
    WHERE chain_id = sqlc.arg(exchange_id)
)
  AND status = 'reserved';

-- deals_broken показывает, сколько уже подтверждённых обменов сорвал пользователь.
-- Отказ от ещё не подтверждённого предложения в этот счётчик не входит.
-- name: IncrementUserDealsBroken :execrows
UPDATE users
SET deals_broken = deals_broken + 1
WHERE id = sqlc.arg(user_id);
