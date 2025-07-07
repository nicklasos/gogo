-- +goose Up
-- +goose StatementBegin
ALTER TABLE users 
ADD COLUMN status ENUM('active', 'inactive', 'pending') DEFAULT 'pending' AFTER email,
ADD COLUMN email_verified BOOLEAN DEFAULT FALSE AFTER status,
ADD COLUMN email_verified_at TIMESTAMP NULL AFTER email_verified;

-- Add index for status queries
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_email_verified ON users(email_verified);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Remove indexes first
DROP INDEX IF EXISTS idx_users_status ON users;
DROP INDEX IF EXISTS idx_users_email_verified ON users;

-- Remove columns
ALTER TABLE users 
DROP COLUMN IF EXISTS email_verified_at,
DROP COLUMN IF EXISTS email_verified,
DROP COLUMN IF EXISTS status;
-- +goose StatementEnd