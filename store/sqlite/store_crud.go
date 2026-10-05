package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/Aeomar999/CommPit/core"
)

func (s *Store) CreateProject(ctx context.Context, p *core.Project) error {
	settings, _ := json.Marshal(p.Settings)
	err := s.execContext(ctx,
		`INSERT INTO projects (id, name, settings, created_at) VALUES (?, ?, ?, ?)`,
		p.ID, p.Name, string(settings), p.CreatedAt)
	return err
}

func (s *Store) GetProject(ctx context.Context, id string) (*core.Project, error) {
	row := s.queryRowContext(ctx,
		`SELECT id, name, settings, created_at FROM projects WHERE id = ?`, id)
	var p core.Project
	var settings string
	if err := row.Scan(&p.ID, &p.Name, &settings, &p.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.NewNotFound("project not found", "id")
		}
		return nil, err
	}
	json.Unmarshal([]byte(settings), &p.Settings)
	return &p, nil
}

func (s *Store) UpdateProject(ctx context.Context, p *core.Project) error {
	settings, _ := json.Marshal(p.Settings)
	err := s.execContext(ctx,
		`UPDATE projects SET name = ?, settings = ? WHERE id = ?`,
		p.Name, string(settings), p.ID)
	return err
}

func (s *Store) ListProjects(ctx context.Context, limit int, cursor string) ([]*core.Project, string, error) {
	query := `SELECT id, name, settings, created_at FROM projects`
	args := []interface{}{}
	if cursor != "" {
		query += ` WHERE id > ?`
		args = append(args, cursor)
	}
	query += ` ORDER BY id LIMIT ?`
	args = append(args, limit+1)

	rows, err := s.queryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var projects []*core.Project
	for rows.Next() {
		var p core.Project
		var settings string
		if err := rows.Scan(&p.ID, &p.Name, &settings, &p.CreatedAt); err != nil {
			return nil, "", err
		}
		json.Unmarshal([]byte(settings), &p.Settings)
		projects = append(projects, &p)
	}

	var nextCursor string
	if len(projects) > limit {
		nextCursor = projects[limit-1].ID
		projects = projects[:limit]
	}
	return projects, nextCursor, nil
}

func (s *Store) CreateCredential(ctx context.Context, c *core.Credential) error {
	err := s.execContext(ctx,
		`INSERT INTO credentials (id, provider, key, project_id, created_at) VALUES (?, ?, ?, ?, ?)`,
		c.ID, c.Provider, c.Key, c.ProjectID, c.CreatedAt)
	return err
}

