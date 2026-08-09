-- +goose Up

CREATE TYPE admin_audit_action AS ENUM (
    'report_assigned',
    'report_resolved',
    'report_rejected',
    'report_messages_viewed',
    'user_blocked',
    'user_unblocked',
    'exchange_cancelled'
);

CREATE TYPE admin_audit_target AS ENUM ('report', 'user', 'exchange');

ALTER TABLE users
    ADD COLUMN is_blocked boolean NOT NULL DEFAULT false,
    ADD COLUMN blocked_at timestamptz,
    ADD COLUMN blocked_by uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD CONSTRAINT users_global_block_state_check CHECK (
        (is_blocked AND blocked_at IS NOT NULL)
        OR
        (NOT is_blocked AND blocked_at IS NULL AND blocked_by IS NULL)
    );

-- target_id полиморфный: в зависимости от target_type это reports, users или chains.
-- Поэтому внешний ключ на него намеренно не ставится.
CREATE TABLE admin_audit_log (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id    uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action      admin_audit_action NOT NULL,
    target_type admin_audit_target NOT NULL,
    target_id   uuid NOT NULL,
    metadata    jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX admin_audit_log_created_at_idx
    ON admin_audit_log (created_at DESC, id DESC);
CREATE INDEX admin_audit_log_admin_id_idx
    ON admin_audit_log (admin_id, created_at DESC, id DESC);
CREATE INDEX admin_audit_log_action_idx
    ON admin_audit_log (action, created_at DESC, id DESC);

-- +goose Down

DROP TABLE admin_audit_log;

ALTER TABLE users
    DROP CONSTRAINT users_global_block_state_check,
    DROP COLUMN blocked_by,
    DROP COLUMN blocked_at,
    DROP COLUMN is_blocked;

DROP TYPE admin_audit_target;
DROP TYPE admin_audit_action;
