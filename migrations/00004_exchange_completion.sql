-- +goose Up

-- Согласие участвовать и подтверждение фактического получения вещи — разные события.
-- Отдельная отметка позволяет frontend показать, кто уже завершил обмен, а транзакции —
-- безопасно определить последнего участника и не начислить статистику повторно.
ALTER TABLE chain_participants
    ADD COLUMN completion_confirmed_at timestamptz;

-- +goose Down

ALTER TABLE chain_participants
    DROP COLUMN completion_confirmed_at;
