-- +goose Up

ALTER TABLE items
    ADD COLUMN max_chain_length integer NOT NULL DEFAULT 5
        CHECK (max_chain_length BETWEEN 2 AND 5),
    ADD COLUMN min_participant_rating double precision NOT NULL DEFAULT 0
        CHECK (min_participant_rating BETWEEN 0 AND 5),
    -- true сохраняет существующее ранжирование для старых объявлений.
    ADD COLUMN prefer_reliable_participants boolean NOT NULL DEFAULT true;

-- +goose Down

ALTER TABLE items
    DROP COLUMN prefer_reliable_participants,
    DROP COLUMN min_participant_rating,
    DROP COLUMN max_chain_length;
