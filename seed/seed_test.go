// seed/seed_test.go
package seed

import (
	"testing"

	"github.com/rohil/gofun/feature"
	"github.com/rohil/gofun/store"
)

func TestGenerate(t *testing.T) {
	fs := store.NewFeatureStore()
	ids := Generate(fs, 75)

	if len(ids) != 75 {
		t.Fatalf("expected 75 customer IDs, got %d", len(ids))
	}

	for _, id := range ids {
		profile, ok := fs.Get(feature.EntityProfile, id)
		if !ok {
			t.Fatalf("customer %s missing profile", id)
		}
		if profile[feature.TenureMonths] == 0 && profile[feature.PlanTier] == 0 {
			t.Fatalf("customer %s has empty profile", id)
		}
	}

	_, ok := fs.Get(feature.EntityRawLogins, ids[0])
	if !ok {
		t.Fatalf("customer %s missing raw_logins", ids[0])
	}
	_, ok = fs.Get(feature.EntityRawPayments, ids[0])
	if !ok {
		t.Fatalf("customer %s missing raw_payments", ids[0])
	}
	_, ok = fs.Get(feature.EntityRawTickets, ids[0])
	if !ok {
		t.Fatalf("customer %s missing raw_tickets", ids[0])
	}
}

func TestGenerateDeterministic(t *testing.T) {
	fs1 := store.NewFeatureStore()
	fs2 := store.NewFeatureStore()
	ids1 := Generate(fs1, 10)
	ids2 := Generate(fs2, 10)

	for i := range ids1 {
		if ids1[i] != ids2[i] {
			t.Fatalf("expected deterministic output, got %s vs %s at index %d", ids1[i], ids2[i], i)
		}
	}
}
