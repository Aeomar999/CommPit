-- +goose Up
-- +goose StatementBegin

CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    settings TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE credentials (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    key TEXT NOT NULL,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, key)
);

CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    batch_id TEXT REFERENCES batches(id) ON DELETE SET NULL,
    channel TEXT NOT NULL CHECK (channel IN ('sms', 'email')),
    direction TEXT NOT NULL CHECK (direction IN ('outbound', 'inbound')),
    provider TEXT NOT NULL,
    provider_ref TEXT NOT NULL DEFAULT '',
    from_addr TEXT NOT NULL,
    to_addr TEXT NOT NULL,
    cc TEXT NOT NULL DEFAULT '[]',
    bcc TEXT NOT NULL DEFAULT '[]',
    subject TEXT NOT NULL DEFAULT '',
    body_text TEXT NOT NULL DEFAULT '',
    body_html TEXT NOT NULL DEFAULT '',
    raw_blob_id TEXT REFERENCES blobs(id) ON DELETE SET NULL,
    encoding TEXT NOT NULL DEFAULT '',
    segments INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL CHECK (status IN ('queued', 'sent', 'delivered', 'undelivered', 'failed', 'received')) DEFAULT 'queued',
    error_code TEXT,
    error_message TEXT,
    callback_url TEXT,
    extracted_codes TEXT NOT NULL DEFAULT '[]',
    extracted_links TEXT NOT NULL DEFAULT '[]',
    primary_link TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_project_id ON messages(project_id);
CREATE INDEX idx_messages_batch_id ON messages(batch_id);
CREATE INDEX idx_messages_status ON messages(status);
CREATE INDEX idx_messages_to_addr ON messages(to_addr);
CREATE INDEX idx_messages_from_addr ON messages(from_addr);
CREATE INDEX idx_messages_created_at ON messages(created_at);

CREATE TABLE status_events (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('queued', 'sent', 'delivered', 'undelivered', 'failed', 'received')),
    error_code TEXT,
    at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_status_events_message_id ON status_events(message_id);

CREATE TABLE batches (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    channel TEXT NOT NULL CHECK (channel IN ('sms', 'email')),
    total INTEGER NOT NULL DEFAULT 0,
    counts TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_batches_project_id ON batches(project_id);

CREATE TABLE verifications (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    provider_ref TEXT NOT NULL,
    service_ref TEXT,
    to_addr TEXT NOT NULL,
    channel TEXT NOT NULL CHECK (channel IN ('sms', 'email')),
    code TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'canceled', 'expired', 'max_attempts')) DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    expires_at TIMESTAMP NOT NULL,
    message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_verifications_project_id ON verifications(project_id);
CREATE INDEX idx_verifications_provider_ref ON verifications(project_id, provider_ref);

CREATE TABLE unsubscribes (
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    number TEXT NOT NULL,
    at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (project_id, number)
);

CREATE TABLE attachments (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size INTEGER NOT NULL,
    blob_id TEXT NOT NULL REFERENCES blobs(id) ON DELETE CASCADE,
    inline_cid TEXT
);

CREATE INDEX idx_attachments_message_id ON attachments(message_id);

CREATE TABLE blobs (
    id TEXT PRIMARY KEY,
    data BLOB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE webhook_deliveries (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    message_id TEXT REFERENCES messages(id) ON DELETE SET NULL,
    verification_id TEXT REFERENCES verifications(id) ON DELETE SET NULL,
    kind TEXT NOT NULL,
    url TEXT NOT NULL,
    payload TEXT NOT NULL,
    headers TEXT NOT NULL DEFAULT '{}',
    attempt INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL CHECK (status IN ('pending', 'succeeded', 'failed')) DEFAULT 'pending',
    response_status INTEGER,
    response_body TEXT,
    next_retry_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_webhook_deliveries_project_id ON webhook_deliveries(project_id);
CREATE INDEX idx_webhook_deliveries_next_retry ON webhook_deliveries(next_retry_at) WHERE status = 'pending';

CREATE TABLE request_logs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    adapter TEXT NOT NULL,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    request_headers TEXT NOT NULL DEFAULT '{}',
    request_body BLOB NOT NULL,
    response_status INTEGER NOT NULL,
    response_body BLOB NOT NULL,
    duration_ms INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_request_logs_project_id ON request_logs(project_id);
CREATE INDEX idx_request_logs_created_at ON request_logs(created_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS request_logs;
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS blobs;
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS unsubscribes;
DROP TABLE IF EXISTS verifications;
DROP TABLE IF EXISTS batches;
DROP TABLE IF EXISTS status_events;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS credentials;
DROP TABLE IF EXISTS projects;

-- +goose StatementEnd