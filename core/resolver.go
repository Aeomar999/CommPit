package core

import (
	"context"
	"sync"
)

type projectResolverImpl struct {
	store Store
	mu    sync.Mutex
	cache map[string]string // provider:key -> projectID
}

func NewProjectResolver(store Store) ProjectResolver {
	return &projectResolverImpl{
		store: store,
		cache: make(map[string]string),
	}
}

func (r *projectResolverImpl) Resolve(ctx context.Context, provider, key string) (string, error) {
	cacheKey := provider + ":" + key

	r.mu.Lock()
	if projectID, ok := r.cache[cacheKey]; ok {
		r.mu.Unlock()
		return projectID, nil
	}
	r.mu.Unlock()

	cred, err := r.store.GetCredential(ctx, provider, key)
	if err == nil {
		r.mu.Lock()
		r.cache[cacheKey] = cred.ProjectID
		r.mu.Unlock()
		return cred.ProjectID, nil
	}

	project := &Project{
		ID:        NewProjectID(),
		Name:      provider + ":" + maskKey(key),
		Settings:  map[string]interface{}{},
		CreatedAt: RealClock{}.Now(),
	}
	if err := r.store.CreateProject(ctx, project); err != nil {
		return "", err
	}

	cred = &Credential{
		ID:        NewRequestLogID(),
		Provider:  provider,
		Key:       key,
		ProjectID: project.ID,
		CreatedAt: RealClock{}.Now(),
	}
	if err := r.store.CreateCredential(ctx, cred); err != nil {
		return "", err
	}

	r.mu.Lock()
	r.cache[cacheKey] = project.ID
	r.mu.Unlock()

	return project.ID, nil
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}