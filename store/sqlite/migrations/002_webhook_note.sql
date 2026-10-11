-- +goose Up
-- +goose StatementBegin

-- Operator note on a webhook delivery (e.g. the Docker host rewrite
-- applied at send time). Never sent to the receiver.
ALTER TABLE webhook_deliveries ADD COLUMN note TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE webhook_deliveries DROP COLUMN note;

-- +goose StatementEnd
