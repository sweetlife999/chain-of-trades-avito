-- +goose Up

-- Блокировка направленная: пользователь A может заблокировать B, даже если B
-- не блокировал A. Для поиска обмена такая связь считается несовместимостью
-- в обе стороны — эти пользователи больше не попадут в один новый обмен.
CREATE TABLE user_blocks (
    blocker_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)
);

-- PRIMARY KEY покрывает выборку списка по blocker_id. Обратный индекс нужен
-- DFS для быстрой проверки «кто заблокировал этого кандидата».
CREATE INDEX user_blocks_blocked_id_idx ON user_blocks (blocked_id);

-- +goose Down

DROP TABLE user_blocks;
