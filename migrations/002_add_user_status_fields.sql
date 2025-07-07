-- +goose Up
-- +goose StatementBegin
-- Create enum type for status
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'pending');

ALTER TABLE users 
ADD COLUMN status user_status DEFAULT 'pending',
ADD COLUMN email_verified BOOLEAN DEFAULT FALSE,
ADD COLUMN email_verified_at TIMESTAMP NULL;

-- Add indexes for status queries
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_email_verified ON users(email_verified);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Remove indexes first
DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_users_email_verified;

-- Remove columns
ALTER TABLE users 
DROP COLUMN IF EXISTS email_verified_at,
DROP COLUMN IF EXISTS email_verified,
DROP COLUMN IF EXISTS status;

-- Drop enum type
DROP TYPE IF EXISTS user_status;
-- +goose StatementEnd