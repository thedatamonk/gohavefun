package registry

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type sqliteRegistryStore struct {
	db *sql.DB
}

func NewSQLiteRegistry(dbPath string) (*Registry, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS feature_views (
			name        TEXT PRIMARY KEY,
			entity_type TEXT NOT NULL,
			description TEXT,
			owner       TEXT,
			features    TEXT NOT NULL,
			tags        TEXT,
			created_at  TIMESTAMP NOT NULL,
			updated_at  TIMESTAMP NOT NULL
		)
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}

	return &Registry{store: &sqliteRegistryStore{db: db}}, nil
}

func (s *sqliteRegistryStore) create(view FeatureViewDef) error {
	featuresJSON, err := json.Marshal(view.Features)
	if err != nil {
		return err
	}
	tagsJSON, err := json.Marshal(view.Tags)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	_, err = s.db.Exec(`
		INSERT INTO feature_views (name, entity_type, description, owner, features, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, view.Name, view.EntityType, view.Description, view.Owner, string(featuresJSON), string(tagsJSON), now, now)
	if err != nil {
		return fmt.Errorf("feature view %q already exists", view.Name)
	}
	return nil
}

func (s *sqliteRegistryStore) get(name string) (FeatureViewDef, bool) {
	var view FeatureViewDef
	var featuresJSON, tagsJSON string

	err := s.db.QueryRow(`
		SELECT name, entity_type, description, owner, features, tags, created_at, updated_at
		FROM feature_views WHERE name = ?
	`, name).Scan(&view.Name, &view.EntityType, &view.Description, &view.Owner,
		&featuresJSON, &tagsJSON, &view.CreatedAt, &view.UpdatedAt)
	if err == sql.ErrNoRows {
		return FeatureViewDef{}, false
	}
	if err != nil {
		return FeatureViewDef{}, false
	}

	json.Unmarshal([]byte(featuresJSON), &view.Features)
	json.Unmarshal([]byte(tagsJSON), &view.Tags)
	return view, true
}

func (s *sqliteRegistryStore) list() []FeatureViewDef {
	rows, err := s.db.Query(`
		SELECT name, entity_type, description, owner, features, tags, created_at, updated_at
		FROM feature_views ORDER BY name
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var views []FeatureViewDef
	for rows.Next() {
		var view FeatureViewDef
		var featuresJSON, tagsJSON string
		if err := rows.Scan(&view.Name, &view.EntityType, &view.Description, &view.Owner,
			&featuresJSON, &tagsJSON, &view.CreatedAt, &view.UpdatedAt); err != nil {
			continue
		}
		json.Unmarshal([]byte(featuresJSON), &view.Features)
		json.Unmarshal([]byte(tagsJSON), &view.Tags)
		views = append(views, view)
	}
	return views
}

func (s *sqliteRegistryStore) update(name string, view FeatureViewDef) error {
	featuresJSON, err := json.Marshal(view.Features)
	if err != nil {
		return err
	}
	tagsJSON, err := json.Marshal(view.Tags)
	if err != nil {
		return err
	}

	result, err := s.db.Exec(`
		UPDATE feature_views
		SET entity_type = ?, description = ?, owner = ?, features = ?, tags = ?, updated_at = ?
		WHERE name = ?
	`, view.EntityType, view.Description, view.Owner, string(featuresJSON), string(tagsJSON), time.Now().UTC(), name)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("feature view %q not found", name)
	}
	return nil
}

func (s *sqliteRegistryStore) delete(name string) error {
	result, err := s.db.Exec("DELETE FROM feature_views WHERE name = ?", name)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("feature view %q not found", name)
	}
	return nil
}

func (s *sqliteRegistryStore) close() error {
	return s.db.Close()
}
