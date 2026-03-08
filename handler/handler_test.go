package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rohil/gofun/store"
)

func newTestStore() *store.FeatureStore {
	fs := store.NewFeatureStore()
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
