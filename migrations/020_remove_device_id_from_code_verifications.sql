-- +goose Up
-- +goose StatementBegin
ALTER TABLE code_verifications DROP COLUMN device_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE code_verifications ADD COLUMN device_id VARCHAR(255);
-- +goose StatementEnd