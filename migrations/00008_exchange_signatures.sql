-- +goose Up

-- Подпись описывает точный направленный цикл обмена. Она не зависит от того,
-- с какого объявления DFS начал обход: пары gives_item_id -> receives_item_id
-- сортируются перед объединением.
ALTER TABLE chains
    ADD COLUMN signature text;

-- Старые данные могли уже содержать дубли. Первой записи оставляем каноническую
-- подпись, а ошибочным историческим копиям назначаем служебную уникальную подпись.
-- Благодаря этому миграция не удаляет пользовательскую историю и одновременно
-- запрещает создавать новые копии того же цикла.
WITH base_signatures AS (
    SELECT
        chain.id,
        chain.created_at,
        COALESCE(
            string_agg(
                participant.gives_item_id::text || '>' || participant.receives_item_id::text,
                '|' ORDER BY participant.gives_item_id::text, participant.receives_item_id::text
            ),
            'legacy:' || chain.id::text
        ) AS signature
    FROM chains AS chain
    LEFT JOIN chain_participants AS participant ON participant.chain_id = chain.id
    GROUP BY chain.id, chain.created_at
), ranked_signatures AS (
    SELECT
        id,
        signature,
        row_number() OVER (PARTITION BY signature ORDER BY created_at, id) AS duplicate_number
    FROM base_signatures
)
UPDATE chains AS chain
SET signature = CASE
    WHEN ranked.duplicate_number = 1 THEN ranked.signature
    ELSE ranked.signature || '|legacy:' || ranked.id::text
END
FROM ranked_signatures AS ranked
WHERE chain.id = ranked.id;

ALTER TABLE chains
    ALTER COLUMN signature SET NOT NULL;

CREATE UNIQUE INDEX chains_signature_unique_idx ON chains (signature);

-- +goose Down

DROP INDEX chains_signature_unique_idx;

ALTER TABLE chains
    DROP COLUMN signature;
