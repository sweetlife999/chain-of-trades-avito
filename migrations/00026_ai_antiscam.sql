-- +goose Up

CREATE TYPE antiscam_analysis_status AS ENUM ('pending', 'processing', 'completed', 'failed');
CREATE TYPE antiscam_case_status AS ENUM ('open', 'resolved', 'dismissed');
CREATE TYPE antiscam_category AS ENUM (
    'credentials',
    'external_payment',
    'external_contact',
    'phishing',
    'pressure',
    'other'
);
CREATE TYPE antiscam_decision AS ENUM ('confirmed', 'false_positive');

-- Надёжная очередь фоновой проверки. Одна строка на сообщение не даёт поставить
-- одну и ту же работу дважды, а locked_at позволяет подобрать её после падения worker.
CREATE TABLE antiscam_analyses (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id       uuid NOT NULL UNIQUE REFERENCES chain_messages(id) ON DELETE CASCADE,
    status           antiscam_analysis_status NOT NULL DEFAULT 'pending',
    attempts         integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    available_at     timestamptz NOT NULL DEFAULT now(),
    locked_at        timestamptz,
    rule_score       integer NOT NULL DEFAULT 0 CHECK (rule_score BETWEEN 0 AND 100),
    rule_hits        text[] NOT NULL DEFAULT '{}',
    model_suspicious boolean,
    model_severity   text CHECK (model_severity IN ('low', 'medium', 'high')),
    category         antiscam_category,
    reason           text NOT NULL DEFAULT '' CHECK (char_length(reason) <= 1000),
    evidence         text[] NOT NULL DEFAULT '{}',
    risk             integer NOT NULL DEFAULT 0 CHECK (risk BETWEEN 0 AND 100),
    model_name       text NOT NULL DEFAULT '',
    prompt_version   text NOT NULL DEFAULT '',
    last_error       text NOT NULL DEFAULT '' CHECK (char_length(last_error) <= 2000),
    created_at       timestamptz NOT NULL DEFAULT now(),
    processed_at     timestamptz
);

CREATE INDEX antiscam_analyses_queue_idx
    ON antiscam_analyses (available_at, created_at)
    WHERE status IN ('pending', 'failed');

-- Кейс — карточка для администратора. Несколько сообщений одного человека в
-- одном обмене группируются в одну открытую карточку и не заспамливают очередь.
CREATE TABLE antiscam_cases (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chain_id           uuid NOT NULL REFERENCES chains(id) ON DELETE CASCADE,
    suspect_user_id    uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status             antiscam_case_status NOT NULL DEFAULT 'open',
    risk               integer NOT NULL CHECK (risk BETWEEN 0 AND 100),
    category           antiscam_category NOT NULL,
    reason             text NOT NULL CHECK (char_length(reason) <= 1000),
    reviewed_by        uuid REFERENCES users(id) ON DELETE SET NULL,
    decision           antiscam_decision,
    resolution_comment text NOT NULL DEFAULT '' CHECK (char_length(resolution_comment) <= 2000),
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    closed_at          timestamptz,
    CHECK (
        (status = 'open' AND decision IS NULL AND closed_at IS NULL)
        OR
        (status <> 'open' AND decision IS NOT NULL AND closed_at IS NOT NULL)
    )
);

CREATE UNIQUE INDEX antiscam_cases_one_open_per_user_exchange_idx
    ON antiscam_cases (chain_id, suspect_user_id)
    WHERE status = 'open';
CREATE INDEX antiscam_cases_queue_idx
    ON antiscam_cases (status, risk DESC, updated_at DESC, id DESC);

CREATE TABLE antiscam_case_messages (
    case_id    uuid NOT NULL REFERENCES antiscam_cases(id) ON DELETE CASCADE,
    message_id uuid NOT NULL REFERENCES chain_messages(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (case_id, message_id)
);

CREATE INDEX antiscam_case_messages_message_idx ON antiscam_case_messages (message_id);

CREATE TRIGGER antiscam_cases_set_updated_at
    BEFORE UPDATE ON antiscam_cases
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down

DROP TRIGGER antiscam_cases_set_updated_at ON antiscam_cases;
DROP TABLE antiscam_case_messages;
DROP TABLE antiscam_cases;
DROP TABLE antiscam_analyses;
DROP TYPE antiscam_decision;
DROP TYPE antiscam_category;
DROP TYPE antiscam_case_status;
DROP TYPE antiscam_analysis_status;
