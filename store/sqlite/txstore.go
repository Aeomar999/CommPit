package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"

	"github.com/Aeomar999/CommPit/core"
)

func (t *txStore) CreateProject(ctx context.Context, p *core.Project) error {
	settings, _ := json.Marshal(p.Settings)
	_, err := t.tx.ExecContext(ctx,
		`INSERT INTO projects (id, name, settings, created_at) VALUES (?, ?, ?, ?)`,
		p.ID, p.Name, string(settings), p.CreatedAt)
	return err
}

func (t *txStore) GetProject(ctx context.Context, id string) (*core.Project, error) {
	row := t.tx.QueryRowContext(ctx,
		`SELECT id, name, settings, created_at FROM projects WHERE id = ?`, id)
	var p core.Project
	var settings string
	if err := row.Scan(&p.ID, &p.Name, &settings, &p.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.NewInternal("project not found")
		}
		return nil, err
	}
	json.Unmarshal([]byte(settings), &p.Settings)
	return &p, nil
}

func (t *txStore) UpdateProject(ctx context.Context, p *core.Project) error {
	settings, _ := json.Marshal(p.Settings)
	_, err := t.tx.ExecContext(ctx,
		`UPDATE projects SET name = ?, settings = ? WHERE id = ?`,
		p.Name, string(settings), p.ID)
	return err
}

func (t *txStore) ListProjects(ctx context.Context, limit int, cursor string) ([]*core.Project, string, error) {
	return t.base.ListProjects(ctx, limit, cursor)
}

func (t *txStore) DeleteProject(ctx context.Context, id string) error {
	_, err := t.tx.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	return err
}

func (t *txStore) CreateCredential(ctx context.Context, c *core.Credential) error {
	_, err := t.tx.ExecContext(ctx,
		`INSERT INTO credentials (id, provider, key, project_id, created_at) VALUES (?, ?, ?, ?, ?)`,
		c.ID, c.Provider, c.Key, c.ProjectID, c.CreatedAt)
	return err
}

func (t *txStore) GetCredential(ctx context.Context, provider, key string) (*core.Credential, error) {
	row := t.tx.QueryRowContext(ctx,
		`SELECT id, provider, key, project_id, created_at FROM credentials WHERE provider = ? AND key = ?`,
		provider, key)
	var c core.Credential
	if err := row.Scan(&c.ID, &c.Provider, &c.Key, &c.ProjectID, &c.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.NewInternal("credential not found")
		}
		return nil, err
	}
	return &c, nil
}

func (t *txStore) ListCredentials(ctx context.Context, projectID string) ([]*core.Credential, error) {
	return t.base.ListCredentials(ctx, projectID)
}

func (t *txStore) DeleteCredential(ctx context.Context, id string) error {
	_, err := t.tx.ExecContext(ctx, `DELETE FROM credentials WHERE id = ?`, id)
	return err
}

