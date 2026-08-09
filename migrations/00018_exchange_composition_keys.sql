-- +goose Up

-- Signature describes directed transfers and therefore changes when the same
-- items are found from another traversal direction. composition_key contains
-- only sorted item IDs and is stable for every permutation of the same set.
ALTER TABLE chains
    ADD COLUMN composition_key text;

UPDATE chains AS exchange
SET composition_key = composition.key
FROM (
    SELECT participant.chain_id,
           string_agg(participant.gives_item_id::text, '|' ORDER BY participant.gives_item_id::text) AS key
    FROM chain_participants AS participant
    GROUP BY participant.chain_id
) AS composition
WHERE exchange.id = composition.chain_id;

-- Orphan legacy rows are invalid exchanges but should not make deployment fail.
-- Their synthetic keys can never collide with keys made from UUID item IDs.
UPDATE chains
SET composition_key = 'legacy:' || id::text
WHERE composition_key IS NULL;

ALTER TABLE chains
    ALTER COLUMN composition_key SET NOT NULL;

-- If old active duplicates exist, index creation deliberately stops migration
-- instead of silently cancelling a real exchange. Such rows must be reviewed.
CREATE UNIQUE INDEX chains_active_composition_unique_idx
    ON chains (composition_key)
    WHERE status IN ('proposed', 'confirmed', 'delivering', 'delivered');

-- +goose Down

DROP INDEX chains_active_composition_unique_idx;

ALTER TABLE chains
    DROP COLUMN composition_key;
