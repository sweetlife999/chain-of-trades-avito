-- Рёбра графа не храним: для объявления A соседями считаются доступные объявления B,
-- категория которых входит в список желаемых категорий A.
-- Стабильный порядок нужен, чтобы DFS при одинаковых данных находил одинаковый обмен.
-- name: FindExchangeNeighbors :many
SELECT
    candidate.id,
    candidate.owner_id
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
ORDER BY candidate.created_at, candidate.id;
