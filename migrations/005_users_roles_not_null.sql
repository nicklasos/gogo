-- +goose Up
-- +goose StatementBegin
UPDATE users SET roles = ARRAY['user'] WHERE roles IS NULL;
ALTER TABLE users ALTER COLUMN roles SET NOT NULL;
CREATE INDEX idx_users_roles ON users USING GIN (roles);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_users_roles;
ALTER TABLE users ALTER COLUMN roles DROP NOT NULL;
-- +goose StatementEnd
