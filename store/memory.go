package store

import "sync"

// MemoryStore is an in-memory feature store backed by a map.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]FeatureVector
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]FeatureVector),
	}
}

func makeKey(entityType, entityID string) string {
	return entityType + ":" + entityID
}

func (ms *MemoryStore) Get(entityType, entityID string) (FeatureVector, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	v, ok := ms.data[makeKey(entityType, entityID)]
	return v, ok
}

func (ms *MemoryStore) Set(entityType, entityID string, features FeatureVector) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.data[makeKey(entityType, entityID)] = features
}

func (ms *MemoryStore) GetBatch(keys []Key) map[string]FeatureVector {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	result := make(map[string]FeatureVector, len(keys))
	for _, k := range keys {
		key := makeKey(k.EntityType, k.EntityID)
		if v, ok := ms.data[key]; ok {
			result[key] = v
		}
	}
	return result
}
