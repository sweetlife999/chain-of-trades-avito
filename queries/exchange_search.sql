-- Рёбра графа не храним: для объявления A соседями считаются доступные объявления B,
-- категория которых входит в список желаемых категорий A и от которых владелец A не
-- отказывался.
-- Стабильный порядок нужен, чтобы DFS при одинаковых данных находил одинаковый обмен.
-- name: FindExchangeNeighbors :many
SELECT
    candidate.id,
    candidate.owner_id,
    candidate.max_chain_length,
    candidate.min_participant_rating,
    candidate.prefer_reliable_participants
FROM items AS current_item
JOIN item_wants AS wanted
    ON wanted.item_id = current_item.id
JOIN items AS candidate
    ON candidate.category_id = wanted.category_id
WHERE current_item.id = sqlc.arg(item_id)
  AND current_item.status = 'available'
  AND candidate.status = 'available'
  AND candidate.id <> current_item.id
  AND candidate.owner_id <> current_item.owner_id
  AND NOT EXISTS (
      SELECT 1
      FROM item_refusals AS refusal
      WHERE refusal.user_id = current_item.owner_id
        AND refusal.item_id = candidate.id
  )
ORDER BY candidate.created_at, candidate.id;

-- Кандидат несовместим с текущим путём, если он заблокировал хотя бы одного
-- уже выбранного владельца или кто-то из них заблокировал кандидата.
-- name: HasUserBlockConflict :one
SELECT EXISTS (
    SELECT 1
    FROM user_blocks AS block
    WHERE (
        block.blocker_id = sqlc.arg(candidate_user_id)
        AND block.blocked_id = ANY(sqlc.arg(path_user_ids)::uuid[])
    ) OR (
        block.blocked_id = sqlc.arg(candidate_user_id)
        AND block.blocker_id = ANY(sqlc.arg(path_user_ids)::uuid[])
    )
);

-- Статистика читается одним запросом для всех владельцев найденных циклов. Пользователь
-- без рейтинга получает нейтральные 3.0, чтобы новый аккаунт не оказался автоматически
-- хуже аккаунта с одной оценкой.
-- name: ListExchangeSearchUserStats :many
SELECT
    id,
    deals_completed,
    deals_broken,
    COALESCE(rating::double precision, 3.0)::double precision AS rating
FROM users
WHERE id = ANY(sqlc.arg(user_ids)::uuid[]);

-- Настройки читаются пачкой для всех вещей найденных циклов. Проверка выполняется
-- после DFS: так учитываются требования каждого участника, а не только объявления,
-- с которого конкретный worker начал обход.
-- name: ListExchangeSearchItemFilters :many
SELECT
    id,
    max_chain_length,
    min_participant_rating,
    prefer_reliable_participants
FROM items
WHERE id = ANY(sqlc.arg(item_ids)::uuid[]);
