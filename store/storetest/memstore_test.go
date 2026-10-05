package storetest

import (
	"context"
	"sync"
	"testing"

	"github.com/Aeomar999/CommPit/core"
)

type memStore struct {
	mu                sync.Mutex
	projects          map[string]*core.Project
	credentials       map[string]*core.Credential
	messages          map[string]*core.Message
	statusEvents      map[string][]*core.StatusEvent
	batches           map[string]*core.Batch
	verifications     map[string]*core.Verification
	unsubscribes      map[string]map[string]*core.Unsubscribe
	attachments       map[string]*core.Attachment
	webhookDeliveries map[string]*core.WebhookDelivery
	requestLogs       map[string]*core.RequestLog
	inTx              bool
	txSnapshot        *txSnapshot
}

type txSnapshot struct {
	projects          map[string]*core.Project
	credentials       map[string]*core.Credential
	messages          map[string]*core.Message
	statusEvents      map[string][]*core.StatusEvent
	batches           map[string]*core.Batch
	verifications     map[string]*core.Verification
	unsubscribes      map[string]map[string]*core.Unsubscribe
	attachments       map[string]*core.Attachment
	webhookDeliveries map[string]*core.WebhookDelivery
	requestLogs       map[string]*core.RequestLog
}

func (s *memStore) snapshot() *txSnapshot {
	// Deep copy all maps
	projects := make(map[string]*core.Project, len(s.projects))
	for k, v := range s.projects {
		projects[k] = v
	}
	credentials := make(map[string]*core.Credential, len(s.credentials))
	for k, v := range s.credentials {
		credentials[k] = v
	}
	messages := make(map[string]*core.Message, len(s.messages))
	for k, v := range s.messages {
		messages[k] = v
	}
	statusEvents := make(map[string][]*core.StatusEvent, len(s.statusEvents))
	for k, v := range s.statusEvents {
		cp := make([]*core.StatusEvent, len(v))
		copy(cp, v)
		statusEvents[k] = cp
	}
	batches := make(map[string]*core.Batch, len(s.batches))
	for k, v := range s.batches {
		batches[k] = v
	}
	verifications := make(map[string]*core.Verification, len(s.verifications))
	for k, v := range s.verifications {
		verifications[k] = v
	}
	unsubscribes := make(map[string]map[string]*core.Unsubscribe, len(s.unsubscribes))
	for k, v := range s.unsubscribes {
		unsubscribes[k] = make(map[string]*core.Unsubscribe, len(v))
		for k2, v2 := range v {
			unsubscribes[k][k2] = v2
		}
	}
	attachments := make(map[string]*core.Attachment, len(s.attachments))
	for k, v := range s.attachments {
		attachments[k] = v
	}
	webhookDeliveries := make(map[string]*core.WebhookDelivery, len(s.webhookDeliveries))
	for k, v := range s.webhookDeliveries {
		webhookDeliveries[k] = v
	}
	requestLogs := make(map[string]*core.RequestLog, len(s.requestLogs))
	for k, v := range s.requestLogs {
		requestLogs[k] = v
	}
	return &txSnapshot{
		projects:          projects,
		credentials:       credentials,
		messages:          messages,
		statusEvents:      statusEvents,
		batches:           batches,
		verifications:     verifications,
		unsubscribes:      unsubscribes,
		attachments:       attachments,
		webhookDeliveries: webhookDeliveries,
		requestLogs:       requestLogs,
	}
}

func (s *memStore) restore(snap *txSnapshot) {
	s.projects = snap.projects
	s.credentials = snap.credentials
	s.messages = snap.messages
	s.statusEvents = snap.statusEvents
	s.batches = snap.batches
	s.verifications = snap.verifications
	s.unsubscribes = snap.unsubscribes
	s.attachments = snap.attachments
	s.webhookDeliveries = snap.webhookDeliveries
	s.requestLogs = snap.requestLogs
}

func newMemStore() *memStore {
	return &memStore{
		projects:          make(map[string]*core.Project),
		credentials:       make(map[string]*core.Credential),
		messages:          make(map[string]*core.Message),
		statusEvents:      make(map[string][]*core.StatusEvent),
		batches:           make(map[string]*core.Batch),
		verifications:     make(map[string]*core.Verification),
		unsubscribes:      make(map[string]map[string]*core.Unsubscribe),
		attachments:       make(map[string]*core.Attachment),
		webhookDeliveries: make(map[string]*core.WebhookDelivery),
		requestLogs:       make(map[string]*core.RequestLog),
	}
}

