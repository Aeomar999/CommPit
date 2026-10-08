package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Aeomar999/CommPit/core"
)

func (s *Store) CreateVerification(ctx context.Context, v *core.Verification) error {
	var serviceRef interface{}
	if v.ServiceRef != nil {
		serviceRef = *v.ServiceRef
	}
	err := s.execContext(ctx,
		`INSERT INTO verifications (id, project_id, provider, provider_ref, service_ref, to_addr, channel, code, status, attempts, max_attempts, expires_at, message_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.ProjectID, v.Provider, v.ProviderRef, serviceRef, v.To, v.Channel, v.Code, v.Status, v.Attempts, v.MaxAttempts, v.ExpiresAt, v.MessageID, v.CreatedAt)
	return err
}

func (s *Store) GetVerification(ctx context.Context, projectID, id string) (*core.Verification, error) {
	row := s.queryRowContext(ctx,
		`SELECT id, project_id, provider, provider_ref, service_ref, to_addr, channel, code, status, attempts, max_attempts, expires_at, message_id, created_at
		 FROM verifications WHERE id = ? AND project_id = ?`, id, projectID)
	v, err := scanVerification(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.NewVerificationNotFound("verification not found", "id")
		}
		return nil, err
	}
	return v, nil
}

func (s *Store) GetVerificationByProviderRef(ctx context.Context, projectID, ref string) (*core.Verification, error) {
	row := s.queryRowContext(ctx,
		`SELECT id, project_id, provider, provider_ref, service_ref, to_addr, channel, code, status, attempts, max_attempts, expires_at, message_id, created_at
		 FROM verifications WHERE project_id = ? AND provider_ref = ?`, projectID, ref)
	v, err := scanVerification(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.NewVerificationNotFound("verification not found", "ref")
		}
		return nil, err
	}
	return v, nil
}

func scanVerification(row interface {
	Scan(dest ...interface{}) error
}) (*core.Verification, error) {
	var v core.Verification
	var serviceRef sql.NullString
	err := row.Scan(&v.ID, &v.ProjectID, &v.Provider, &v.ProviderRef, &serviceRef, &v.To, &v.Channel, &v.Code, &v.Status, &v.Attempts, &v.MaxAttempts, &v.ExpiresAt, &v.MessageID, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	if serviceRef.Valid {
		v.ServiceRef = &serviceRef.String
	}
	return &v, nil
}

func (s *Store) UpdateVerification(ctx context.Context, v *core.Verification) error {
	var serviceRef interface{}
	if v.ServiceRef != nil {
		serviceRef = *v.ServiceRef
	}
	err := s.execContext(ctx,
		`UPDATE verifications SET provider = ?, provider_ref = ?, service_ref = ?, to_addr = ?, channel = ?, code = ?, status = ?, attempts = ?, max_attempts = ?, expires_at = ?, message_id = ? WHERE id = ? AND project_id = ?`,
		v.Provider, v.ProviderRef, serviceRef, v.To, v.Channel, v.Code, v.Status, v.Attempts, v.MaxAttempts, v.ExpiresAt, v.MessageID, v.ID, v.ProjectID)
	return err
}

func (s *Store) ListVerifications(ctx context.Context, projectID string, limit int, cursor string) ([]*core.Verification, string, error) {
	query := `SELECT id, project_id, provider, provider_ref, service_ref, to_addr, channel, code, status, attempts, max_attempts, expires_at, message_id, created_at FROM verifications WHERE project_id = ?`
	args := []interface{}{projectID}
	if cursor != "" {
		query += ` AND id > ?`
		args = append(args, cursor)
	}
	query += ` ORDER BY id LIMIT ?`
	args = append(args, limit+1)

	rows, err := s.queryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var verifications []*core.Verification
	for rows.Next() {
		v, err := scanVerification(rows)
		if err != nil {
			return nil, "", err
		}
		verifications = append(verifications, v)
	}

	var nextCursor string
	if len(verifications) > limit {
		nextCursor = verifications[limit-1].ID
		verifications = verifications[:limit]
	}
	return verifications, nextCursor, nil
}

func (s *Store) CreateUnsubscribe(ctx context.Context, u *core.Unsubscribe) error {
	err := s.execContext(ctx,
		`INSERT OR REPLACE INTO unsubscribes (project_id, number, at) VALUES (?, ?, ?)`,
		u.ProjectID, u.Number, u.At)
	return err
}

func (s *Store) DeleteUnsubscribe(ctx context.Context, projectID, number string) error {
	err := s.execContext(ctx,
		`DELETE FROM unsubscribes WHERE project_id = ? AND number = ?`, projectID, number)
	return err
}

func (s *Store) IsUnsubscribed(ctx context.Context, projectID, number string) (bool, error) {
	row := s.queryRowContext(ctx,
		`SELECT 1 FROM unsubscribes WHERE project_id = ? AND number = ?`, projectID, number)
	var exists int
	err := row.Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) CreateAttachment(ctx context.Context, a *core.Attachment) error {
	err := s.execContext(ctx,
		`INSERT INTO attachments (id, message_id, filename, content_type, size, blob_id, inline_cid) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.MessageID, a.Filename, a.ContentType, a.Size, a.BlobID, a.InlineCID)
	return err
}

