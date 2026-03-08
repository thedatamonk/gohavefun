// feature/materializer_test.go
package feature

import (
	"context"
	"testing"
	"time"

	"github.com/rohil/gofun/store"
)

func seedRawEvents(fs *store.FeatureStore, id string) {
	fs.Set(EntityRawLogins, id, store.FeatureVector{
		"count_30d": 20, "avg_session_min": 30, "days_since_last": 2, "features_used_pct": 75,
	})
	fs.Set(EntityRawPayments, id, store.FeatureVector{
		"total": 1200, "late_count": 1, "monthly_avg": 40,
	})
	fs.Set(EntityRawTickets, id, store.FeatureVector{
		"count_90d": 3, "avg_resolution": 24, "escalations": 1,
	})
}

func TestMaterializeOnce(t *testing.T) {
	fs := store.NewFeatureStore()
	seedRawEvents(fs, "cust-0001")

	m := NewMaterializer(fs, []string{"cust-0001"})
	m.RunOnce()

	usage, ok := fs.Get(EntityUsage, "cust-0001")
	if !ok {
		t.Fatal("expected usage_metrics to be materialized")
	}
	if usage[LoginsLast30d] != 20 {
		t.Fatalf("expected logins_last_30d=20, got %f", usage[LoginsLast30d])
	}
	if usage[DaysSinceLastLogin] != 2 {
		t.Fatalf("expected days_since_last_login=2, got %f", usage[DaysSinceLastLogin])
	}

	billing, ok := fs.Get(EntityBilling, "cust-0001")
	if !ok {
		t.Fatal("expected billing to be materialized")
	}
	if billing[TotalSpend] != 1200 {
		t.Fatalf("expected total_spend=1200, got %f", billing[TotalSpend])
	}

	support, ok := fs.Get(EntitySupport, "cust-0001")
	if !ok {
		t.Fatal("expected support to be materialized")
	}
	if support[EscalationCount] != 1 {
		t.Fatalf("expected escalation_count=1, got %f", support[EscalationCount])
	}
}

func TestMaterializerStartStop(t *testing.T) {
	fs := store.NewFeatureStore()
	seedRawEvents(fs, "cust-0001")

	m := NewMaterializer(fs, []string{"cust-0001"})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		m.Start(ctx, 50*time.Millisecond)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("materializer did not stop after context cancellation")
	}

	_, ok := fs.Get(EntityUsage, "cust-0001")
	if !ok {
		t.Fatal("expected usage_metrics after materializer ran")
	}
}
