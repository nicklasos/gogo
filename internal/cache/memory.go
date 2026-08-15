package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// MemoryCache is an in-process cache implementing Cache (for tests and local tools).
type MemoryCache struct {
	mu   sync.Mutex
	data map[string][]byte
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{data: make(map[string][]byte)}
}

func (m *MemoryCache) Get(_ context.Context, key string, dest interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.data[key]
	if !ok {
		return ErrKeyNotFound
	}
	return json.Unmarshal(b, dest)
}

func (m *MemoryCache) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = b
	return nil
}

func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	return m.Forget(ctx, key)
}

func (m *MemoryCache) Remember(ctx context.Context, key string, _ time.Duration, callback func() (interface{}, error), dest interface{}) error {
	err := m.Get(ctx, key, dest)
	if err == nil {
		return nil
	}
	if err != ErrKeyNotFound {
		return err
	}
	value, err := callback()
	if err != nil {
		return err
	}
	if err := m.Set(ctx, key, value, 0); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (m *MemoryCache) Forget(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *MemoryCache) Flush(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string][]byte)
	return nil
}

func (m *MemoryCache) Has(_ context.Context, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.data[key]
	return ok, nil
}

var _ Cache = (*MemoryCache)(nil)