func (t *txStore) CreateMessage(ctx context.Context, m *core.Message) error {
	cc, _ := json.Marshal(m.CC)
	bcc, _ := json.Marshal(m.BCC)
	extractedCodes, _ := json.Marshal(m.ExtractedCodes)
	extractedLinks, _ := json.Marshal(m.ExtractedLinks)

	var batchID, rawBlobID, primaryLink, errorCode, errorMessage, callbackURL interface{}
	if m.BatchID != nil {
		batchID = *m.BatchID
	}
	if m.RawBlobID != nil {
		rawBlobID = *m.RawBlobID
	}
	if m.PrimaryLink != nil {
		primaryLink = *m.PrimaryLink
	}
	if m.ErrorCode != nil {
		errorCode = *m.ErrorCode
	}
	if m.ErrorMessage != nil {
		errorMessage = *m.ErrorMessage
	}
	if m.CallbackURL != nil {
		callbackURL = *m.CallbackURL
	}

	_, err := t.tx.ExecContext(ctx,
		`INSERT INTO messages (id, project_id, batch_id, channel, direction, provider, provider_ref, from_addr, to_addr, cc, bcc, subject, body_text, body_html, raw_blob_id, encoding, segments, status, error_code, error_message, callback_url, extracted_codes, extracted_links, primary_link, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.ProjectID, batchID, m.Channel, m.Direction, m.Provider, m.ProviderRef,
		m.From, m.To, string(cc), string(bcc), m.Subject, m.BodyText, m.BodyHTML, rawBlobID,
		m.Encoding, m.Segments, m.Status, errorCode, errorMessage, callbackURL,
		string(extractedCodes), string(extractedLinks), primaryLink, m.CreatedAt, m.UpdatedAt)
	return err
}

func (t *txStore) GetMessage(ctx context.Context, projectID, messageID string) (*core.Message, error) {
	return t.base.GetMessage(ctx, projectID, messageID)
}

func (t *txStore) UpdateMessage(ctx context.Context, m *core.Message) error {
	cc, _ := json.Marshal(m.CC)
	bcc, _ := json.Marshal(m.BCC)
	extractedCodes, _ := json.Marshal(m.ExtractedCodes)
	extractedLinks, _ := json.Marshal(m.ExtractedLinks)

	var batchID, rawBlobID, primaryLink, errorCode, errorMessage, callbackURL interface{}
	if m.BatchID != nil {
		batchID = *m.BatchID
	}
	if m.RawBlobID != nil {
		rawBlobID = *m.RawBlobID
	}
	if m.PrimaryLink != nil {
		primaryLink = *m.PrimaryLink
	}
	if m.ErrorCode != nil {
		errorCode = *m.ErrorCode
	}
	if m.ErrorMessage != nil {
		errorMessage = *m.ErrorMessage
	}
	if m.CallbackURL != nil {
		callbackURL = *m.CallbackURL
	}

	_, err := t.tx.ExecContext(ctx,
		`UPDATE messages SET batch_id = ?, channel = ?, direction = ?, provider = ?, provider_ref = ?, from_addr = ?, to_addr = ?, cc = ?, bcc = ?, subject = ?, body_text = ?, body_html = ?, raw_blob_id = ?, encoding = ?, segments = ?, status = ?, error_code = ?, error_message = ?, callback_url = ?, extracted_codes = ?, extracted_links = ?, primary_link = ?, updated_at = ?
		 WHERE id = ? AND project_id = ?`,
		batchID, m.Channel, m.Direction, m.Provider, m.ProviderRef, m.From, m.To, string(cc), string(bcc), m.Subject, m.BodyText, m.BodyHTML, rawBlobID,
		m.Encoding, m.Segments, m.Status, errorCode, errorMessage, callbackURL,
		string(extractedCodes), string(extractedLinks), primaryLink, m.UpdatedAt, m.ID, m.ProjectID)
	return err
}

func (t *txStore) ListMessages(ctx context.Context, projectID string, filter core.MessageFilter) ([]*core.Message, string, error) {
	return t.base.ListMessages(ctx, projectID, filter)
}

func (t *txStore) DeleteMessages(ctx context.Context, projectID string) error {
	_, err := t.tx.ExecContext(ctx, `DELETE FROM messages WHERE project_id = ?`, projectID)
	return err
}

func (t *txStore) DeleteMessage(ctx context.Context, projectID, messageID string) error {
	_, err := t.tx.ExecContext(ctx, `DELETE FROM messages WHERE project_id = ? AND id = ?`, projectID, messageID)
	return err
}

func (t *txStore) ListInFlightMessages(ctx context.Context) ([]*core.Message, error) {
	rows, err := t.tx.QueryContext(ctx,
		`SELECT id, project_id, batch_id, channel, direction, provider, provider_ref, from_addr, to_addr, cc, bcc, subject, body_text, body_html, raw_blob_id, encoding, segments, status, error_code, error_message, callback_url, extracted_codes, extracted_links, primary_link, created_at, updated_at
		 FROM messages WHERE status IN ('queued', 'sent') ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*core.Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func (t *txStore) CreateStatusEvent(ctx context.Context, e *core.StatusEvent) error {
	_, err := t.tx.ExecContext(ctx,
		`INSERT INTO status_events (id, message_id, status, error_code, at) VALUES (?, ?, ?, ?, ?)`,
		e.ID, e.MessageID, e.Status, e.ErrorCode, e.At)
	return err
}

func (t *txStore) GetStatusEvents(ctx context.Context, messageID string) ([]*core.StatusEvent, error) {
	return t.base.GetStatusEvents(ctx, messageID)
}

func (t *txStore) CreateBatch(ctx context.Context, b *core.Batch) error {
	counts, _ := json.Marshal(b.Counts)
	_, err := t.tx.ExecContext(ctx,
		`INSERT INTO batches (id, project_id, provider, channel, total, counts, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		b.ID, b.ProjectID, b.Provider, b.Channel, b.Total, string(counts), b.CreatedAt)
	return err
}

func (t *txStore) GetBatch(ctx context.Context, projectID, batchID string) (*core.Batch, error) {
	return t.base.GetBatch(ctx, projectID, batchID)
}

func (t *txStore) UpdateBatch(ctx context.Context, b *core.Batch) error {
	counts, _ := json.Marshal(b.Counts)
	_, err := t.tx.ExecContext(ctx,
		`UPDATE batches SET total = ?, counts = ? WHERE id = ? AND project_id = ?`,
		b.Total, string(counts), b.ID, b.ProjectID)
	return err
}

func (t *txStore) ListBatches(ctx context.Context, projectID string, limit int, cursor string) ([]*core.Batch, string, error) {
	return t.base.ListBatches(ctx, projectID, limit, cursor)
}

func (t *txStore) RecomputeBatchCounts(ctx context.Context, projectID, batchID string) (*core.Batch, error) {
	rows, err := t.tx.QueryContext(ctx,
		`SELECT status, COUNT(*) FROM messages WHERE project_id = ? AND batch_id = ? GROUP BY status`,
		projectID, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	total := 0
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
		total += count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	batch, err := t.GetBatch(ctx, projectID, batchID)
	if err != nil {
		return nil, err
	}
	batch.Counts = counts
	batch.Total = total
	if err := t.UpdateBatch(ctx, batch); err != nil {
		return nil, err
	}
	return batch, nil
}

func (t *txStore) CreateVerification(ctx context.Context, v *core.Verification) error {
	var serviceRef interface{}
	if v.ServiceRef != nil {
		serviceRef = *v.ServiceRef
	}
	_, err := t.tx.ExecContext(ctx,
		`INSERT INTO verifications (id, project_id, provider, provider_ref, service_ref, to_addr, channel, code, status, attempts, max_attempts, expires_at, message_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.ProjectID, v.Provider, v.ProviderRef, serviceRef, v.To, v.Channel, v.Code, v.Status, v.Attempts, v.MaxAttempts, v.ExpiresAt, v.MessageID, v.CreatedAt)
	return err
}

func (t *txStore) GetVerification(ctx context.Context, projectID, id string) (*core.Verification, error) {
	return t.base.GetVerification(ctx, projectID, id)
}

func (t *txStore) GetVerificationByProviderRef(ctx context.Context, projectID, ref string) (*core.Verification, error) {
	return t.base.GetVerificationByProviderRef(ctx, projectID, ref)
}

func (t *txStore) UpdateVerification(ctx context.Context, v *core.Verification) error {
	var serviceRef interface{}
	if v.ServiceRef != nil {
		serviceRef = *v.ServiceRef
	}
	_, err := t.tx.ExecContext(ctx,
		`UPDATE verifications SET provider = ?, provider_ref = ?, service_ref = ?, to_addr = ?, channel = ?, code = ?, status = ?, attempts = ?, max_attempts = ?, expires_at = ?, message_id = ? WHERE id = ? AND project_id = ?`,
		v.Provider, v.ProviderRef, serviceRef, v.To, v.Channel, v.Code, v.Status, v.Attempts, v.MaxAttempts, v.ExpiresAt, v.MessageID, v.ID, v.ProjectID)
	return err
}

func (t *txStore) ListVerifications(ctx context.Context, projectID string, limit int, cursor string) ([]*core.Verification, string, error) {
	return t.base.ListVerifications(ctx, projectID, limit, cursor)
}

func (t *txStore) CreateUnsubscribe(ctx context.Context, u *core.Unsubscribe) error {
	_, err := t.tx.ExecContext(ctx,
		`INSERT OR REPLACE INTO unsubscribes (project_id, number, at) VALUES (?, ?, ?)`,
		u.ProjectID, u.Number, u.At)
	return err
}

func (t *txStore) DeleteUnsubscribe(ctx context.Context, projectID, number string) error {
	_, err := t.tx.ExecContext(ctx,
		`DELETE FROM unsubscribes WHERE project_id = ? AND number = ?`, projectID, number)
	return err
}

func (t *txStore) IsUnsubscribed(ctx context.Context, projectID, number string) (bool, error) {
	return t.base.IsUnsubscribed(ctx, projectID, number)
}

func (t *txStore) CreateAttachment(ctx context.Context, a *core.Attachment) error {
	_, err := t.tx.ExecContext(ctx,
		`INSERT INTO attachments (id, message_id, filename, content_type, size, blob_id, inline_cid) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.MessageID, a.Filename, a.ContentType, a.Size, a.BlobID, a.InlineCID)
	return err
}

func (t *txStore) GetAttachment(ctx context.Context, id string) (*core.Attachment, error) {
	return t.base.GetAttachment(ctx, id)
}

func (t *txStore) ListAttachments(ctx context.Context, messageID string) ([]*core.Attachment, error) {
	return t.base.ListAttachments(ctx, messageID)
}

func (t *txStore) CreateWebhookDelivery(ctx context.Context, w *core.WebhookDelivery) error {
	payload, _ := json.Marshal(w.Payload)
	headers, _ := json.Marshal(w.Headers)

	var messageID, verificationID interface{}
	if w.MessageID != nil {
		messageID = *w.MessageID
	}
	if w.VerificationID != nil {
		verificationID = *w.VerificationID
	}
	var nextRetryAt interface{}
	if w.NextRetryAt != nil {
		nextRetryAt = *w.NextRetryAt
	}

	_, err := t.tx.ExecContext(ctx,
		`INSERT INTO webhook_deliveries (id, project_id, message_id, verification_id, kind, url, payload, headers, attempt, status, response_status, response_body, next_retry_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ID, w.ProjectID, messageID, verificationID, w.Kind, w.URL, string(payload), string(headers),
		w.Attempt, w.Status, w.ResponseStatus, w.ResponseBody, nextRetryAt, w.CreatedAt)
	return err
}

func (t *txStore) GetWebhookDelivery(ctx context.Context, id string) (*core.WebhookDelivery, error) {
	return t.base.GetWebhookDelivery(ctx, id)
}

func (t *txStore) UpdateWebhookDelivery(ctx context.Context, w *core.WebhookDelivery) error {
	payload, _ := json.Marshal(w.Payload)
	headers, _ := json.Marshal(w.Headers)

	var messageID, verificationID interface{}
	if w.MessageID != nil {
		messageID = *w.MessageID
	}
	if w.VerificationID != nil {
		verificationID = *w.VerificationID
	}
	var nextRetryAt interface{}
	if w.NextRetryAt != nil {
		nextRetryAt = *w.NextRetryAt
	}

	_, err := t.tx.ExecContext(ctx,
		`UPDATE webhook_deliveries SET project_id = ?, message_id = ?, verification_id = ?, kind = ?, url = ?, payload = ?, headers = ?, attempt = ?, status = ?, response_status = ?, response_body = ?, next_retry_at = ? WHERE id = ?`,
		w.ProjectID, messageID, verificationID, w.Kind, w.URL, string(payload), string(headers),
		w.Attempt, w.Status, w.ResponseStatus, w.ResponseBody, nextRetryAt, w.ID)
	return err
}

func (t *txStore) ListPendingWebhooks(ctx context.Context, limit int) ([]*core.WebhookDelivery, error) {
	return t.base.ListPendingWebhooks(ctx, limit)
}

func (t *txStore) CreateRequestLog(ctx context.Context, l *core.RequestLog) error {
	headers, _ := json.Marshal(l.RequestHeaders)
	_, err := t.tx.ExecContext(ctx,
		`INSERT INTO request_logs (id, project_id, adapter, method, path, request_headers, request_body, response_status, response_body, duration_ms, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.ProjectID, l.Adapter, l.Method, l.Path, string(headers), l.RequestBody, l.ResponseStatus, l.ResponseBody, l.DurationMS, l.CreatedAt)
	return err
}

func (t *txStore) GetRequestLog(ctx context.Context, id string) (*core.RequestLog, error) {
	return t.base.GetRequestLog(ctx, id)
}

func (t *txStore) ListRequestLogs(ctx context.Context, projectID string, limit int, cursor string) ([]*core.RequestLog, string, error) {
	return t.base.ListRequestLogs(ctx, projectID, limit, cursor)
}

func (t *txStore) Put(ctx context.Context, id string, r io.Reader) (int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO blobs (id, data) VALUES (?, ?)`, id, data)
	return int64(len(data)), err
}

func (t *txStore) Get(ctx context.Context, id string) (io.ReadCloser, error) {
	return t.base.Get(ctx, id)
}

func (t *txStore) Delete(ctx context.Context, id string) error {
	_, err := t.tx.ExecContext(ctx, `DELETE FROM blobs WHERE id = ?`, id)
	return err
}
