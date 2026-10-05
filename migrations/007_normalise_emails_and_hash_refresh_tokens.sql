-- +goose Up
-- +goose StatementBegin

-- Emails are compared case-insensitively from now on. Rows are lower-cased unless that
-- would collide with another account; the unique index below then fails on those rows,
-- which is the point: two accounts that differ only by case have to be resolved by hand.
UPDATE users u
SET email = LOWER(u.email)
WHERE u.email <> LOWER(u.email)
  AND NOT EXISTS (
      SELECT 1 FROM users other
      WHERE other.id <> u.id AND LOWER(other.email) = LOWER(u.email)
  );

DROP INDEX IF EXISTS idx_users_email_lower;
CREATE UNIQUE INDEX idx_users_email_lower ON users (LOWER(email));

-- Refresh tokens are stored as SHA-256 hashes, like emailed tokens. Hashing the existing
-- rows in place keeps everyone signed in.
UPDATE refresh_tokens SET token = encode(sha256(convert_to(token, 'UTF8')), 'hex');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Hashing cannot be undone: existing sessions have to sign in again after a rollback.
DELETE FROM refresh_tokens;
DROP INDEX IF EXISTS idx_users_email_lower;
CREATE INDEX idx_users_email_lower ON users (LOWER(email));
-- +goose StatementEnd
