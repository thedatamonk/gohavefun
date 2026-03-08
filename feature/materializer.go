// feature/materializer.go
package feature

import (
	"context"
	"fmt"
	"time"

	"github.com/rohil/gofun/store"
)

type Materializer struct {
	store       *store.FeatureStore
	customerIDs []string
}

func NewMaterializer(fs *store.FeatureStore, customerIDs []string) *Materializer {
	return &Materializer{store: fs, customerIDs: customerIDs}
}

func (m *Materializer) RunOnce() {
	for _, id := range m.customerIDs {
		m.materializeCustomer(id)
	}
}

func (m *Materializer) materializeCustomer(id string) {
	if raw, ok := m.store.Get(EntityRawLogins, id); ok {
		m.store.Set(EntityUsage, id, store.FeatureVector{
			LoginsLast30d:      raw["count_30d"],
			AvgSessionMinutes:  raw["avg_session_min"],
			DaysSinceLastLogin: raw["days_since_last"],
			FeatureAdoptionPct: raw["features_used_pct"],
		})
	}

	if raw, ok := m.store.Get(EntityRawPayments, id); ok {
		m.store.Set(EntityBilling, id, store.FeatureVector{
			TotalSpend:        raw["total"],
			LatePaymentsCount: raw["late_count"],
			AvgMonthlySpend:   raw["monthly_avg"],
		})
	}

	if raw, ok := m.store.Get(EntityRawTickets, id); ok {
		m.store.Set(EntitySupport, id, store.FeatureVector{
			TicketsLast90d:     raw["count_90d"],
			AvgResolutionHours: raw["avg_resolution"],
			EscalationCount:    raw["escalations"],
		})
	}
}

func (m *Materializer) Start(ctx context.Context, interval time.Duration) {
	fmt.Printf("Materializer started (interval: %s, customers: %d)\n", interval, len(m.customerIDs))
	m.RunOnce()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Materializer stopped.")
			return
		case <-ticker.C:
			m.RunOnce()
		}
	}
}
