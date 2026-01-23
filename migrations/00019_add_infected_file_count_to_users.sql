-- +goose Up
ALTER TABLE users ADD COLUMN infected_file_count INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN infected_file_count;
