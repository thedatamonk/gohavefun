// scoring/scorer_test.go
package scoring

import (
	"math"
	"os"
	"path/filepath"
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

func TestScorerFallback(t *testing.T) {
	// No model file -> should use logistic regression fallback
	s := NewScorer("/nonexistent/path")

	features := map[string]store.FeatureVector{
		"customer_profile": {"tenure_months": 24, "plan_tier": 2, "monthly_charge": 29.99},
		"usage_metrics":    {"logins_last_30d": 20, "avg_session_minutes": 30, "days_since_last_login": 1, "feature_adoption_pct": 80},
		"billing":          {"total_spend": 1000, "late_payments_count": 0, "avg_monthly_spend": 30},
		"support":          {"tickets_last_90d": 0, "avg_resolution_hours": 0, "escalation_count": 0},
	}

	result := s.Predict("cust-0001", features)

	if result.CustomerID != "cust-0001" {
		t.Errorf("expected customer_id cust-0001, got %s", result.CustomerID)
	}
	if result.RiskLevel == "" {
		t.Error("expected non-empty risk_level")
	}
	if result.ChurnProbability < 0 || result.ChurnProbability > 1 {
		t.Errorf("churn_probability out of range: %f", result.ChurnProbability)
	}
}

func TestScorerWithModel(t *testing.T) {
	// Create a minimal model file
	modelJSON := `{
		"learner": {
			"gradient_booster": {
				"model": {
					"trees": [
						{
							"id": 0,
							"tree_param": {"num_nodes": "3"},
							"left_children":  [1, -1, -1],
							"right_children": [2, -1, -1],
							"split_indices":  [0, 0, 0],
							"split_conditions": [10.0, 0.0, 0.0],
							"default_left":   [1, 0, 0],
							"base_weights":   [0.0, -0.5, 0.3]
						}
					]
				}
			},
			"learner_model_param": {
				"base_score": "0.000000e+00",
				"num_feature": "13"
			}
		}
	}`

	metaJSON := `{"feature_names": ["tenure_months","plan_tier","monthly_charge","logins_last_30d","avg_session_minutes","days_since_last_login","feature_adoption_pct","total_spend","late_payments_count","avg_monthly_spend","tickets_last_90d","avg_resolution_hours","escalation_count"]}`

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "model.json"), []byte(modelJSON), 0644)
	os.WriteFile(filepath.Join(dir, "meta.json"), []byte(metaJSON), 0644)

	s := NewScorer(dir)

	features := map[string]store.FeatureVector{
		"customer_profile": {"tenure_months": 5, "plan_tier": 1, "monthly_charge": 49.99},
		"usage_metrics":    {"logins_last_30d": 2, "avg_session_minutes": 5, "days_since_last_login": 20, "feature_adoption_pct": 10},
		"billing":          {"total_spend": 100, "late_payments_count": 3, "avg_monthly_spend": 50},
		"support":          {"tickets_last_90d": 5, "avg_resolution_hours": 48, "escalation_count": 3},
	}

	result := s.Predict("cust-0050", features)

	if result.CustomerID != "cust-0050" {
		t.Errorf("expected customer_id cust-0050, got %s", result.CustomerID)
	}
	// tenure_months=5 < 10.0 -> left -> leaf -0.5 -> sigmoid(-0.5) ~ 0.378
	if result.ChurnProbability < 0.37 || result.ChurnProbability > 0.39 {
		t.Errorf("expected ~0.378, got %f", result.ChurnProbability)
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
