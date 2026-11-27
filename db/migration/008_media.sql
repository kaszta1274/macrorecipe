-- +goose Up
ALTER TABLE users ADD COLUMN avatar_url TEXT;
ALTER TABLE recipes ADD COLUMN image_url TEXT;

-- +goose Down
ALTER TABLE recipes DROP COLUMN IF EXISTS image_url;
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;