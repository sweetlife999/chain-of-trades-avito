-- +goose Up

-- Администратор остаётся обычным пользователем и входит через общий /auth/login.
-- Отдельная таблица admins дублировала бы nickname, пароль и профиль пользователя.
ALTER TABLE users
    ADD COLUMN is_admin boolean NOT NULL DEFAULT false;

-- +goose Down

ALTER TABLE users
    DROP COLUMN is_admin;
