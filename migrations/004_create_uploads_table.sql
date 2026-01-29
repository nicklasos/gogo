-- +goose Up
-- +goose StatementBegin
CREATE TABLE uploads (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    folder_id INTEGER NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('image', 'video', 'document', 'audio', 'other')),
    relative_path VARCHAR(500) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_uploads_user_id ON uploads(user_id);
CREATE INDEX idx_uploads_folder_id ON uploads(folder_id);
CREATE INDEX idx_uploads_type ON uploads(type);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_uploads_type;
DROP INDEX IF EXISTS idx_uploads_folder_id;
DROP INDEX IF EXISTS idx_uploads_user_id;
DROP TABLE IF EXISTS uploads;
-- +goose StatementEnd
