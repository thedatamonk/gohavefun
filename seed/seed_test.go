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

	// Verify all customers have raw events
	for _, id := range ids {
		if _, ok := fs.Get(feature.EntityRawLogins, id); !ok {
			t.Fatalf("customer %s missing raw_logins", id)
		}
		if _, ok := fs.Get(feature.EntityRawPayments, id); !ok {
			t.Fatalf("customer %s missing raw_payments", id)
		}
		if _, ok := fs.Get(feature.EntityRawTickets, id); !ok {
			t.Fatalf("customer %s missing raw_tickets", id)
		}
	}
}

func TestGenerateDeterministic(t *testing.T) {
	fs1 := store.NewFeatureStore()
	fs2 := store.NewFeatureStore()
	ids1 := Generate(fs1, 10)
	ids2 := Generate(fs2, 10)

	// Compare actual feature values, not just IDs (which are index-based)
	for i := range ids1 {
		p1, _ := fs1.Get(feature.EntityProfile, ids1[i])
		p2, _ := fs2.Get(feature.EntityProfile, ids2[i])
		if p1[feature.TenureMonths] != p2[feature.TenureMonths] {
			t.Fatalf("non-deterministic: customer %d tenure differs: %f vs %f",
				i, p1[feature.TenureMonths], p2[feature.TenureMonths])
		}
		if p1[feature.MonthlyCharge] != p2[feature.MonthlyCharge] {
			t.Fatalf("non-deterministic: customer %d monthly_charge differs: %f vs %f",
				i, p1[feature.MonthlyCharge], p2[feature.MonthlyCharge])
		}
	}
}
