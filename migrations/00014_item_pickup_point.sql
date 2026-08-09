-- +goose Up

-- Где физически лежит вещь. NULL = дома у владельца. Отдельная колонка, а не таблица
-- «вещь в пункте»: у вещи ровно одно место хранения, история переездов продукту не нужна.
-- ON DELETE RESTRICT предсказан комментарием к DeletePickupPoint в queries/pickup_points.sql
-- и уже обрабатывается: pickuppoint/repository переводит нарушение этого ключа в ErrInUse.
ALTER TABLE items
    ADD COLUMN pickup_point_id uuid REFERENCES pickup_points(id) ON DELETE RESTRICT;

-- Частичный: у большинства вещей колонка пустая, а индекс нужен ровно проверке RESTRICT
-- при удалении пункта — без него она уходит в seq scan по всем вещам.
CREATE INDEX items_pickup_point_id_idx ON items (pickup_point_id)
    WHERE pickup_point_id IS NOT NULL;

-- Подпись держит инвариант «нет двух открытых предложений с одним составом», и открытых
-- статусов стало четыре. Без этого цепь в доставке выпадала бы из-под индекса, и DFS
-- собрал бы тот же состав второй раз, пока первый едет.
DROP INDEX chains_signature_unique_idx;

CREATE UNIQUE INDEX chains_signature_unique_idx
    ON chains (signature) WHERE status IN ('proposed', 'confirmed', 'delivering', 'delivered');

-- +goose Down

DROP INDEX chains_signature_unique_idx;

CREATE UNIQUE INDEX chains_signature_unique_idx
    ON chains (signature) WHERE status IN ('proposed', 'confirmed');

DROP INDEX items_pickup_point_id_idx;

ALTER TABLE items
    DROP COLUMN pickup_point_id;
