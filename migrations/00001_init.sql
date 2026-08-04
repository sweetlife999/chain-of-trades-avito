-- +goose Up

-- Статусы держим ENUM-типами: sqlc генерирует из них типизированные Go-константы,
-- опечатка в статусе ловится компилятором, а не в рантайме.
CREATE TYPE item_status        AS ENUM ('available', 'reserved', 'traded', 'withdrawn');
CREATE TYPE chain_status       AS ENUM ('proposed', 'confirmed', 'completed', 'cancelled');
CREATE TYPE participant_status AS ENUM ('pending', 'accepted', 'declined');

CREATE TABLE users (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    nickname        text NOT NULL CHECK (char_length(nickname) BETWEEN 3 AND 32),
    password_hash   text NOT NULL,
    photo_url       text,
    description     text NOT NULL DEFAULT '',
    deals_completed integer NOT NULL DEFAULT 0 CHECK (deals_completed >= 0),
    deals_broken    integer NOT NULL DEFAULT 0 CHECK (deals_broken >= 0),
    rating          numeric(3,2) CHECK (rating BETWEEN 0 AND 5),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- Логин идёт по nickname, поэтому уникальность регистронезависимая.
-- Индекс по lower() вместо citext: не тянем расширение, sqlc маппит колонку в обычный string.
CREATE UNIQUE INDEX users_nickname_lower_key ON users (lower(nickname));

-- Справочник: строк десяток, публичные, перебирать нечего — uuid тут ничего не даёт.
CREATE TABLE categories (
    id   smallint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug text NOT NULL UNIQUE,
    name text NOT NULL
);

CREATE TABLE items (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    uuid     NOT NULL REFERENCES users(id)      ON DELETE CASCADE,
    category_id smallint NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    title       text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 120),
    description text NOT NULL DEFAULT '',
    photo_url   text,
    status      item_status NOT NULL DEFAULT 'available',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX items_owner_id_idx  ON items (owner_id);
CREATE INDEX items_available_idx ON items (category_id) WHERE status = 'available';

-- Желание привязано к вещи, а не к пользователю: ребро графа обмена получается
-- однозначным (эта вещь -> эта категория), и матчингу не надо гадать, за что её отдают.
-- Несколько строк на одну вещь читаются как ИЛИ.
CREATE TABLE item_wants (
    item_id     uuid     NOT NULL REFERENCES items(id)      ON DELETE CASCADE,
    category_id smallint NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    PRIMARY KEY (item_id, category_id)
);

-- Обратный обход: «кто хочет категорию X» — это половина поиска цикла.
CREATE INDEX item_wants_category_id_idx ON item_wants (category_id);

-- Без статуса 'accepted': согласие — свойство участника, а не цепочки.
-- Дублировать агрегат здесь = гарантированно разъехаться с chain_participants.
CREATE TABLE chains (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    status     chain_status NOT NULL DEFAULT 'proposed',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    closed_at  timestamptz
);

CREATE INDEX chains_status_idx ON chains (status);

CREATE TABLE chain_participants (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chain_id         uuid NOT NULL REFERENCES chains(id) ON DELETE CASCADE,
    -- RESTRICT: удаление пользователя не предусмотрено (issue #4 — CRU без D),
    -- и уж точно не под ногами у живой цепочки.
    user_id          uuid NOT NULL REFERENCES users(id)  ON DELETE RESTRICT,
    gives_item_id    uuid NOT NULL REFERENCES items(id)  ON DELETE RESTRICT,
    receives_item_id uuid NOT NULL REFERENCES items(id)  ON DELETE RESTRICT,
    position         integer NOT NULL CHECK (position >= 0),
    status           participant_status NOT NULL DEFAULT 'pending',
    decided_at       timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (chain_id, user_id),
    UNIQUE (chain_id, position),
    UNIQUE (chain_id, gives_item_id),
    UNIQUE (chain_id, receives_item_id),
    CHECK  (gives_item_id <> receives_item_id)
);

CREATE INDEX chain_participants_user_id_idx       ON chain_participants (user_id);
CREATE INDEX chain_participants_gives_item_id_idx ON chain_participants (gives_item_id);

-- updated_at держим триггером, а не в каждом UPDATE: так его не забудет ни один запрос.
-- +goose StatementBegin
CREATE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER users_set_updated_at  BEFORE UPDATE ON users  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER items_set_updated_at  BEFORE UPDATE ON items  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER chains_set_updated_at BEFORE UPDATE ON chains FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down

DROP TABLE chain_participants;
DROP TABLE item_wants;
DROP TABLE chains;
DROP TABLE items;
DROP TABLE categories;
DROP TABLE users;

DROP FUNCTION set_updated_at();

DROP TYPE participant_status;
DROP TYPE chain_status;
DROP TYPE item_status;
