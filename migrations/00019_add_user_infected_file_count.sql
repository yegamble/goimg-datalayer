-- +goose Up
ALTER TABLE users ADD COLUMN infected_file_count INTEGER NOT NULL DEFAULT 0;
COMMENT ON COLUMN users.infected_file_count IS 'Count of infected files uploaded by the user';

-- +goose Down
ALTER TABLE users DROP COLUMN infected_file_count;
