-- +goose Up
-- +goose StatementBegin
CREATE TABLE examples (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_examples_user_id ON examples(user_id);
CREATE INDEX idx_examples_created_at ON examples(created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_examples_created_at;
DROP INDEX IF EXISTS idx_examples_user_id;
DROP TABLE IF EXISTS examples;
-- +goose StatementEnd
