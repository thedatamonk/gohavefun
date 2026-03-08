package store

// FeatureVector is a map of feature names to float64 values.
type FeatureVector map[string]float64

// Key identifies a feature entity.
type Key struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

// Store is the interface for feature storage backends.
type Store interface {
	Get(entityType, entityID string) (FeatureVector, bool)
	Set(entityType, entityID string, features FeatureVector)
	GetBatch(keys []Key) map[string]FeatureVector
}
