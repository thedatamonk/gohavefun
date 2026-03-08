package store

import "sync"

type FeatureVector map[string]float64

type FeatureStore struct {
	mu   sync.RWMutex
	data map[string]FeatureVector
}

func NewFeatureStore() *FeatureStore {
	return &FeatureStore{
		data: make(map[string]FeatureVector),
	}
}

func makeKey(entityType, entityID string) string {
	return entityType + ":" + entityID
}

func (fs *FeatureStore) Get(entityType, entityID string) (FeatureVector, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	v, ok := fs.data[makeKey(entityType, entityID)]
	return v, ok
}

func (fs *FeatureStore) Set(entityType, entityID string, features FeatureVector) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.data[makeKey(entityType, entityID)] = features
}

type Key struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

func (fs *FeatureStore) GetBatch(keys []Key) map[string]FeatureVector {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	result := make(map[string]FeatureVector, len(keys))
	for _, k := range keys {
		key := makeKey(k.EntityType, k.EntityID)
		if v, ok := fs.data[key]; ok {
			result[key] = v
		}
	}
	return result
}
