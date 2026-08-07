-- +goose Up

-- Отказ по смыслу — вырезанное ребро графа, а не забаненный состав. Запрет по подписи
-- цикла был неправильной единицей сразу в обе стороны: «A не хочу смартфон B» он
-- запоминал как конкретную тройку людей и переподставлял ту же вещь в цепочке с другим
-- третьим участником, а отказ одного участника убивал состав и для согласных остальных.
-- Здесь хранится ровно намерение: этот пользователь не хочет получать эту вещь.
CREATE TABLE item_refusals (
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id    uuid NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, item_id)
);

-- Обратный индекс, как у user_blocks, не нужен: DFS всегда спрашивает про пару целиком.

-- +goose Down

DROP TABLE item_refusals;
