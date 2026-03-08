package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rohil/gofun/store"
)

func newTestStore() *store.MemoryStore {
	fs := store.NewMemoryStore()
	fs.Set("user", "123", store.FeatureVector{"age": 25, "score": 0.85})
	return fs
}

func TestHealth(t *testing.T) {
	h := New(newTestStore())
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
	h := New(newTestStore())
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
	h := New(newTestStore())
	req := httptest.NewRequest("GET", "/features/user/999", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSetFeature(t *testing.T) {
	h := New(newTestStore())
	body := strings.NewReader(`{"age": 30, "score": 0.95}`)
	req := httptest.NewRequest("POST", "/features/user/456", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
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
	h := New(fs)

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
	h := New(newTestStore())
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
	h := New(setupChurnStore())
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
	h := New(setupChurnStore())
	req := httptest.NewRequest("GET", "/predict/nonexistent", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCustomerFeaturesEndpoint(t *testing.T) {
	h := New(setupChurnStore())
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
	h := New(setupChurnStore())
	req := httptest.NewRequest("GET", "/customers/nonexistent/features", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
