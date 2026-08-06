-- Все участники нужны frontend, поэтому текущего пользователя подмешиваем отдельным
-- JOIN: основной JOIN оставил бы в ответе только его строку. UNIQUE (chain_id, user_id)
-- гарантирует, что этот JOIN даёт ровно одну строку и не размножает участников,
-- а заодно открывает доступ к его отметке о прочтении треда.
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
    receives_category.name    AS receives_category_name,
    (
        SELECT count(*)
        FROM chain_messages AS unread
        WHERE unread.chain_id = exchange.id
          AND unread.created_at > coalesce(current_participant.messages_read_at, '-infinity')
          -- IS DISTINCT FROM, а не <>: у событий обмена автора нет, и они тоже считаются
          -- непрочитанными — ради них счётчик в основном и нужен.
          AND unread.author_id IS DISTINCT FROM sqlc.arg(user_id)
    ) AS unread_count
FROM chains AS exchange
JOIN chain_participants AS current_participant
    ON current_participant.chain_id = exchange.id
   AND current_participant.user_id = sqlc.arg(user_id)
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
    receives_category.name    AS receives_category_name,
    (
        SELECT count(*)
        FROM chain_messages AS unread
        WHERE unread.chain_id = exchange.id
          AND unread.created_at > coalesce(current_participant.messages_read_at, '-infinity')
          AND unread.author_id IS DISTINCT FROM sqlc.arg(user_id)
    ) AS unread_count
FROM chains AS exchange
-- LEFT JOIN, в отличие от списка: обмен обязан вернуться и постороннему, иначе сервис
-- ответил бы «не найден» вместо «нельзя смотреть». Счётчик для него всё равно не поедет.
LEFT JOIN chain_participants AS current_participant
    ON current_participant.chain_id = exchange.id
   AND current_participant.user_id = sqlc.arg(user_id)
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
