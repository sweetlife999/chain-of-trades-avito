-- +goose Up

ALTER TABLE reports
    ADD COLUMN assigned_at timestamptz,
    ADD COLUMN closed_at timestamptz,
    ADD COLUMN resolution_comment text NOT NULL DEFAULT ''
        CHECK (char_length(resolution_comment) <= 2000);

-- Сохраняем совместимость с уже существующими назначенными и закрытыми жалобами.
UPDATE reports
SET assigned_at = created_at
WHERE assignee_id IS NOT NULL;

UPDATE reports
SET closed_at = created_at
WHERE status IN ('resolved', 'rejected');

ALTER TABLE reports
    ADD CONSTRAINT reports_closed_state_check CHECK (
        (status = 'open' AND closed_at IS NULL)
        OR
        (status IN ('resolved', 'rejected') AND closed_at IS NOT NULL)
    );

-- Нужен для фильтра «мои жалобы» и проверки уже назначенной очереди.
CREATE INDEX reports_assignee_id_idx ON reports (assignee_id, created_at, id)
WHERE assignee_id IS NOT NULL;

-- +goose Down

DROP INDEX reports_assignee_id_idx;

ALTER TABLE reports
    DROP CONSTRAINT reports_closed_state_check,
    DROP COLUMN resolution_comment,
    DROP COLUMN closed_at,
    DROP COLUMN assigned_at;
