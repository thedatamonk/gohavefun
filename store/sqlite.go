package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// SQLiteStore is a SQLite-backed feature store.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLiteStore backed by the given database file.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS features (
			entity_type TEXT NOT NULL,
			entity_id   TEXT NOT NULL,
			data        TEXT NOT NULL,
			PRIMARY KEY (entity_type, entity_id)
		)
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Get retrieves a feature vector for the given entity.
func (s *SQLiteStore) Get(entityType, entityID string) (FeatureVector, bool) {
	var data string
	err := s.db.QueryRow(
		"SELECT data FROM features WHERE entity_type = ? AND entity_id = ?",
		entityType, entityID,
	).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		return nil, false
	}

	var fv FeatureVector
	if err := json.Unmarshal([]byte(data), &fv); err != nil {
		return nil, false
	}
	return fv, true
}

// Set stores a feature vector for the given entity, overwriting any existing value.
func (s *SQLiteStore) Set(entityType, entityID string, features FeatureVector) {
	data, err := json.Marshal(features)
	if err != nil {
		return
	}

	s.db.Exec(`
		INSERT INTO features (entity_type, entity_id, data) VALUES (?, ?, ?)
		ON CONFLICT(entity_type, entity_id) DO UPDATE SET data = excluded.data
	`, entityType, entityID, string(data))
}

// GetBatch retrieves feature vectors for multiple keys at once.
func (s *SQLiteStore) GetBatch(keys []Key) map[string]FeatureVector {
	if len(keys) == 0 {
		return make(map[string]FeatureVector)
	}

	placeholders := make([]string, len(keys))
	args := make([]any, 0, len(keys)*2)
	for i, k := range keys {
		placeholders[i] = "(?, ?)"
		args = append(args, k.EntityType, k.EntityID)
	}

	query := fmt.Sprintf(
		"SELECT entity_type, entity_id, data FROM features WHERE (entity_type, entity_id) IN (%s)",
		strings.Join(placeholders, ", "),
	)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return make(map[string]FeatureVector)
	}
	defer rows.Close()

	result := make(map[string]FeatureVector)
	for rows.Next() {
		var entityType, entityID, data string
		if err := rows.Scan(&entityType, &entityID, &data); err != nil {
			continue
		}
		var fv FeatureVector
		if err := json.Unmarshal([]byte(data), &fv); err != nil {
			continue
		}
		result[makeKey(entityType, entityID)] = fv
	}
	return result
}
