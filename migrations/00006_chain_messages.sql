-- +goose Up

-- Тред обмена держит и переписку участников, и события сделки. Разделять их на две
-- таблицы нечем: и то и другое — строка с текстом, автором и временем, а frontend
-- показывает их одной лентой. Отличает их kind.
CREATE TYPE chain_message_kind AS ENUM (
    'text',
    'participant_accepted',
    'participant_declined',
    'participant_completed',
    'exchange_confirmed',
    'exchange_completed',
    'exchange_superseded'
);

CREATE TABLE chain_messages (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chain_id   uuid NOT NULL REFERENCES chains(id) ON DELETE CASCADE,
    -- NULL = событие принадлежит всему обмену, а не человеку.
    author_id  uuid,
    kind       chain_message_kind NOT NULL DEFAULT 'text',
    -- Текст события не хранится: фразу собирает frontend по kind и автору. Иначе язык
    -- интерфейса пришлось бы вшить в базу, а INSERT — тащить джойн за никнеймом.
    body       text,
    -- clock_timestamp(), а не now(): now() возвращает время начала транзакции, поэтому
    -- события, записанные одной транзакцией (участник согласился -> обмен подтверждён),
    -- получили бы одинаковую метку и встали бы в ленте в случайном порядке.
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),

    -- «Автор пишет только в тот обмен, где участвует» — правило БД, а не сервиса.
    -- Ключ опирается на UNIQUE (chain_id, user_id) из 00001_init. При author_id IS NULL
    -- ключ по MATCH SIMPLE не проверяется, поэтому события обмена проходят свободно.
    -- Отдельный ключ на users не нужен: этот его накрывает.
    FOREIGN KEY (chain_id, author_id)
        REFERENCES chain_participants (chain_id, user_id) ON DELETE CASCADE,

    -- Текст есть ровно у сообщений людей, у событий его нет.
    CHECK ((kind = 'text') = (body IS NOT NULL)),
    CHECK (kind <> 'text' OR author_id IS NOT NULL),
    CHECK (body IS NULL OR char_length(body) BETWEEN 1 AND 2000)
);

-- Тред всегда читается целиком и в хронологическом порядке; id в индексе разводит
-- сообщения, отправленные в одну миллисекунду.
CREATE INDEX chain_messages_chain_created_idx ON chain_messages (chain_id, created_at, id);

-- Отметка «докуда участник дочитал тред». NULL = не открывал ни разу.
ALTER TABLE chain_participants
    ADD COLUMN messages_read_at timestamptz;

-- +goose Down

ALTER TABLE chain_participants
    DROP COLUMN messages_read_at;

DROP TABLE chain_messages;

DROP TYPE chain_message_kind;
