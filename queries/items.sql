-- Категории приходят из API слагами, а не числовыми id: slug — стабильный публичный ключ.
-- Разыменование живёт прямо в запросе, и неизвестный слаг превращается в NULL, который
-- ловит NOT NULL на колонке. Так «такой категории нет» — ошибка БД, а не тихая подстановка.

-- name: InsertItem :one
INSERT INTO items (owner_id, category_id, title, description, photo_urls)
VALUES (
    sqlc.arg(owner_id),
    (SELECT id FROM categories WHERE slug = sqlc.arg(category)),
    sqlc.arg(title),
    sqlc.arg(description),
    sqlc.arg(photo_urls)
)
RETURNING id;

-- LEFT JOIN, а не INNER: неизвестный слаг даёт строку с category_id = NULL и валит вставку.
-- INNER молча выбросил бы её, и пользователь получил бы 201 с потерянным желанием.
-- name: InsertItemWants :exec
INSERT INTO item_wants (item_id, category_id)
SELECT sqlc.arg(item_id)::uuid, c.id
FROM unnest(sqlc.arg(wants)::text[]) AS want(slug)
LEFT JOIN categories c ON c.slug = want.slug;

-- name: DeleteItemWants :exec
DELETE FROM item_wants WHERE item_id = $1;

-- name: ItemHasOpenExchange :one
SELECT EXISTS (
    SELECT 1
    FROM chain_participants AS participant
    JOIN chains AS exchange ON exchange.id = participant.chain_id
    WHERE (participant.gives_item_id = sqlc.arg(item_id)
           OR participant.receives_item_id = sqlc.arg(item_id))
      AND exchange.status IN ('proposed', 'confirmed')
);

-- name: GetItemByID :one
SELECT
    i.*,
    c.slug AS category,
    COALESCE(
        (SELECT array_agg(want_category.slug ORDER BY want_category.slug)
         FROM item_wants w
         JOIN categories want_category ON want_category.id = w.category_id
         WHERE w.item_id = i.id),
        '{}'
    )::text[] AS wants
FROM items i
JOIN categories c ON c.id = i.category_id
WHERE i.id = $1;

-- NULL в аргументе означает «не менять поле» — тот же приём, что в UpdateUserProfile.
-- Категория идёт через CASE, а не COALESCE: COALESCE не отличил бы «не передана»
-- от «передана несуществующая» и во втором случае молча оставил бы старую.
-- name: UpdateItem :exec
UPDATE items SET
    title       = COALESCE(sqlc.narg(title), title),
    description = COALESCE(sqlc.narg(description), description),
    photo_urls  = COALESCE(sqlc.narg(photo_urls)::text[], photo_urls),
    category_id = CASE
        WHEN sqlc.narg(category)::text IS NULL THEN items.category_id
        ELSE (SELECT c.id FROM categories c WHERE c.slug = sqlc.narg(category))
    END
WHERE items.id = sqlc.arg(id);

-- :execrows — ноль удалённых строк отличает «не было такой вещи» (404) от успеха.
-- name: DeleteItem :execrows
DELETE FROM items WHERE id = $1;
