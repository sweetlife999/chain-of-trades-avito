-- Все участники нужны frontend, поэтому текущего пользователя подмешиваем отдельным
-- JOIN: основной JOIN оставил бы в ответе только его строку. UNIQUE (chain_id, user_id)
-- гарантирует, что этот JOIN даёт ровно одну строку и не размножает участников,
-- а заодно открывает доступ к его отметке о прочтении треда.
-- name: ListExchangesByUser :many
SELECT
    exchange.id          AS exchange_id,
    exchange.status      AS exchange_status,
    exchange.cancel_reason AS exchange_cancel_reason,
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
    gives_pickup.id        AS gives_pickup_point_id,
    gives_pickup.name      AS gives_pickup_point_name,
    gives_pickup.address   AS gives_pickup_point_address,
    receives_item.id          AS receives_item_id,
    receives_item.title       AS receives_item_title,
    receives_item.description AS receives_item_description,
    receives_item.status      AS receives_item_status,
    receives_category.slug    AS receives_category_slug,
    receives_category.name    AS receives_category_name,
    receives_pickup.id        AS receives_pickup_point_id,
    receives_pickup.name      AS receives_pickup_point_name,
    receives_pickup.address   AS receives_pickup_point_address,
    (
        SELECT count(*)
        FROM chain_messages AS unread
        WHERE unread.chain_id = exchange.id
          AND unread.created_at > coalesce(current_participant.messages_read_at, '-infinity')
          -- IS DISTINCT FROM, а не <>: у событий обмена автора нет, и они тоже считаются
          -- непрочитанными — ради них счётчик в основном и нужен.
          AND unread.author_id IS DISTINCT FROM sqlc.arg(user_id)
    ) AS unread_count,
    rated.user_id                                          AS rating_target_user_id,
    (exchange.closed_at + interval '14 days')::timestamptz AS rating_until,
    my_rating.score                                        AS my_rating_score,
    my_rating.comment                                      AS my_rating_comment
FROM chains AS exchange
JOIN chain_participants AS current_participant
    ON current_participant.chain_id = exchange.id
   AND current_participant.user_id = sqlc.arg(user_id)
-- Кого оценивает текущий пользователь: участник, чья отдаваемая вещь пришла к нему.
-- UNIQUE (chain_id, gives_item_id) и UNIQUE (chain_id, rater_id) держат оба JOIN'а
-- однострочными, поэтому участники не размножаются.
LEFT JOIN chain_participants AS rated
    ON rated.chain_id = exchange.id
   AND rated.gives_item_id = current_participant.receives_item_id
LEFT JOIN chain_ratings AS my_rating
    ON my_rating.chain_id = exchange.id
   AND my_rating.rater_id = sqlc.arg(user_id)
JOIN chain_participants AS participant
    ON participant.chain_id = exchange.id
JOIN users AS exchange_user
    ON exchange_user.id = participant.user_id
JOIN items AS gives_item
    ON gives_item.id = participant.gives_item_id
JOIN categories AS gives_category
    ON gives_category.id = gives_item.category_id
-- LEFT: вещь дома пункта не имеет, INNER выбросил бы её участника из ответа целиком.
LEFT JOIN pickup_points AS gives_pickup
    ON gives_pickup.id = gives_item.pickup_point_id
JOIN items AS receives_item
    ON receives_item.id = participant.receives_item_id
JOIN categories AS receives_category
    ON receives_category.id = receives_item.category_id
LEFT JOIN pickup_points AS receives_pickup
    ON receives_pickup.id = receives_item.pickup_point_id
ORDER BY exchange.created_at DESC, exchange.id, participant.position;

-- name: GetExchangeByID :many
SELECT
    exchange.id          AS exchange_id,
    exchange.status      AS exchange_status,
    exchange.cancel_reason AS exchange_cancel_reason,
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
    gives_pickup.id        AS gives_pickup_point_id,
    gives_pickup.name      AS gives_pickup_point_name,
    gives_pickup.address   AS gives_pickup_point_address,
    receives_item.id          AS receives_item_id,
    receives_item.title       AS receives_item_title,
    receives_item.description AS receives_item_description,
    receives_item.status      AS receives_item_status,
    receives_category.slug    AS receives_category_slug,
    receives_category.name    AS receives_category_name,
    receives_pickup.id        AS receives_pickup_point_id,
    receives_pickup.name      AS receives_pickup_point_name,
    receives_pickup.address   AS receives_pickup_point_address,
    (
        SELECT count(*)
        FROM chain_messages AS unread
        WHERE unread.chain_id = exchange.id
          AND unread.created_at > coalesce(current_participant.messages_read_at, '-infinity')
          AND unread.author_id IS DISTINCT FROM sqlc.arg(user_id)
    ) AS unread_count,
    rated.user_id                                          AS rating_target_user_id,
    (exchange.closed_at + interval '14 days')::timestamptz AS rating_until,
    my_rating.score                                        AS my_rating_score,
    my_rating.comment                                      AS my_rating_comment
FROM chains AS exchange
-- LEFT JOIN, в отличие от списка: обмен обязан вернуться и постороннему, иначе сервис
-- ответил бы «не найден» вместо «нельзя смотреть». Счётчик для него всё равно не поедет.
LEFT JOIN chain_participants AS current_participant
    ON current_participant.chain_id = exchange.id
   AND current_participant.user_id = sqlc.arg(user_id)