func (s *Store) GetCredential(ctx context.Context, provider, key string) (*core.Credential, error) {
	row := s.queryRowContext(ctx,
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

func (s *Store) ListCredentials(ctx context.Context, projectID string) ([]*core.Credential, error) {
	rows, err := s.queryContext(ctx,
		`SELECT id, provider, key, project_id, created_at FROM credentials WHERE project_id = ?`,
		projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []*core.Credential
	for rows.Next() {
		var c core.Credential
		if err := rows.Scan(&c.ID, &c.Provider, &c.Key, &c.ProjectID, &c.CreatedAt); err != nil {
			return nil, err
		}
		creds = append(creds, &c)
	}
	return creds, nil
}

func (s *Store) DeleteCredential(ctx context.Context, id string) error {
	err := s.execContext(ctx, `DELETE FROM credentials WHERE id = ?`, id)
	return err
}

func scanMessage(row interface {
	Scan(dest ...interface{}) error
}) (*core.Message, error) {
	var m core.Message
	var cc, bcc, extractedCodes, extractedLinks string
	var batchID, rawBlobID, primaryLink, errorCode, errorMessage, callbackURL sql.NullString
	err := row.Scan(
		&m.ID, &m.ProjectID, &batchID, &m.Channel, &m.Direction, &m.Provider, &m.ProviderRef,
		&m.From, &m.To, &cc, &bcc, &m.Subject, &m.BodyText, &m.BodyHTML, &rawBlobID,
		&m.Encoding, &m.Segments, &m.Status, &errorCode, &errorMessage, &callbackURL,
		&extractedCodes, &extractedLinks, &primaryLink, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if batchID.Valid {
		m.BatchID = &batchID.String
	}
	if rawBlobID.Valid {
		m.RawBlobID = &rawBlobID.String
	}
	if primaryLink.Valid {
		m.PrimaryLink = &primaryLink.String
	}
	if errorCode.Valid {
		m.ErrorCode = &errorCode.String
	}
	if errorMessage.Valid {
		m.ErrorMessage = &errorMessage.String
	}
	if callbackURL.Valid {
		m.CallbackURL = &callbackURL.String
	}
	json.Unmarshal([]byte(cc), &m.CC)
	json.Unmarshal([]byte(bcc), &m.BCC)
	json.Unmarshal([]byte(extractedCodes), &m.ExtractedCodes)
	json.Unmarshal([]byte(extractedLinks), &m.ExtractedLinks)
	return &m, nil
}

func (s *Store) CreateMessage(ctx context.Context, m *core.Message) error {
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

	err := s.execContext(ctx,
		`INSERT INTO messages (id, project_id, batch_id, channel, direction, provider, provider_ref, from_addr, to_addr, cc, bcc, subject, body_text, body_html, raw_blob_id, encoding, segments, status, error_code, error_message, callback_url, extracted_codes, extracted_links, primary_link, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.ProjectID, batchID, m.Channel, m.Direction, m.Provider, m.ProviderRef,
		m.From, m.To, string(cc), string(bcc), m.Subject, m.BodyText, m.BodyHTML, rawBlobID,
		m.Encoding, m.Segments, m.Status, errorCode, errorMessage, callbackURL,
		string(extractedCodes), string(extractedLinks), primaryLink, m.CreatedAt, m.UpdatedAt)
	return err
}

func (s *Store) GetMessage(ctx context.Context, projectID, messageID string) (*core.Message, error) {
	row := s.queryRowContext(ctx,
		`SELECT id, project_id, batch_id, channel, direction, provider, provider_ref, from_addr, to_addr, cc, bcc, subject, body_text, body_html, raw_blob_id, encoding, segments, status, error_code, error_message, callback_url, extracted_codes, extracted_links, primary_link, created_at, updated_at
		 FROM messages WHERE id = ? AND project_id = ?`, messageID, projectID)
	m, err := scanMessage(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.NewNotFound("message not found", "id")
		}
		return nil, err
	}
	return m, nil
}

func (s *Store) UpdateMessage(ctx context.Context, m *core.Message) error {
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

	err := s.execContext(ctx,
		`UPDATE messages SET batch_id = ?, channel = ?, direction = ?, provider = ?, provider_ref = ?, from_addr = ?, to_addr = ?, cc = ?, bcc = ?, subject = ?, body_text = ?, body_html = ?, raw_blob_id = ?, encoding = ?, segments = ?, status = ?, error_code = ?, error_message = ?, callback_url = ?, extracted_codes = ?, extracted_links = ?, primary_link = ?, updated_at = ?
		 WHERE id = ? AND project_id = ?`,
		batchID, m.Channel, m.Direction, m.Provider, m.ProviderRef, m.From, m.To, string(cc), string(bcc), m.Subject, m.BodyText, m.BodyHTML, rawBlobID,
		m.Encoding, m.Segments, m.Status, errorCode, errorMessage, callbackURL,
		string(extractedCodes), string(extractedLinks), primaryLink, m.UpdatedAt, m.ID, m.ProjectID)
	return err
}

func (s *Store) ListMessages(ctx context.Context, projectID string, filter core.MessageFilter) ([]*core.Message, string, error) {
	query := `SELECT id, project_id, batch_id, channel, direction, provider, provider_ref, from_addr, to_addr, cc, bcc, subject, body_text, body_html, raw_blob_id, encoding, segments, status, error_code, error_message, callback_url, extracted_codes, extracted_links, primary_link, created_at, updated_at FROM messages WHERE project_id = ?`
	args := []interface{}{projectID}

	if filter.Channel != nil {
		query += ` AND channel = ?`
		args = append(args, *filter.Channel)
	}
	if filter.Status != nil {
		query += ` AND status = ?`
		args = append(args, *filter.Status)
	}
	if filter.To != nil {
		query += ` AND to_addr = ?`
		args = append(args, *filter.To)
	}
	if filter.From != nil {
		query += ` AND from_addr = ?`
		args = append(args, *filter.From)
	}
	if filter.BatchID != nil {
		query += ` AND batch_id = ?`
		args = append(args, *filter.BatchID)
	}
	if filter.Direction != nil {
		query += ` AND direction = ?`
		args = append(args, *filter.Direction)
	}
	if filter.Since != nil {
		query += ` AND created_at > ?`
		args = append(args, *filter.Since)
	}

	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, filter.Limit+1)

	rows, err := s.queryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var messages []*core.Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, "", err
		}
		messages = append(messages, m)
	}

	var nextCursor string
	if len(messages) > filter.Limit {
		nextCursor = messages[filter.Limit-1].ID
		messages = messages[:filter.Limit]
	}
	return messages, nextCursor, nil
}

func (s *Store) DeleteMessages(ctx context.Context, projectID string) error {
	err := s.execContext(ctx, `DELETE FROM messages WHERE project_id = ?`, projectID)
	return err
}

func (s *Store) CreateStatusEvent(ctx context.Context, e *core.StatusEvent) error {
	err := s.execContext(ctx,
		`INSERT INTO status_events (id, message_id, status, error_code, at) VALUES (?, ?, ?, ?, ?)`,
		e.ID, e.MessageID, e.Status, e.ErrorCode, e.At)
	return err
}

func (s *Store) GetStatusEvents(ctx context.Context, messageID string) ([]*core.StatusEvent, error) {
	rows, err := s.queryContext(ctx,
		`SELECT id, message_id, status, error_code, at FROM status_events WHERE message_id = ? ORDER BY at`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*core.StatusEvent
	for rows.Next() {
		var e core.StatusEvent
		var errorCode sql.NullString
		if err := rows.Scan(&e.ID, &e.MessageID, &e.Status, &errorCode, &e.At); err != nil {
			return nil, err
		}
		if errorCode.Valid {
			e.ErrorCode = &errorCode.String
		}
		events = append(events, &e)
	}
	return events, nil
}

func (s *Store) CreateBatch(ctx context.Context, b *core.Batch) error {
	counts, _ := json.Marshal(b.Counts)
	err := s.execContext(ctx,
		`INSERT INTO batches (id, project_id, provider, channel, total, counts, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		b.ID, b.ProjectID, b.Provider, b.Channel, b.Total, string(counts), b.CreatedAt)
	return err
}

func (s *Store) GetBatch(ctx context.Context, projectID, batchID string) (*core.Batch, error) {
	row := s.queryRowContext(ctx,
		`SELECT id, project_id, provider, channel, total, counts, created_at FROM batches WHERE id = ? AND project_id = ?`,
		batchID, projectID)
	var b core.Batch
	var counts string
	if err := row.Scan(&b.ID, &b.ProjectID, &b.Provider, &b.Channel, &b.Total, &counts, &b.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.NewNotFound("batch not found", "id")
		}
		return nil, err
	}
	json.Unmarshal([]byte(counts), &b.Counts)
	return &b, nil
}

func (s *Store) UpdateBatch(ctx context.Context, b *core.Batch) error {
	counts, _ := json.Marshal(b.Counts)
	err := s.execContext(ctx,
		`UPDATE batches SET total = ?, counts = ? WHERE id = ? AND project_id = ?`,
		b.Total, string(counts), b.ID, b.ProjectID)
	return err
}

func (s *Store) ListBatches(ctx context.Context, projectID string, limit int, cursor string) ([]*core.Batch, string, error) {
	query := `SELECT id, project_id, provider, channel, total, counts, created_at FROM batches WHERE project_id = ?`
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

	var batches []*core.Batch
	for rows.Next() {
		var b core.Batch
		var counts string
		if err := rows.Scan(&b.ID, &b.ProjectID, &b.Provider, &b.Channel, &b.Total, &counts, &b.CreatedAt); err != nil {
			return nil, "", err
		}
		json.Unmarshal([]byte(counts), &b.Counts)
		batches = append(batches, &b)
	}

	var nextCursor string
	if len(batches) > limit {
		nextCursor = batches[limit-1].ID
		batches = batches[:limit]
	}
	return batches, nextCursor, nil
}
