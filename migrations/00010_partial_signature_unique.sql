-- +goose Up

-- Подпись обязана держать один инвариант: не может быть двух открытых предложений
-- с одним составом. Индекс по всем строкам держал заодно и «не предлагать то, от чего
-- отказались», причём вечно и даже когда никто не отказывался: вытесненная чужим
-- подтверждением цепочка не собиралась больше никогда. Явный отказ теперь хранится
-- отдельно, поэтому индекс сужается до открытых обменов.
DROP INDEX chains_signature_unique_idx;

CREATE UNIQUE INDEX chains_signature_unique_idx
    ON chains (signature) WHERE status IN ('proposed', 'confirmed');

-- +goose Down

-- ponytail: откат падает, если к этому моменту накопились cancelled-дубли, а после Up
-- это штатное состояние. Дедуп подписей, как в 00008, — при первой же нужде.
DROP INDEX chains_signature_unique_idx;

CREATE UNIQUE INDEX chains_signature_unique_idx ON chains (signature);