-- Постороннему current_participant не нашёлся, поэтому и партнёр для оценки не найдётся:
-- обе колонки приедут пустыми, а до ответа дело всё равно не дойдёт — сервис отдаст 403.
LEFT JOIN chain_participants AS rated
    ON rated.chain_id = exchange.id
   AND rated.gives_item_id = current_participant.receives_item_id
LEFT JOIN chain_ratings AS my_rating
    ON my_rating.chain_id = exchange.id
   AND my_rating.rater_id = sqlc.arg(user_id)
JOIN chain_participants AS participant
    ON participant.chain_id = exchange.id
JOIN users AS exchange_user
    ON exchange_user.id = participant.user_id
JOIN items AS gives_item
    ON gives_item.id = participant.gives_item_id
JOIN categories AS gives_category
    ON gives_category.id = gives_item.category_id
-- LEFT: вещь дома пункта не имеет, INNER выбросил бы её участника из ответа целиком.
LEFT JOIN pickup_points AS gives_pickup
    ON gives_pickup.id = gives_item.pickup_point_id
JOIN items AS receives_item
    ON receives_item.id = participant.receives_item_id
JOIN categories AS receives_category
    ON receives_category.id = receives_item.category_id
LEFT JOIN pickup_points AS receives_pickup
    ON receives_pickup.id = receives_item.pickup_point_id
WHERE exchange.id = sqlc.arg(exchange_id)
ORDER BY participant.position;

-- Общий список для администратора: существующая ручка фильтрует его по user_id,
-- очередь доставки — по status. Пустой status означает любой активный этап.
-- name: ListActiveExchangesForAdmin :many
WITH selected_exchanges AS (
    SELECT exchange.id, exchange.created_at
    FROM chains AS exchange
    WHERE exchange.status IN ('proposed', 'confirmed', 'delivering', 'delivered')
      AND (
          sqlc.narg(user_id)::uuid IS NULL
          OR EXISTS (
              SELECT 1
              FROM chain_participants AS selected_participant
              WHERE selected_participant.chain_id = exchange.id
                AND selected_participant.user_id = sqlc.narg(user_id)::uuid
          )
      )
      AND (
          sqlc.arg(exchange_status)::text = ''
          OR exchange.status::text = sqlc.arg(exchange_status)::text
      )
    ORDER BY exchange.created_at DESC, exchange.id
    LIMIT sqlc.arg(page_limit)
    OFFSET sqlc.arg(page_offset)
)
SELECT
    exchange.id          AS exchange_id,
    exchange.status      AS exchange_status,
    exchange.cancel_reason AS exchange_cancel_reason,
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
    gives_pickup.id        AS gives_pickup_point_id,
    gives_pickup.name      AS gives_pickup_point_name,
    gives_pickup.address   AS gives_pickup_point_address,
    receives_item.id          AS receives_item_id,
    receives_item.title       AS receives_item_title,
    receives_item.description AS receives_item_description,
    receives_item.status      AS receives_item_status,
    receives_category.slug    AS receives_category_slug,
    receives_category.name    AS receives_category_name,
    receives_pickup.id        AS receives_pickup_point_id,
    receives_pickup.name      AS receives_pickup_point_name,
    receives_pickup.address   AS receives_pickup_point_address,
    0::bigint AS unread_count,
    -- Оценка — свойство «меня» в обмене, а у административного списка текущего участника
    -- нет. Пустые колонки держат строки трёх запросов одинаковыми: на этом стоит общий
    -- маппер в read.go, и разъехавшиеся колонки ломают сборку, а не ответ.
    NULL::uuid        AS rating_target_user_id,
    NULL::timestamptz AS rating_until,
    NULL::smallint    AS my_rating_score,
    NULL::text        AS my_rating_comment
FROM selected_exchanges AS selected
JOIN chains AS exchange
    ON exchange.id = selected.id
JOIN chain_participants AS participant
    ON participant.chain_id = exchange.id
JOIN users AS exchange_user
    ON exchange_user.id = participant.user_id
JOIN items AS gives_item
    ON gives_item.id = participant.gives_item_id
JOIN categories AS gives_category
    ON gives_category.id = gives_item.category_id
-- LEFT: вещь дома пункта не имеет, INNER выбросил бы её участника из ответа целиком.
LEFT JOIN pickup_points AS gives_pickup
    ON gives_pickup.id = gives_item.pickup_point_id
JOIN items AS receives_item
    ON receives_item.id = participant.receives_item_id
JOIN categories AS receives_category
    ON receives_category.id = receives_item.category_id
LEFT JOIN pickup_points AS receives_pickup
    ON receives_pickup.id = receives_item.pickup_point_id
ORDER BY exchange.created_at DESC, exchange.id, participant.position;

-- name: CountActiveExchangesForAdmin :one
SELECT count(*)
FROM chains AS exchange
WHERE exchange.status IN ('proposed', 'confirmed', 'delivering', 'delivered')
  AND (
      sqlc.narg(user_id)::uuid IS NULL
      OR EXISTS (
          SELECT 1
          FROM chain_participants AS selected_participant
          WHERE selected_participant.chain_id = exchange.id
            AND selected_participant.user_id = sqlc.narg(user_id)::uuid
      )
  )
  AND (
      sqlc.arg(exchange_status)::text = ''
      OR exchange.status::text = sqlc.arg(exchange_status)::text
  );