func (s *memStore) CreateProject(ctx context.Context, p *core.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects[p.ID] = p
	return nil
}

func (s *memStore) GetProject(ctx context.Context, id string) (*core.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.projects[id]
	if !ok {
		return nil, core.NewNotFound("project not found", "id")
	}
	return p, nil
}

func (s *memStore) UpdateProject(ctx context.Context, p *core.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects[p.ID] = p
	return nil
}

func (s *memStore) ListProjects(ctx context.Context, limit int, cursor string) ([]*core.Project, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*core.Project
	for _, p := range s.projects {
		result = append(result, p)
		if len(result) >= limit {
			break
		}
	}
	return result, "", nil
}

func (s *memStore) CreateCredential(ctx context.Context, c *core.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := c.Provider + ":" + c.Key
	s.credentials[key] = c
	return nil
}

func (s *memStore) GetCredential(ctx context.Context, provider, key string) (*core.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.credentials[provider+":"+key]
	if !ok {
		return nil, core.NewNotFound("credential not found", "id")
	}
	return c, nil
}

func (s *memStore) ListCredentials(ctx context.Context, projectID string) ([]*core.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*core.Credential
	for _, c := range s.credentials {
		if c.ProjectID == projectID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *memStore) DeleteCredential(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, c := range s.credentials {
		if c.ID == id {
			delete(s.credentials, k)
			return nil
		}
	}
	return nil
}

func (s *memStore) CreateMessage(ctx context.Context, m *core.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[m.ID] = m
	return nil
}

func (s *memStore) GetMessage(ctx context.Context, projectID, messageID string) (*core.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.messages[messageID]
	if !ok || m.ProjectID != projectID {
		return nil, core.NewNotFound("message not found", "id")
	}
	return m, nil
}

func (s *memStore) UpdateMessage(ctx context.Context, m *core.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[m.ID] = m
	return nil
}

func (s *memStore) ListMessages(ctx context.Context, projectID string, filter core.MessageFilter) ([]*core.Message, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*core.Message
	for _, m := range s.messages {
		if m.ProjectID != projectID {
			continue
		}
		if filter.Channel != nil && m.Channel != *filter.Channel {
			continue
		}
		if filter.Status != nil && m.Status != *filter.Status {
			continue
		}
		if filter.To != nil && m.To != *filter.To {
			continue
		}
		if filter.From != nil && m.From != *filter.From {
			continue
		}
		if filter.BatchID != nil && (m.BatchID == nil || *m.BatchID != *filter.BatchID) {
			continue
		}
		if filter.Direction != nil && m.Direction != *filter.Direction {
			continue
		}
		if filter.Since != nil && m.CreatedAt.Before(*filter.Since) {
			continue
		}
		result = append(result, m)
		if len(result) >= filter.Limit {
			break
		}
	}
	return result, "", nil
}

func (s *memStore) DeleteMessages(ctx context.Context, projectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, m := range s.messages {
		if m.ProjectID == projectID {
			delete(s.messages, id)
		}
	}
	return nil
}

func (s *memStore) CreateStatusEvent(ctx context.Context, e *core.StatusEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusEvents[e.MessageID] = append(s.statusEvents[e.MessageID], e)
	return nil
}

func (s *memStore) GetStatusEvents(ctx context.Context, messageID string) ([]*core.StatusEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statusEvents[messageID], nil
}

func (s *memStore) CreateBatch(ctx context.Context, b *core.Batch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.batches[b.ID] = b
	return nil
}

func (s *memStore) GetBatch(ctx context.Context, projectID, batchID string) (*core.Batch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.batches[batchID]
	if !ok || b.ProjectID != projectID {
		return nil, core.NewNotFound("batch not found", "id")
	}
	return b, nil
}

func (s *memStore) UpdateBatch(ctx context.Context, b *core.Batch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.batches[b.ID] = b
	return nil
}

func (s *memStore) ListBatches(ctx context.Context, projectID string, limit int, cursor string) ([]*core.Batch, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*core.Batch
	for _, b := range s.batches {
		if b.ProjectID == projectID {
			result = append(result, b)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, "", nil
}

func (s *memStore) CreateVerification(ctx context.Context, v *core.Verification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.verifications[v.ID] = v
	return nil
}

func (s *memStore) GetVerification(ctx context.Context, projectID, id string) (*core.Verification, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.verifications[id]
	if !ok || v.ProjectID != projectID {
		return nil, core.NewVerificationNotFound("verification not found", "id")
	}
	return v, nil
}

func (s *memStore) GetVerificationByProviderRef(ctx context.Context, projectID, ref string) (*core.Verification, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range s.verifications {
		if v.ProjectID == projectID && v.ProviderRef == ref {
			return v, nil
		}
	}
	return nil, core.NewVerificationNotFound("verification not found", "ref")
}

func (s *memStore) UpdateVerification(ctx context.Context, v *core.Verification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.verifications[v.ID] = v
	return nil
}

func (s *memStore) ListVerifications(ctx context.Context, projectID string, limit int, cursor string) ([]*core.Verification, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*core.Verification
	for _, v := range s.verifications {
		if v.ProjectID == projectID {
			result = append(result, v)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, "", nil
}

func (s *memStore) CreateUnsubscribe(ctx context.Context, u *core.Unsubscribe) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unsubscribes[u.ProjectID] == nil {
		s.unsubscribes[u.ProjectID] = make(map[string]*core.Unsubscribe)
	}
	s.unsubscribes[u.ProjectID][u.Number] = u
	return nil
}

func (s *memStore) DeleteUnsubscribe(ctx context.Context, projectID, number string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unsubscribes[projectID] != nil {
		delete(s.unsubscribes[projectID], number)
	}
	return nil
}

func (s *memStore) IsUnsubscribed(ctx context.Context, projectID, number string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unsubscribes[projectID] != nil {
		_, ok := s.unsubscribes[projectID][number]
		return ok, nil
	}
	return false, nil
}

func (s *memStore) CreateAttachment(ctx context.Context, a *core.Attachment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attachments[a.ID] = a
	return nil
}

func (s *memStore) GetAttachment(ctx context.Context, id string) (*core.Attachment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.attachments[id]
	if !ok {
		return nil, core.NewNotFound("attachment not found", "id")
	}
	return a, nil
}

func (s *memStore) ListAttachments(ctx context.Context, messageID string) ([]*core.Attachment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*core.Attachment
	for _, a := range s.attachments {
		if a.MessageID == messageID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (s *memStore) CreateWebhookDelivery(ctx context.Context, w *core.WebhookDelivery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.webhookDeliveries[w.ID] = w
	return nil
}

func (s *memStore) GetWebhookDelivery(ctx context.Context, id string) (*core.WebhookDelivery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.webhookDeliveries[id]
	if !ok {
		return nil, core.NewNotFound("webhook delivery not found", "id")
	}
	return w, nil
}

func (s *memStore) UpdateWebhookDelivery(ctx context.Context, w *core.WebhookDelivery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.webhookDeliveries[w.ID] = w
	return nil
}

func (s *memStore) ListPendingWebhooks(ctx context.Context, limit int) ([]*core.WebhookDelivery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*core.WebhookDelivery
	for _, w := range s.webhookDeliveries {
		if w.Status == core.WebhookPending {
			result = append(result, w)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (s *memStore) CreateRequestLog(ctx context.Context, l *core.RequestLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestLogs[l.ID] = l
	return nil
}

func (s *memStore) GetRequestLog(ctx context.Context, id string) (*core.RequestLog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.requestLogs[id]
	if !ok {
		return nil, core.NewNotFound("request log not found", "id")
	}
	return l, nil
}

func (s *memStore) ListRequestLogs(ctx context.Context, projectID string, limit int, cursor string) ([]*core.RequestLog, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*core.RequestLog
	for _, l := range s.requestLogs {
		if l.ProjectID == projectID {
			result = append(result, l)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, "", nil
}

func (s *memStore) Transaction(ctx context.Context, fn func(core.Store) error) error {
	s.mu.Lock()
	snap := s.snapshot()
	s.mu.Unlock()

	err := fn(s)

	if err != nil {
		s.mu.Lock()
		s.restore(snap)
		s.mu.Unlock()
	}
	return err
}

func TestMemStoreConformance(t *testing.T) {
	RunStoreTests(t, func() (core.Store, func()) {
		s := newMemStore()
		return s, func() {}
	})
}
