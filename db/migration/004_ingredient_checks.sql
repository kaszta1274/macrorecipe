-- +goose Up
ALTER TABLE ingredients
    ADD CONSTRAINT check_protein_non_negative CHECK (protein >= 0),
    ADD CONSTRAINT check_fat_non_negative CHECK (fat >= 0),
    ADD CONSTRAINT check_carbohydrates_non_negative CHECK (carbohydrates >= 0);

-- +goose Down
ALTER TABLE ingredients
    DROP CONSTRAINT check_protein_non_negative,
    DROP CONSTRAINT check_fat_non_negative,
    DROP CONSTRAINT check_carbohydrates_non_negative;