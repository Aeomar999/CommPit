-- +goose Up
-- +goose StatementBegin

-- Provider that formatted a webhook delivery (needed to route inbound
-- replies back through the provider's ReplyParser).
ALTER TABLE webhook_deliveries ADD COLUMN provider TEXT NOT NULL DEFAULT '';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE webhook_deliveries DROP COLUMN provider;

-- +goose StatementEnd
