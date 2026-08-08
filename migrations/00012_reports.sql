-- +goose Up

CREATE TYPE report_reason AS ENUM ('spam', 'abuse', 'other');
CREATE TYPE report_status AS ENUM ('open', 'resolved', 'rejected');

-- Жалоба ссылается на сообщение, а не на человека: автор и обмен выводятся джойном
-- к chain_messages. Копия автора в строке была бы вторым источником правды, который
-- база не сверит, — тот же отказ, что у chains.status от статусов участников.
CREATE TABLE reports (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id uuid NOT NULL REFERENCES users(id)          ON DELETE CASCADE,
    message_id  uuid NOT NULL REFERENCES chain_messages(id) ON DELETE CASCADE,
    reason      report_reason NOT NULL,
    comment     text NOT NULL DEFAULT '' CHECK (char_length(comment) <= 2000),
    status      report_status NOT NULL DEFAULT 'open',
    -- Кто разбирает. NULL = жалобу ещё никто не взял. SET NULL, а не CASCADE:
    -- уход модератора не должен уносить с собой жалобу.
    assignee_id uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    -- Одна жалоба на сообщение от одного человека. Заменяет rate-limiting, которого
    -- в проекте нет: повтор ловится вставкой, а не проверкой перед ней.
    UNIQUE (reporter_id, message_id)
);

-- Очередь модерации: открытые, от старых к новым. Закрытым в очереди делать нечего,
-- поэтому индекс частичный.
CREATE INDEX reports_open_idx ON reports (created_at, id) WHERE status = 'open';

-- «Сколько жалоб на это сообщение»: UNIQUE начинается с reporter_id и это не покрывает.
-- Индекса по assignee_id нет намеренно: «мои жалобы» понадобятся вместе с чтением,
-- которого пока нет.
CREATE INDEX reports_message_id_idx ON reports (message_id);

-- +goose Down

DROP TABLE reports;

DROP TYPE report_status;
DROP TYPE report_reason;
