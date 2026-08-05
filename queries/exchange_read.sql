-- Все участники нужны frontend, поэтому пользователя фильтруем через EXISTS,
-- а не через основной JOIN: иначе в ответе осталась бы только его строка.
-- name: ListExchangesByUser :many
SELECT
    exchange.id          AS exchange_id,
    exchange.status      AS exchange_status,
    exchange.created_at  AS exchange_created_at,
    exchange.updated_at  AS exchange_updated_at,
    exchange.closed_at   AS exchange_closed_at,
    participant.user_id,
    participant.position,
    participant.status   AS participant_status,
    participant.decided_at,
    participant.completion_confirmed_at,
    exchange_user.nickname,
    exchange_user.photo_url AS user_photo_url,
    gives_item.id          AS gives_item_id,
    gives_item.title       AS gives_item_title,
    gives_item.description AS gives_item_description,
    gives_item.status      AS gives_item_status,
    gives_category.slug    AS gives_category_slug,
    gives_category.name    AS gives_category_name,
    receives_item.id          AS receives_item_id,
    receives_item.title       AS receives_item_title,
    receives_item.description AS receives_item_description,
    receives_item.status      AS receives_item_status,
    receives_category.slug    AS receives_category_slug,
    receives_category.name    AS receives_category_name
FROM chains AS exchange
JOIN chain_participants AS participant
    ON participant.chain_id = exchange.id
JOIN users AS exchange_user
    ON exchange_user.id = participant.user_id
JOIN items AS gives_item
    ON gives_item.id = participant.gives_item_id
JOIN categories AS gives_category
    ON gives_category.id = gives_item.category_id
JOIN items AS receives_item
    ON receives_item.id = participant.receives_item_id
JOIN categories AS receives_category
    ON receives_category.id = receives_item.category_id
WHERE EXISTS (
    SELECT 1
    FROM chain_participants AS current_participant
    WHERE current_participant.chain_id = exchange.id
      AND current_participant.user_id = sqlc.arg(user_id)
)
ORDER BY exchange.created_at DESC, exchange.id, participant.position;

-- name: GetExchangeByID :many
SELECT
    exchange.id          AS exchange_id,
    exchange.status      AS exchange_status,
    exchange.created_at  AS exchange_created_at,
    exchange.updated_at  AS exchange_updated_at,
    exchange.closed_at   AS exchange_closed_at,
    participant.user_id,
    participant.position,
    participant.status   AS participant_status,
    participant.decided_at,
    participant.completion_confirmed_at,
    exchange_user.nickname,
    exchange_user.photo_url AS user_photo_url,
    gives_item.id          AS gives_item_id,
    gives_item.title       AS gives_item_title,
    gives_item.description AS gives_item_description,
    gives_item.status      AS gives_item_status,
    gives_category.slug    AS gives_category_slug,
    gives_category.name    AS gives_category_name,
    receives_item.id          AS receives_item_id,
    receives_item.title       AS receives_item_title,
    receives_item.description AS receives_item_description,
    receives_item.status      AS receives_item_status,
    receives_category.slug    AS receives_category_slug,
    receives_category.name    AS receives_category_name
FROM chains AS exchange
JOIN chain_participants AS participant
    ON participant.chain_id = exchange.id
JOIN users AS exchange_user
    ON exchange_user.id = participant.user_id
JOIN items AS gives_item
    ON gives_item.id = participant.gives_item_id
JOIN categories AS gives_category
    ON gives_category.id = gives_item.category_id
JOIN items AS receives_item
    ON receives_item.id = participant.receives_item_id
JOIN categories AS receives_category
    ON receives_category.id = receives_item.category_id
WHERE exchange.id = sqlc.arg(exchange_id)
ORDER BY participant.position;
