package handler

import (
	"encoding/json"
	"net/http"

	"github.com/rohil/gofun/store"
)

func New(fs *store.FeatureStore) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /features/{entity_type}/{entity_id}", handleGet(fs))
	mux.HandleFunc("POST /features/{entity_type}/{entity_id}", handleSet(fs))
	mux.HandleFunc("POST /features/batch", handleBatch(fs))

	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleGet(fs *store.FeatureStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entityType := r.PathValue("entity_type")
		entityID := r.PathValue("entity_id")

		features, ok := fs.Get(entityType, entityID)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(features)
	}
}

func handleSet(fs *store.FeatureStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entityType := r.PathValue("entity_type")
		entityID := r.PathValue("entity_id")

		var features store.FeatureVector
		if err := json.NewDecoder(r.Body).Decode(&features); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		fs.Set(entityType, entityID, features)
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(features)
	}
}

type batchRequest struct {
	Keys []store.Key `json:"keys"`
}

func handleBatch(fs *store.FeatureStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req batchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		result := fs.GetBatch(req.Keys)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}