func (s *Store) GetAttachment(ctx context.Context, id string) (*core.Attachment, error) {
	row := s.queryRowContext(ctx,
		`SELECT id, message_id, filename, content_type, size, blob_id, inline_cid FROM attachments WHERE id = ?`, id)
	var a core.Attachment
	var inlineCID sql.NullString
	if err := row.Scan(&a.ID, &a.MessageID, &a.Filename, &a.ContentType, &a.Size, &a.BlobID, &inlineCID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.NewNotFound("attachment not found", "id")
		}
		return nil, err
	}
	if inlineCID.Valid {
		a.InlineCID = inlineCID.String
	}
	return &a, nil
}

func (s *Store) ListAttachments(ctx context.Context, messageID string) ([]*core.Attachment, error) {
	rows, err := s.queryContext(ctx,
		`SELECT id, message_id, filename, content_type, size, blob_id, inline_cid FROM attachments WHERE message_id = ?`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []*core.Attachment
	for rows.Next() {
		var a core.Attachment
		var inlineCID sql.NullString
		if err := rows.Scan(&a.ID, &a.MessageID, &a.Filename, &a.ContentType, &a.Size, &a.BlobID, &inlineCID); err != nil {
			return nil, err
		}
		if inlineCID.Valid {
			a.InlineCID = inlineCID.String
		}
		attachments = append(attachments, &a)
	}
	return attachments, nil
}

func (s *Store) CreateWebhookDelivery(ctx context.Context, w *core.WebhookDelivery) error {
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

	err := s.execContext(ctx,
		`INSERT INTO webhook_deliveries (id, project_id, message_id, verification_id, kind, url, payload, headers, attempt, status, response_status, response_body, next_retry_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ID, w.ProjectID, messageID, verificationID, w.Kind, w.URL, string(payload), string(headers),
		w.Attempt, w.Status, w.ResponseStatus, w.ResponseBody, nextRetryAt, w.CreatedAt)
	return err
}

func (s *Store) GetWebhookDelivery(ctx context.Context, id string) (*core.WebhookDelivery, error) {
	row := s.queryRowContext(ctx,
		`SELECT id, project_id, message_id, verification_id, kind, url, payload, headers, attempt, status, response_status, response_body, next_retry_at, created_at
		 FROM webhook_deliveries WHERE id = ?`, id)
	w, err := scanWebhookDelivery(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.NewNotFound("webhook delivery not found", "id")
		}
		return nil, err
	}
	return w, nil
}

func scanWebhookDelivery(row interface {
	Scan(dest ...interface{}) error
}) (*core.WebhookDelivery, error) {
	var w core.WebhookDelivery
	var payload, headers string
	var messageID, verificationID sql.NullString
	var responseStatus sql.NullInt64
	var responseBody, nextRetryAt sql.NullString
	err := row.Scan(&w.ID, &w.ProjectID, &messageID, &verificationID, &w.Kind, &w.URL, &payload, &headers, &w.Attempt, &w.Status, &responseStatus, &responseBody, &nextRetryAt, &w.CreatedAt)
	if err != nil {
		return nil, err
	}
	if messageID.Valid {
		w.MessageID = &messageID.String
	}
	if verificationID.Valid {
		w.VerificationID = &verificationID.String
	}
	if responseStatus.Valid {
		val := int(responseStatus.Int64)
		w.ResponseStatus = &val
	}
	if responseBody.Valid {
		w.ResponseBody = &responseBody.String
	}
	if nextRetryAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", nextRetryAt.String)
		w.NextRetryAt = &t
	}
	json.Unmarshal([]byte(payload), &w.Payload)
	json.Unmarshal([]byte(headers), &w.Headers)
	return &w, nil
}

func (s *Store) UpdateWebhookDelivery(ctx context.Context, w *core.WebhookDelivery) error {
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

	err := s.execContext(ctx,
		`UPDATE webhook_deliveries SET project_id = ?, message_id = ?, verification_id = ?, kind = ?, url = ?, payload = ?, headers = ?, attempt = ?, status = ?, response_status = ?, response_body = ?, next_retry_at = ? WHERE id = ?`,
		w.ProjectID, messageID, verificationID, w.Kind, w.URL, string(payload), string(headers),
		w.Attempt, w.Status, w.ResponseStatus, w.ResponseBody, nextRetryAt, w.ID)
	return err
}

func (s *Store) ListPendingWebhooks(ctx context.Context, limit int) ([]*core.WebhookDelivery, error) {
	rows, err := s.queryContext(ctx,
		`SELECT id, project_id, message_id, verification_id, kind, url, payload, headers, attempt, status, response_status, response_body, next_retry_at, created_at
		 FROM webhook_deliveries WHERE status = 'pending' AND (next_retry_at IS NULL OR next_retry_at <= ?) ORDER BY next_retry_at LIMIT ?`,
		time.Now(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []*core.WebhookDelivery
	for rows.Next() {
		w, err := scanWebhookDelivery(rows)
		if err != nil {
			return nil, err
		}
		deliveries = append(deliveries, w)
	}
	return deliveries, nil
}

func (s *Store) CreateRequestLog(ctx context.Context, l *core.RequestLog) error {
	headers, _ := json.Marshal(l.RequestHeaders)
	err := s.execContext(ctx,
		`INSERT INTO request_logs (id, project_id, adapter, method, path, request_headers, request_body, response_status, response_body, duration_ms, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.ProjectID, l.Adapter, l.Method, l.Path, string(headers), l.RequestBody, l.ResponseStatus, l.ResponseBody, l.DurationMS, l.CreatedAt)
	return err
}

func (s *Store) GetRequestLog(ctx context.Context, id string) (*core.RequestLog, error) {
	row := s.queryRowContext(ctx,
		`SELECT id, project_id, adapter, method, path, request_headers, request_body, response_status, response_body, duration_ms, created_at
		 FROM request_logs WHERE id = ?`, id)
	var l core.RequestLog
	var headers string
	if err := row.Scan(&l.ID, &l.ProjectID, &l.Adapter, &l.Method, &l.Path, &headers, &l.RequestBody, &l.ResponseStatus, &l.ResponseBody, &l.DurationMS, &l.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.NewNotFound("request log not found", "id")
		}
		return nil, err
	}
	json.Unmarshal([]byte(headers), &l.RequestHeaders)
	return &l, nil
}

func (s *Store) ListRequestLogs(ctx context.Context, projectID string, limit int, cursor string) ([]*core.RequestLog, string, error) {
	query := `SELECT id, project_id, adapter, method, path, request_headers, request_body, response_status, response_body, duration_ms, created_at FROM request_logs WHERE project_id = ?`
	args := []interface{}{projectID}
	if cursor != "" {
		query += ` AND id > ?`
		args = append(args, cursor)
	}
	query += ` ORDER BY id LIMIT ?`
	args = append(args, limit+1)

	rows, err := s.queryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var logs []*core.RequestLog
	for rows.Next() {
		var l core.RequestLog
		var headers string
		if err := rows.Scan(&l.ID, &l.ProjectID, &l.Adapter, &l.Method, &l.Path, &headers, &l.RequestBody, &l.ResponseStatus, &l.ResponseBody, &l.DurationMS, &l.CreatedAt); err != nil {
			return nil, "", err
		}
		json.Unmarshal([]byte(headers), &l.RequestHeaders)
		logs = append(logs, &l)
	}

	var nextCursor string
	if len(logs) > limit {
		nextCursor = logs[limit-1].ID
		logs = logs[:limit]
	}
	return logs, nextCursor, nil
}
