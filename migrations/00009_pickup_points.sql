-- +goose Up

CREATE TABLE pickup_points (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 120),
    address    text NOT NULL CHECK (char_length(btrim(address)) BETWEEN 1 AND 500),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER pickup_points_set_updated_at
BEFORE UPDATE ON pickup_points
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down

DROP TABLE pickup_points;
