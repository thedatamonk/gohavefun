// seed/seed.go
package seed

import (
	"fmt"
	"math/rand"

	"github.com/rohil/gofun/feature"
	"github.com/rohil/gofun/registry"
	"github.com/rohil/gofun/store"
)

func SeedRegistry(reg *registry.Registry) {
	views := []registry.FeatureViewDef{
		{
			Name: "customer_profile", EntityType: "customer",
			Description: "Customer profile features", Owner: "growth-team",
			Features: []registry.FeatureDef{
				{Name: "tenure_months", Dtype: "float64", Description: "Months since signup"},
				{Name: "plan_tier", Dtype: "float64", Description: "1=basic, 2=pro, 3=enterprise"},
				{Name: "monthly_charge", Dtype: "float64", Description: "Monthly subscription price"},
			},
			Tags: map[string]string{"group": "profile"},
		},
		{
			Name: "usage_metrics", EntityType: "customer",
			Description: "Usage-derived metrics", Owner: "product-team",
			Features: []registry.FeatureDef{
				{Name: "logins_last_30d", Dtype: "float64", Description: "Login count in last 30 days"},
				{Name: "avg_session_minutes", Dtype: "float64", Description: "Average session length"},
				{Name: "days_since_last_login", Dtype: "float64", Description: "Days since last activity"},
				{Name: "feature_adoption_pct", Dtype: "float64", Description: "Percent of features used"},
			},
			Tags: map[string]string{"group": "usage"},
		},
		{
			Name: "billing", EntityType: "customer",
			Description: "Billing-derived metrics", Owner: "finance-team",
			Features: []registry.FeatureDef{
				{Name: "total_spend", Dtype: "float64", Description: "Lifetime spend"},
				{Name: "late_payments_count", Dtype: "float64", Description: "Number of late payments"},
				{Name: "avg_monthly_spend", Dtype: "float64", Description: "Average monthly billing"},
			},
			Tags: map[string]string{"group": "billing"},
		},
		{
			Name: "support", EntityType: "customer",
			Description: "Support interaction metrics", Owner: "support-team",
			Features: []registry.FeatureDef{
				{Name: "tickets_last_90d", Dtype: "float64", Description: "Tickets in last 90 days"},
				{Name: "avg_resolution_hours", Dtype: "float64", Description: "Average ticket resolution time"},
				{Name: "escalation_count", Dtype: "float64", Description: "Number of escalated tickets"},
			},
			Tags: map[string]string{"group": "support"},
		},
		{
			Name: "raw_logins", EntityType: "customer",
			Description: "Raw login event data", Owner: "platform-team",
			Features: []registry.FeatureDef{
				{Name: "count_30d", Dtype: "float64"},
				{Name: "avg_session_min", Dtype: "float64"},
				{Name: "days_since_last", Dtype: "float64"},
				{Name: "features_used_pct", Dtype: "float64"},
			},
			Tags: map[string]string{"group": "raw"},
		},
		{
			Name: "raw_payments", EntityType: "customer",
			Description: "Raw payment event data", Owner: "platform-team",
			Features: []registry.FeatureDef{
				{Name: "total", Dtype: "float64"},
				{Name: "late_count", Dtype: "float64"},
				{Name: "monthly_avg", Dtype: "float64"},
			},
			Tags: map[string]string{"group": "raw"},
		},
		{
			Name: "raw_tickets", EntityType: "customer",
			Description: "Raw support ticket data", Owner: "platform-team",
			Features: []registry.FeatureDef{
				{Name: "count_90d", Dtype: "float64"},
				{Name: "avg_resolution", Dtype: "float64"},
				{Name: "escalations", Dtype: "float64"},
			},
			Tags: map[string]string{"group": "raw"},
		},
	}

	for _, v := range views {
		reg.Create(v) // ignore error if already exists
	}
}

func Generate(fs store.Store, n int) []string {
	rng := rand.New(rand.NewSource(42))
	ids := make([]string, n)

	for i := range n {
		id := fmt.Sprintf("cust-%04d", i+1)
		ids[i] = id

		persona := rng.Float64()

		var profile store.FeatureVector
		var rawLogins, rawPayments, rawTickets store.FeatureVector

		switch {
		case persona < 0.3: // Loyal
			profile = store.FeatureVector{
				feature.TenureMonths:  float64(24 + rng.Intn(60)),
				feature.PlanTier:      float64(2 + rng.Intn(2)),
				feature.MonthlyCharge: 19.99 + rng.Float64()*30,
			}
			rawLogins = store.FeatureVector{
				"count_30d":         float64(15 + rng.Intn(20)),
				"avg_session_min":   float64(20 + rng.Intn(40)),
				"days_since_last":   float64(rng.Intn(3)),
				"features_used_pct": float64(60 + rng.Intn(40)),
			}
			rawPayments = store.FeatureVector{
				"total":       float64(500 + rng.Intn(3000)),
				"late_count":  0,
				"monthly_avg": float64(20 + rng.Intn(30)),
			}
			rawTickets = store.FeatureVector{
				"count_90d":      float64(rng.Intn(2)),
				"avg_resolution": float64(rng.Intn(12)),
				"escalations":    0,
			}

		case persona < 0.7: // Mixed
			profile = store.FeatureVector{
				feature.TenureMonths:  float64(6 + rng.Intn(30)),
				feature.PlanTier:      float64(1 + rng.Intn(3)),
				feature.MonthlyCharge: 14.99 + rng.Float64()*50,
			}
			rawLogins = store.FeatureVector{
				"count_30d":         float64(5 + rng.Intn(20)),
				"avg_session_min":   float64(5 + rng.Intn(30)),
				"days_since_last":   float64(rng.Intn(15)),
				"features_used_pct": float64(20 + rng.Intn(60)),
			}
			rawPayments = store.FeatureVector{
				"total":       float64(100 + rng.Intn(1500)),
				"late_count":  float64(rng.Intn(3)),
				"monthly_avg": float64(15 + rng.Intn(40)),
			}
			rawTickets = store.FeatureVector{
				"count_90d":      float64(rng.Intn(5)),
				"avg_resolution": float64(rng.Intn(48)),
				"escalations":    float64(rng.Intn(2)),
			}

		default: // At-risk
			profile = store.FeatureVector{
				feature.TenureMonths:  float64(1 + rng.Intn(12)),
				feature.PlanTier:      1,
				feature.MonthlyCharge: 29.99 + rng.Float64()*40,
			}
			rawLogins = store.FeatureVector{
				"count_30d":         float64(rng.Intn(5)),
				"avg_session_min":   float64(rng.Intn(10)),
				"days_since_last":   float64(10 + rng.Intn(25)),
				"features_used_pct": float64(rng.Intn(25)),
			}
			rawPayments = store.FeatureVector{
				"total":       float64(50 + rng.Intn(500)),
				"late_count":  float64(2 + rng.Intn(5)),
				"monthly_avg": float64(30 + rng.Intn(40)),
			}
			rawTickets = store.FeatureVector{
				"count_90d":      float64(3 + rng.Intn(8)),
				"avg_resolution": float64(24 + rng.Intn(72)),
				"escalations":    float64(1 + rng.Intn(4)),
			}
		}

		fs.Set(feature.EntityProfile, id, profile)
		fs.Set(feature.EntityRawLogins, id, rawLogins)
		fs.Set(feature.EntityRawPayments, id, rawPayments)
		fs.Set(feature.EntityRawTickets, id, rawTickets)
	}

	return ids
}
