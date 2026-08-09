-- +goose Up

ALTER TYPE admin_audit_action ADD VALUE IF NOT EXISTS 'exchange_delivered';

-- +goose Down

-- PostgreSQL не умеет безопасно удалять отдельное значение ENUM без пересоздания типа.
-- При откате приложения лишнее неиспользуемое значение не влияет на старый код.
