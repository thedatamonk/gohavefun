// scoring/scorer_test.go
package scoring

import (
	"math"
	"testing"

	"github.com/rohil/gofun/store"
)

func TestSigmoid(t *testing.T) {
	if got := sigmoid(0); math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("sigmoid(0) should be 0.5, got %f", got)
	}
	if got := sigmoid(100); got < 0.99 {
		t.Fatalf("sigmoid(100) should be ~1.0, got %f", got)
	}
	if got := sigmoid(-100); got > 0.01 {
		t.Fatalf("sigmoid(-100) should be ~0.0, got %f", got)
	}
}

func TestPredictLoyalCustomer(t *testing.T) {
	features := map[string]store.FeatureVector{
		"customer_profile": {"tenure_months": 60, "plan_tier": 3, "monthly_charge": 29.99},
		"usage_metrics":    {"logins_last_30d": 25, "avg_session_minutes": 45, "days_since_last_login": 1, "feature_adoption_pct": 85},
		"billing":          {"total_spend": 1800, "late_payments_count": 0, "avg_monthly_spend": 30},
		"support":          {"tickets_last_90d": 0, "avg_resolution_hours": 0, "escalation_count": 0},
	}
	result := Predict("loyal-1", features)
	if result.ChurnProbability >= 0.3 {
		t.Fatalf("loyal customer should have low churn, got %f", result.ChurnProbability)
	}
	if result.RiskLevel != "low" {
		t.Fatalf("expected risk_level=low, got %s", result.RiskLevel)
	}
}

func TestPredictAtRiskCustomer(t *testing.T) {
	features := map[string]store.FeatureVector{
		"customer_profile": {"tenure_months": 3, "plan_tier": 1, "monthly_charge": 49.99},
		"usage_metrics":    {"logins_last_30d": 1, "avg_session_minutes": 2, "days_since_last_login": 25, "feature_adoption_pct": 10},
		"billing":          {"total_spend": 150, "late_payments_count": 3, "avg_monthly_spend": 50},
		"support":          {"tickets_last_90d": 5, "avg_resolution_hours": 72, "escalation_count": 2},
	}
	result := Predict("atrisk-1", features)
	if result.ChurnProbability <= 0.7 {
		t.Fatalf("at-risk customer should have high churn, got %f", result.ChurnProbability)
	}
	if result.RiskLevel != "high" {
		t.Fatalf("expected risk_level=high, got %s", result.RiskLevel)
	}
}

func TestTopRiskFactors(t *testing.T) {
	features := map[string]store.FeatureVector{
		"customer_profile": {"tenure_months": 3, "plan_tier": 1, "monthly_charge": 49.99},
		"usage_metrics":    {"logins_last_30d": 1, "avg_session_minutes": 2, "days_since_last_login": 25, "feature_adoption_pct": 10},
		"billing":          {"total_spend": 150, "late_payments_count": 3, "avg_monthly_spend": 50},
		"support":          {"tickets_last_90d": 5, "avg_resolution_hours": 72, "escalation_count": 2},
	}
	result := Predict("test", features)
	if len(result.TopRiskFactors) == 0 {
		t.Fatal("expected at least one risk factor")
	}
	for _, f := range result.TopRiskFactors {
		if f.Contribution <= 0 {
			t.Fatalf("risk factor %q should have positive contribution, got %f", f.Feature, f.Contribution)
		}
	}
}
