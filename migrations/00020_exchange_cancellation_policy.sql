-- +goose Up

-- Статус отвечает на вопрос «закрыт ли обмен», причина — почему именно он закрыт.
-- Отдельное поле не размножает статусы и позволяет frontend объяснить отмену.
CREATE TYPE chain_cancel_reason AS ENUM (
    'proposal_declined',
    'confirmed_broken',
    'superseded',
    'item_withdrawn',
    'user_blocked',
    'admin_cancelled',
    'legacy'
);

ALTER TABLE chains ADD COLUMN cancel_reason chain_cancel_reason;

-- У старых отмен нет надёжно восстановимой причины. Помечаем их явно, после чего
-- constraint гарантирует, что новые отмены без причины в БД не попадут.
UPDATE chains
SET cancel_reason = 'legacy'
WHERE status = 'cancelled';

ALTER TABLE chains ADD CONSTRAINT chains_cancel_reason_matches_status
CHECK (
    (status = 'cancelled' AND cancel_reason IS NOT NULL)
    OR (status <> 'cancelled' AND cancel_reason IS NULL)
);

-- Срыв confirmed — не отказ от отдельной вещи, поэтому item_refusals здесь неверен.
-- Запоминается только точный набор объявлений; состав с заменой хотя бы одной вещи
-- остаётся допустимым.
CREATE TABLE broken_exchange_compositions (
    composition_key text PRIMARY KEY CHECK (composition_key <> ''),
    source_chain_id uuid NOT NULL UNIQUE REFERENCES chains(id),
    created_at      timestamptz NOT NULL DEFAULT now()
);

-- +goose Down

DROP TABLE broken_exchange_compositions;
ALTER TABLE chains DROP CONSTRAINT chains_cancel_reason_matches_status;
ALTER TABLE chains DROP COLUMN cancel_reason;
DROP TYPE chain_cancel_reason;
