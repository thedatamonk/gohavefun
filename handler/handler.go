package handler

import (
	"encoding/json"
	"net/http"

	"github.com/rohil/gofun/feature"
	"github.com/rohil/gofun/registry"
	"github.com/rohil/gofun/scoring"
	"github.com/rohil/gofun/store"
)

func New(fs store.Store, reg *registry.Registry, scorer *scoring.Scorer) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /features/{entity_type}/{entity_id}", handleGet(fs))
	mux.HandleFunc("POST /features/{entity_type}/{entity_id}", handleSet(fs, reg))
	mux.HandleFunc("POST /features/batch", handleBatch(fs))
	mux.HandleFunc("GET /predict/{customer_id}", handlePredict(fs, scorer))
	mux.HandleFunc("GET /customers/{customer_id}/features", handleCustomerFeatures(fs))

	mux.HandleFunc("GET /registry/feature-views", handleListViews(reg))
	mux.HandleFunc("GET /registry/feature-views/{name}", handleGetView(reg))
	mux.HandleFunc("POST /registry/feature-views", handleCreateView(reg))
	mux.HandleFunc("PUT /registry/feature-views/{name}", handleUpdateView(reg))
	mux.HandleFunc("DELETE /registry/feature-views/{name}", handleDeleteView(reg))

	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleGet(fs store.Store) http.HandlerFunc {
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

func handleSet(fs store.Store, reg *registry.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entityType := r.PathValue("entity_type")
		entityID := r.PathValue("entity_id")

		var features store.FeatureVector
		if err := json.NewDecoder(r.Body).Decode(&features); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		featureNames := make([]string, 0, len(features))
		for k := range features {
			featureNames = append(featureNames, k)
		}
		if err := reg.ValidateFeatures(entityType, featureNames); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		fs.Set(entityType, entityID, features)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(features)
	}
}

type batchRequest struct {
	Keys []store.Key `json:"keys"`
}

func handleBatch(fs store.Store) http.HandlerFunc {
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

func gatherFeatureGroups(fs store.Store, customerID string) (map[string]store.FeatureVector, bool) {
	groups := make(map[string]store.FeatureVector)
	groupNames := []string{
		feature.EntityProfile, feature.EntityUsage,
		feature.EntityBilling, feature.EntitySupport,
	}
	found := false
	for _, g := range groupNames {
		if fv, ok := fs.Get(g, customerID); ok {
			groups[g] = fv
			found = true
		}
	}
	return groups, found
}

func handlePredict(fs store.Store, scorer *scoring.Scorer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		customerID := r.PathValue("customer_id")
		groups, found := gatherFeatureGroups(fs, customerID)
		if !found {
			http.Error(w, `{"error":"customer not found"}`, http.StatusNotFound)
			return
		}
		result := scorer.Predict(customerID, groups)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func handleCustomerFeatures(fs store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		customerID := r.PathValue("customer_id")
		groups, found := gatherFeatureGroups(fs, customerID)
		if !found {
			http.Error(w, `{"error":"customer not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(groups)
	}
}

func handleListViews(reg *registry.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		views := reg.List()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(views)
	}
}

func handleGetView(reg *registry.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		view, ok := reg.Get(name)
		if !ok {
			http.Error(w, `{"error":"feature view not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(view)
	}
}

func handleCreateView(reg *registry.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var view registry.FeatureViewDef
		if err := json.NewDecoder(r.Body).Decode(&view); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}
		if err := reg.Create(view); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(view)
	}
}

func handleUpdateView(reg *registry.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		var view registry.FeatureViewDef
		if err := json.NewDecoder(r.Body).Decode(&view); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}
		if err := reg.Update(name, view); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(view)
	}
}

func handleDeleteView(reg *registry.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if err := reg.Delete(name); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
