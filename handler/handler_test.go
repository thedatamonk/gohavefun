package handler

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rohil/gofun/registry"
	"github.com/rohil/gofun/scoring"
	"github.com/rohil/gofun/store"
)

func newTestRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	dir := t.TempDir()
	r, err := registry.NewSQLiteRegistry(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

func newTestStore() *store.MemoryStore {
	fs := store.NewMemoryStore()
	fs.Set("user", "123", store.FeatureVector{"age": 25, "score": 0.85})
	return fs
}

func TestHealth(t *testing.T) {
	reg := newTestRegistry(t)
	h := New(newTestStore(), reg, scoring.NewScorer(""))
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestGetFeature(t *testing.T) {
	reg := newTestRegistry(t)
	h := New(newTestStore(), reg, scoring.NewScorer(""))
	req := httptest.NewRequest("GET", "/features/user/123", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"age"`) {
		t.Fatalf("expected age in response: %s", w.Body.String())
	}
}

func TestGetFeatureNotFound(t *testing.T) {
	reg := newTestRegistry(t)
	h := New(newTestStore(), reg, scoring.NewScorer(""))
	req := httptest.NewRequest("GET", "/features/user/999", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSetFeature(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Create(registry.FeatureViewDef{
		Name: "user", EntityType: "user",
		Features: []registry.FeatureDef{
			{Name: "age", Dtype: "float64"},
			{Name: "score", Dtype: "float64"},
		},
	})
	h := New(newTestStore(), reg, scoring.NewScorer(""))
	body := strings.NewReader(`{"age": 30, "score": 0.95}`)
	req := httptest.NewRequest("POST", "/features/user/456", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it was stored
	req2 := httptest.NewRequest("GET", "/features/user/456", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 on re-read, got %d", w2.Code)
	}
}

func TestBatch(t *testing.T) {
	fs := newTestStore()
	fs.Set("item", "abc", store.FeatureVector{"price": 9.99})
	reg := newTestRegistry(t)
	h := New(fs, reg, scoring.NewScorer(""))

	body := strings.NewReader(`{"keys":[{"entity_type":"user","entity_id":"123"},{"entity_type":"item","entity_id":"abc"},{"entity_type":"user","entity_id":"missing"}]}`)
	req := httptest.NewRequest("POST", "/features/batch", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := w.Body.String()
	if !strings.Contains(resp, "user:123") || !strings.Contains(resp, "item:abc") {
		t.Fatalf("expected both keys in response: %s", resp)
	}
	if strings.Contains(resp, "missing") {
		t.Fatalf("did not expect missing key in response: %s", resp)
	}
}

func TestSetInvalidJSON(t *testing.T) {
	reg := newTestRegistry(t)
	h := New(newTestStore(), reg, scoring.NewScorer(""))
	body := strings.NewReader(`not json`)
	req := httptest.NewRequest("POST", "/features/user/456", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func setupChurnStore() *store.MemoryStore {
	fs := store.NewMemoryStore()
	fs.Set("customer_profile", "cust-0001", store.FeatureVector{
		"tenure_months": 60, "plan_tier": 3, "monthly_charge": 29.99,
	})
	fs.Set("usage_metrics", "cust-0001", store.FeatureVector{
		"logins_last_30d": 25, "avg_session_minutes": 45,
		"days_since_last_login": 1, "feature_adoption_pct": 85,
	})
	fs.Set("billing", "cust-0001", store.FeatureVector{
		"total_spend": 1800, "late_payments_count": 0, "avg_monthly_spend": 30,
	})
	fs.Set("support", "cust-0001", store.FeatureVector{
		"tickets_last_90d": 0, "avg_resolution_hours": 0, "escalation_count": 0,
	})
	return fs
}

func TestPredictEndpoint(t *testing.T) {
	reg := newTestRegistry(t)
	h := New(setupChurnStore(), reg, scoring.NewScorer(""))
	req := httptest.NewRequest("GET", "/predict/cust-0001", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "churn_probability") {
		t.Fatalf("expected churn_probability in response: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "risk_level") {
		t.Fatalf("expected risk_level in response: %s", w.Body.String())
	}
}

func TestPredictNotFound(t *testing.T) {
	reg := newTestRegistry(t)
	h := New(setupChurnStore(), reg, scoring.NewScorer(""))
	req := httptest.NewRequest("GET", "/predict/nonexistent", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCustomerFeaturesEndpoint(t *testing.T) {
	reg := newTestRegistry(t)
	h := New(setupChurnStore(), reg, scoring.NewScorer(""))
	req := httptest.NewRequest("GET", "/customers/cust-0001/features", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	for _, group := range []string{"customer_profile", "usage_metrics", "billing", "support"} {
		if !strings.Contains(body, group) {
			t.Fatalf("expected %s in response: %s", group, body)
		}
	}
}

func TestCustomerFeaturesNotFound(t *testing.T) {
	reg := newTestRegistry(t)
	h := New(setupChurnStore(), reg, scoring.NewScorer(""))
	req := httptest.NewRequest("GET", "/customers/nonexistent/features", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- Registry endpoint tests ---

func TestRegistryCreateAndGet(t *testing.T) {
	reg := newTestRegistry(t)
	h := New(newTestStore(), reg, scoring.NewScorer(""))

	// Create
	body := strings.NewReader(`{"name":"orders","entity_type":"order","features":[{"name":"total","dtype":"float64"}]}`)
	req := httptest.NewRequest("POST", "/registry/feature-views", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Get
	req2 := httptest.NewRequest("GET", "/registry/feature-views/orders", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), `"orders"`) {
		t.Fatalf("expected orders in response: %s", w2.Body.String())
	}
}

func TestRegistryList(t *testing.T) {
	reg := newTestRegistry(t)
	h := New(newTestStore(), reg, scoring.NewScorer(""))

	for _, name := range []string{"view_a", "view_b"} {
		body := strings.NewReader(`{"name":"` + name + `","entity_type":"t","features":[{"name":"f","dtype":"float64"}]}`)
		req := httptest.NewRequest("POST", "/registry/feature-views", body)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 creating %s, got %d", name, w.Code)
		}
	}

	req := httptest.NewRequest("GET", "/registry/feature-views", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := w.Body.String()
	if !strings.Contains(resp, "view_a") || !strings.Contains(resp, "view_b") {
		t.Fatalf("expected both views in response: %s", resp)
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	reg := newTestRegistry(t)
	h := New(newTestStore(), reg, scoring.NewScorer(""))

	req := httptest.NewRequest("GET", "/registry/feature-views/nonexistent", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestRegistryUpdate(t *testing.T) {
	reg := newTestRegistry(t)
	h := New(newTestStore(), reg, scoring.NewScorer(""))

	// Create
	body := strings.NewReader(`{"name":"v1","entity_type":"e","features":[{"name":"f1","dtype":"float64"}]}`)
	req := httptest.NewRequest("POST", "/registry/feature-views", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	// Update
	body2 := strings.NewReader(`{"name":"v1","entity_type":"e","features":[{"name":"f1","dtype":"float64"},{"name":"f2","dtype":"int64"}]}`)
	req2 := httptest.NewRequest("PUT", "/registry/feature-views/v1", body2)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// Verify
	req3 := httptest.NewRequest("GET", "/registry/feature-views/v1", nil)
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, req3)
	if !strings.Contains(w3.Body.String(), "f2") {
		t.Fatalf("expected f2 in updated view: %s", w3.Body.String())
	}
}

func TestRegistryDelete(t *testing.T) {
	reg := newTestRegistry(t)
	h := New(newTestStore(), reg, scoring.NewScorer(""))

	// Create
	body := strings.NewReader(`{"name":"del_me","entity_type":"e","features":[{"name":"f","dtype":"float64"}]}`)
	req := httptest.NewRequest("POST", "/registry/feature-views", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	// Delete
	req2 := httptest.NewRequest("DELETE", "/registry/feature-views/del_me", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w2.Code)
	}

	// Verify gone
	req3 := httptest.NewRequest("GET", "/registry/feature-views/del_me", nil)
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, req3)
	if w3.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w3.Code)
	}
}

func TestFeatureWriteValidation(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Create(registry.FeatureViewDef{
		Name: "user", EntityType: "user",
		Features: []registry.FeatureDef{
			{Name: "age", Dtype: "float64"},
			{Name: "score", Dtype: "float64"},
		},
	})
	h := New(newTestStore(), reg, scoring.NewScorer(""))

	// Valid write
	body := strings.NewReader(`{"age": 30, "score": 0.9}`)
	req := httptest.NewRequest("POST", "/features/user/789", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for valid write, got %d: %s", w.Code, w.Body.String())
	}

	// Unknown feature
	body2 := strings.NewReader(`{"age": 30, "unknown_feat": 1}`)
	req2 := httptest.NewRequest("POST", "/features/user/789", body2)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown feature, got %d: %s", w2.Code, w2.Body.String())
	}

	// Unknown view
	body3 := strings.NewReader(`{"x": 1}`)
	req3 := httptest.NewRequest("POST", "/features/nonexistent/789", body3)
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, req3)
	if w3.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown view, got %d: %s", w3.Code, w3.Body.String())
	}
}
