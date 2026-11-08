-- +goose Up
CREATE TABLE ingredients (
    id UUID PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    protein REAL NOT NULL,
    fat REAL NOT NULL,
    carbohydrates REAL NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS ingredients;