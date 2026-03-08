package store

import (
	"sync"
	"testing"
)

func TestSetAndGet(t *testing.T) {
	fs := NewMemoryStore()
	fs.Set("user", "1", FeatureVector{"age": 25})

	got, ok := fs.Get("user", "1")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got["age"] != 25 {
		t.Fatalf("expected age=25, got %v", got["age"])
	}
}

func TestGetMissing(t *testing.T) {
	fs := NewMemoryStore()
	_, ok := fs.Get("user", "999")
	if ok {
		t.Fatal("expected key to be missing")
	}
}

func TestSetOverwrite(t *testing.T) {
	fs := NewMemoryStore()
	fs.Set("user", "1", FeatureVector{"age": 25})
	fs.Set("user", "1", FeatureVector{"age": 30})

	got, _ := fs.Get("user", "1")
	if got["age"] != 30 {
		t.Fatalf("expected age=30 after overwrite, got %v", got["age"])
	}
}

func TestGetBatch(t *testing.T) {
	fs := NewMemoryStore()
	fs.Set("user", "1", FeatureVector{"age": 25})
	fs.Set("user", "2", FeatureVector{"age": 30})

	result := fs.GetBatch([]Key{
		{EntityType: "user", EntityID: "1"},
		{EntityType: "user", EntityID: "2"},
		{EntityType: "user", EntityID: "999"},
	})

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	if result["user:1"]["age"] != 25 {
		t.Fatalf("expected user:1 age=25, got %v", result["user:1"]["age"])
	}
}

func TestConcurrentAccess(t *testing.T) {
	fs := NewMemoryStore()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			fs.Set("user", "concurrent", FeatureVector{"val": float64(n)})
		}(i)
	}

	// Concurrent reads alongside writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fs.Get("user", "concurrent")
		}()
	}

	wg.Wait()

	// Verify the key exists with some value
	_, ok := fs.Get("user", "concurrent")
	if !ok {
		t.Fatal("expected key to exist after concurrent writes")
	}
}
