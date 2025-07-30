-- +goose Up
-- +goose StatementBegin

-- Add user_id column (nullable, foreign key to users table)
ALTER TABLE push_tokens ADD COLUMN user_id BIGINT;

-- Add deleted_at column for soft deletes
ALTER TABLE push_tokens ADD COLUMN deleted_at TIMESTAMP;

-- Add foreign key constraint to users table
ALTER TABLE push_tokens ADD CONSTRAINT fk_push_tokens_user_id 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

-- Add index on user_id for efficient queries
CREATE INDEX idx_push_tokens_user_id ON push_tokens(user_id);

-- Add index on deleted_at for efficient soft delete queries
CREATE INDEX idx_push_tokens_deleted_at ON push_tokens(deleted_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop indexes
DROP INDEX IF EXISTS idx_push_tokens_deleted_at;
DROP INDEX IF EXISTS idx_push_tokens_user_id;

-- Drop foreign key constraint
ALTER TABLE push_tokens DROP CONSTRAINT IF EXISTS fk_push_tokens_user_id;

-- Drop columns
ALTER TABLE push_tokens DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE push_tokens DROP COLUMN IF EXISTS user_id;

-- +goose StatementEnd