-- +goose Up

-- Уведомление хранит сам факт доставки события пользователю. Текст не дублируется:
-- kind, автор и контекст обмена читаются из chain_messages, а фразу собирает frontend.
-- message_id = NULL означает единственное уведомление о создании нового предложения.
CREATE TABLE notifications (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chain_id   uuid NOT NULL REFERENCES chains(id) ON DELETE CASCADE,
    message_id uuid REFERENCES chain_messages(id) ON DELETE CASCADE,
    kind       text NOT NULL CHECK (char_length(kind) BETWEEN 1 AND 64),
    read_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);

-- Одно событие треда доставляется конкретному пользователю только один раз, даже если
-- вставку случайно повторит несколько конкурентных процессов.
CREATE UNIQUE INDEX notifications_user_message_unique_idx
    ON notifications (user_id, message_id)
    WHERE message_id IS NOT NULL;

-- На один новый обмен участник получает ровно одно уведомление о предложении.
CREATE UNIQUE INDEX notifications_user_proposal_unique_idx
    ON notifications (user_id, chain_id)
    WHERE message_id IS NULL;

CREATE INDEX notifications_user_created_idx
    ON notifications (user_id, created_at DESC, id DESC);

CREATE INDEX notifications_user_unread_idx
    ON notifications (user_id, created_at DESC, id DESC)
    WHERE read_at IS NULL;

-- Любой способ создать участника обмена автоматически создаёт уведомление о новом
-- предложении. Триггер выполняется внутри исходной транзакции: при rollback обе записи
-- исчезают вместе.
-- +goose StatementBegin
CREATE FUNCTION notify_exchange_participant() RETURNS trigger AS $$
BEGIN
    INSERT INTO notifications (user_id, chain_id, kind)
    VALUES (NEW.user_id, NEW.chain_id, 'exchange_proposed')
    ON CONFLICT DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER chain_participants_notify_insert
AFTER INSERT ON chain_participants
FOR EACH ROW EXECUTE FUNCTION notify_exchange_participant();

-- Текстовые сообщения и системные события проходят через одну таблицу, поэтому один
-- триггер покрывает чат, подтверждение, доставку, отмену и завершение. Автору его же
-- действие не отправляется; системное событие с NULL-автором получают все участники.
-- +goose StatementBegin
CREATE FUNCTION notify_chain_message() RETURNS trigger AS $$
BEGIN
    INSERT INTO notifications (user_id, chain_id, message_id, kind, created_at)
    SELECT
        participant.user_id,
        NEW.chain_id,
        NEW.id,
        NEW.kind::text,
        NEW.created_at
    FROM chain_participants AS participant
    WHERE participant.chain_id = NEW.chain_id
      AND (NEW.author_id IS NULL OR participant.user_id <> NEW.author_id)
    ON CONFLICT DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER chain_messages_notify_insert
AFTER INSERT ON chain_messages
FOR EACH ROW EXECUTE FUNCTION notify_chain_message();

-- +goose Down

DROP TRIGGER chain_messages_notify_insert ON chain_messages;
DROP FUNCTION notify_chain_message();

DROP TRIGGER chain_participants_notify_insert ON chain_participants;
DROP FUNCTION notify_exchange_participant();

DROP TABLE notifications;
