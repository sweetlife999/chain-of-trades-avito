-- +goose Up

CREATE TYPE support_thread_status AS ENUM ('open', 'in_progress', 'closed');

CREATE TABLE support_threads (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject           text NOT NULL CHECK (char_length(subject) BETWEEN 3 AND 160),
    status            support_thread_status NOT NULL DEFAULT 'open',
    assigned_admin_id uuid REFERENCES users(id) ON DELETE SET NULL,
    user_read_at      timestamptz,
    admin_read_at     timestamptz,
    created_at        timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at        timestamptz NOT NULL DEFAULT clock_timestamp(),
    closed_at         timestamptz,

    CHECK ((status = 'closed') = (closed_at IS NOT NULL))
);

-- Одновременно у пользователя может быть только одно незакрытое обращение. Это не
-- мешает хранить историю и создать новое обращение после закрытия предыдущего.
CREATE UNIQUE INDEX support_threads_one_active_per_user_idx
    ON support_threads (user_id)
    WHERE status <> 'closed';

CREATE INDEX support_threads_queue_idx
    ON support_threads (status, updated_at DESC, id DESC);

CREATE INDEX support_threads_assignee_idx
    ON support_threads (assigned_admin_id, status, updated_at DESC)
    WHERE assigned_admin_id IS NOT NULL;

CREATE TABLE support_messages (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id  uuid NOT NULL REFERENCES support_threads(id) ON DELETE CASCADE,
    author_id  uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    body       text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX support_messages_thread_created_idx
    ON support_messages (thread_id, created_at, id);

-- Уведомление теперь может вести либо в обмен, либо в поддержку. Старые записи
-- остаются валидными: у них заполнен chain_id, новые сообщения поддержки используют
-- support_thread_id и support_message_id.
ALTER TABLE notifications
    ALTER COLUMN chain_id DROP NOT NULL,
    ADD COLUMN support_thread_id uuid REFERENCES support_threads(id) ON DELETE CASCADE,
    ADD COLUMN support_message_id uuid REFERENCES support_messages(id) ON DELETE CASCADE,
    ADD CONSTRAINT notifications_exactly_one_target CHECK (
        num_nonnulls(chain_id, support_thread_id) = 1
    ),
    ADD CONSTRAINT notifications_support_message_target CHECK (
        support_message_id IS NULL OR support_thread_id IS NOT NULL
    );

CREATE UNIQUE INDEX notifications_user_support_message_unique_idx
    ON notifications (user_id, support_message_id)
    WHERE support_message_id IS NOT NULL;

-- Ответ администратора получает владелец обращения. Новое сообщение пользователя
-- получают администраторы, чтобы обращение появилось не только в очереди, но и в
-- общем центре уведомлений.
-- +goose StatementBegin
CREATE FUNCTION notify_support_message() RETURNS trigger AS $$
DECLARE
    author_is_admin boolean;
BEGIN
    SELECT is_admin INTO author_is_admin
    FROM users
    WHERE id = NEW.author_id;

    IF author_is_admin THEN
        INSERT INTO notifications (
            user_id,
            support_thread_id,
            support_message_id,
            kind,
            created_at
        )
        SELECT
            thread.user_id,
            NEW.thread_id,
            NEW.id,
            'support_message',
            NEW.created_at
        FROM support_threads AS thread
        WHERE thread.id = NEW.thread_id
          AND thread.user_id <> NEW.author_id
        ON CONFLICT DO NOTHING;
    ELSE
        INSERT INTO notifications (
            user_id,
            support_thread_id,
            support_message_id,
            kind,
            created_at
        )
        SELECT
            admin.id,
            NEW.thread_id,
            NEW.id,
            'support_message',
            NEW.created_at
        FROM users AS admin
        WHERE admin.is_admin
          AND admin.id <> NEW.author_id
        ON CONFLICT DO NOTHING;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER support_messages_notify_insert
AFTER INSERT ON support_messages
FOR EACH ROW EXECUTE FUNCTION notify_support_message();

-- +goose Down

DROP TRIGGER support_messages_notify_insert ON support_messages;
DROP FUNCTION notify_support_message();

DROP INDEX notifications_user_support_message_unique_idx;

ALTER TABLE notifications
    DROP CONSTRAINT notifications_support_message_target,
    DROP CONSTRAINT notifications_exactly_one_target,
    DROP COLUMN support_message_id,
    DROP COLUMN support_thread_id,
    ALTER COLUMN chain_id SET NOT NULL;

DROP TABLE support_messages;
DROP TABLE support_threads;
DROP TYPE support_thread_status;
