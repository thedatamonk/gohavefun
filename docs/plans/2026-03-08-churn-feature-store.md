# Churn Feature Store Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extend the generic Go feature store into a customer churn prediction system with materialization pipeline and scoring endpoint.

**Architecture:** Raw events are seeded at startup, a background materializer goroutine computes derived features every 10s, and HTTP endpoints serve feature lookups and churn predictions using logistic regression.

**Tech Stack:** Go standard library only (net/http, sync, math, math/rand, context, encoding/json)

---

### Task 1: Feature Schema Package

**Files:**
- Create: `feature/schema.go`
- Test: `feature/schema_test.go`

**Step 1: Write the test**

```go
// feature/schema_test.go
package feature

import "testing"

func TestFeatureNames(t *testing.T) {
	// Verify all 13 features are defined
	if len(AllFeatureNames) != 13 {
		t.Fatalf("expected 13 features, got %d", len(AllFeatureNames))
	}
}

func TestWeights(t *testing.T) {
	// Verify weights exist for every feature
	for _, name := range AllFeatureNames {
		if _, ok := Weights[name]; !ok {
			t.Fatalf("missing weight for feature %q", name)
		}
	}
	// Verify bias exists
	if Bias == 0 {
		t.Fatal("bias should not be zero")
	}
}

func TestFeatureGroups(t *testing.T) {
	groups := []string{"customer_profile", "usage_metrics", "billing", "support"}
	for _, g := range groups {
		names, ok := FeatureGroups[g]
		if !ok {
			t.Fatalf("missing feature group %q", g)
		}
		if len(names) == 0 {
			t.Fatalf("feature group %q is empty", g)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./feature/... -v`
Expected: FAIL — package doesn't exist yet

**Step 3: Write implementation**

```go
// feature/schema.go
package feature

// Feature name constants
const (
	TenureMonths       = "tenure_months"
	PlanTier           = "plan_tier"
	MonthlyCharge      = "monthly_charge"
	LoginsLast30d      = "logins_last_30d"
	AvgSessionMinutes  = "avg_session_minutes"
	DaysSinceLastLogin = "days_since_last_login"
	FeatureAdoptionPct = "feature_adoption_pct"
	TotalSpend         = "total_spend"
	LatePaymentsCount  = "late_payments_count"
	AvgMonthlySpend    = "avg_monthly_spend"
	TicketsLast90d     = "tickets_last_90d"
	AvgResolutionHours = "avg_resolution_hours"
	EscalationCount    = "escalation_count"
)

var AllFeatureNames = []string{
	TenureMonths, PlanTier, MonthlyCharge,
	LoginsLast30d, AvgSessionMinutes, DaysSinceLastLogin, FeatureAdoptionPct,
	TotalSpend, LatePaymentsCount, AvgMonthlySpend,
	TicketsLast90d, AvgResolutionHours, EscalationCount,
}

var Weights = map[string]float64{
	TenureMonths:       -0.05,
	PlanTier:           -0.3,
	MonthlyCharge:      +0.02,
	LoginsLast30d:      -0.15,
	AvgSessionMinutes:  -0.03,
	DaysSinceLastLogin: +0.1,
	FeatureAdoptionPct: -0.02,
	TotalSpend:         -0.001,
	LatePaymentsCount:  +0.4,
	AvgMonthlySpend:    -0.01,
	TicketsLast90d:     +0.2,
	AvgResolutionHours: +0.05,
	EscalationCount:    +0.5,
}

const Bias = -0.5

// FeatureGroups maps group name → feature names in that group
var FeatureGroups = map[string][]string{
	"customer_profile": {TenureMonths, PlanTier, MonthlyCharge},
	"usage_metrics":    {LoginsLast30d, AvgSessionMinutes, DaysSinceLastLogin, FeatureAdoptionPct},
	"billing":          {TotalSpend, LatePaymentsCount, AvgMonthlySpend},
	"support":          {TicketsLast90d, AvgResolutionHours, EscalationCount},
}

// EntityType constants for store keys
const (
	EntityProfile = "customer_profile"
	EntityUsage   = "usage_metrics"
	EntityBilling = "billing"
	EntitySupport = "support"
	EntityRawLogins   = "raw_logins"
	EntityRawPayments = "raw_payments"
	EntityRawTickets  = "raw_tickets"
)
```

**Step 4: Run tests**

Run: `go test ./feature/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add feature/
git commit -m "feat: add feature schema with groups, weights, and constants"
```

---

### Task 2: Scoring Package

**Files:**
- Create: `scoring/scorer.go`
- Test: `scoring/scorer_test.go`

**Step 1: Write the test**

```go
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
	// Top factors should have positive contribution (pushing toward churn)
	for _, f := range result.TopRiskFactors {
		if f.Contribution <= 0 {
			t.Fatalf("risk factor %q should have positive contribution, got %f", f.Feature, f.Contribution)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./scoring/... -v`
Expected: FAIL — package doesn't exist

**Step 3: Write implementation**

```go
// scoring/scorer.go
package scoring

import (
	"math"
	"sort"

	"github.com/rohil/gofun/feature"
	"github.com/rohil/gofun/store"
)

type RiskFactor struct {
	Feature      string  `json:"feature"`
	Value        float64 `json:"value"`
	Weight       float64 `json:"weight"`
	Contribution float64 `json:"contribution"`
}

type PredictionResult struct {
	CustomerID       string       `json:"customer_id"`
	ChurnProbability float64      `json:"churn_probability"`
	RiskLevel        string       `json:"risk_level"`
	TopRiskFactors   []RiskFactor `json:"top_risk_factors"`
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// Predict takes all feature groups for a customer and returns a churn prediction.
func Predict(customerID string, featureGroups map[string]store.FeatureVector) PredictionResult {
	// Flatten all feature groups into one map
	allFeatures := make(map[string]float64)
	for _, fv := range featureGroups {
		for k, v := range fv {
			allFeatures[k] = v
		}
	}

	// Compute weighted sum
	sum := feature.Bias
	var factors []RiskFactor
	for _, name := range feature.AllFeatureNames {
		val := allFeatures[name]
		w := feature.Weights[name]
		contribution := w * val
		sum += contribution
		factors = append(factors, RiskFactor{
			Feature:      name,
			Value:        val,
			Weight:       w,
			Contribution: contribution,
		})
	}

	prob := sigmoid(sum)

	// Determine risk level
	riskLevel := "low"
	if prob >= 0.7 {
		riskLevel = "high"
	} else if prob >= 0.3 {
		riskLevel = "medium"
	}

	// Sort factors by contribution descending (most positive = most risk)
	sort.Slice(factors, func(i, j int) bool {
		return factors[i].Contribution > factors[j].Contribution
	})

	// Keep top 3 positive contributors
	var topFactors []RiskFactor
	for _, f := range factors {
		if f.Contribution > 0 && len(topFactors) < 3 {
			topFactors = append(topFactors, f)
		}
	}

	return PredictionResult{
		CustomerID:       customerID,
		ChurnProbability: math.Round(prob*1000) / 1000, // round to 3 decimals
		RiskLevel:        riskLevel,
		TopRiskFactors:   topFactors,
	}
}
```

**Step 4: Run tests**

Run: `go test ./scoring/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add scoring/
git commit -m "feat: add churn scoring with logistic regression and risk factors"
```

---

### Task 3: Seed Data Generator

**Files:**
- Create: `seed/seed.go`
- Test: `seed/seed_test.go`

**Step 1: Write the test**

```go
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

	// Verify each customer has profile data
	for _, id := range ids {
		profile, ok := fs.Get(feature.EntityProfile, id)
		if !ok {
			t.Fatalf("customer %s missing profile", id)
		}
		if profile[feature.TenureMonths] == 0 && profile[feature.PlanTier] == 0 {
			t.Fatalf("customer %s has empty profile", id)
		}
	}

	// Verify raw events exist for first customer
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

	// Same seed → same IDs
	for i := range ids1 {
		if ids1[i] != ids2[i] {
			t.Fatalf("expected deterministic output, got %s vs %s at index %d", ids1[i], ids2[i], i)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./seed/... -v`
Expected: FAIL

**Step 3: Write implementation**

```go
// seed/seed.go
package seed

import (
	"fmt"
	"math/rand"

	"github.com/rohil/gofun/feature"
	"github.com/rohil/gofun/store"
)

// Generate creates n customers with profile data and raw events.
// Returns the list of customer IDs. Uses a fixed seed for reproducibility.
func Generate(fs *store.FeatureStore, n int) []string {
	rng := rand.New(rand.NewSource(42))
	ids := make([]string, n)

	for i := range n {
		id := fmt.Sprintf("cust-%04d", i+1)
		ids[i] = id

		// Assign persona: ~30% loyal, ~40% mixed, ~30% at-risk
		persona := rng.Float64()

		var profile store.FeatureVector
		var rawLogins, rawPayments, rawTickets store.FeatureVector

		switch {
		case persona < 0.3: // Loyal
			profile = store.FeatureVector{
				feature.TenureMonths:  float64(24 + rng.Intn(60)),
				feature.PlanTier:      float64(2 + rng.Intn(2)), // 2 or 3
				feature.MonthlyCharge: 19.99 + rng.Float64()*30,
			}
			rawLogins = store.FeatureVector{
				"count_30d":          float64(15 + rng.Intn(20)),
				"avg_session_min":    float64(20 + rng.Intn(40)),
				"days_since_last":    float64(rng.Intn(3)),
				"features_used_pct":  float64(60 + rng.Intn(40)),
			}
			rawPayments = store.FeatureVector{
				"total":        float64(500 + rng.Intn(3000)),
				"late_count":   0,
				"monthly_avg":  float64(20 + rng.Intn(30)),
			}
			rawTickets = store.FeatureVector{
				"count_90d":     float64(rng.Intn(2)),
				"avg_resolution": float64(rng.Intn(12)),
				"escalations":   0,
			}

		case persona < 0.7: // Mixed
			profile = store.FeatureVector{
				feature.TenureMonths:  float64(6 + rng.Intn(30)),
				feature.PlanTier:      float64(1 + rng.Intn(3)), // 1, 2, or 3
				feature.MonthlyCharge: 14.99 + rng.Float64()*50,
			}
			rawLogins = store.FeatureVector{
				"count_30d":          float64(5 + rng.Intn(20)),
				"avg_session_min":    float64(5 + rng.Intn(30)),
				"days_since_last":    float64(rng.Intn(15)),
				"features_used_pct":  float64(20 + rng.Intn(60)),
			}
			rawPayments = store.FeatureVector{
				"total":        float64(100 + rng.Intn(1500)),
				"late_count":   float64(rng.Intn(3)),
				"monthly_avg":  float64(15 + rng.Intn(40)),
			}
			rawTickets = store.FeatureVector{
				"count_90d":     float64(rng.Intn(5)),
				"avg_resolution": float64(rng.Intn(48)),
				"escalations":   float64(rng.Intn(2)),
			}

		default: // At-risk
			profile = store.FeatureVector{
				feature.TenureMonths:  float64(1 + rng.Intn(12)),
				feature.PlanTier:      1,
				feature.MonthlyCharge: 29.99 + rng.Float64()*40,
			}
			rawLogins = store.FeatureVector{
				"count_30d":          float64(rng.Intn(5)),
				"avg_session_min":    float64(rng.Intn(10)),
				"days_since_last":    float64(10 + rng.Intn(25)),
				"features_used_pct":  float64(rng.Intn(25)),
			}
			rawPayments = store.FeatureVector{
				"total":        float64(50 + rng.Intn(500)),
				"late_count":   float64(2 + rng.Intn(5)),
				"monthly_avg":  float64(30 + rng.Intn(40)),
			}
			rawTickets = store.FeatureVector{
				"count_90d":     float64(3 + rng.Intn(8)),
				"avg_resolution": float64(24 + rng.Intn(72)),
				"escalations":   float64(1 + rng.Intn(4)),
			}
		}

		fs.Set(feature.EntityProfile, id, profile)
		fs.Set(feature.EntityRawLogins, id, rawLogins)
		fs.Set(feature.EntityRawPayments, id, rawPayments)
		fs.Set(feature.EntityRawTickets, id, rawTickets)
	}

	return ids
}
```

**Step 4: Run tests**

Run: `go test ./seed/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add seed/
git commit -m "feat: add seed data generator with 3 customer personas"
```

---

### Task 4: Materializer

**Files:**
- Create: `feature/materializer.go`
- Test: `feature/materializer_test.go`

**Step 1: Write the test**

```go
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

	// Check usage metrics were computed
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

	// Check billing
	billing, ok := fs.Get(EntityBilling, "cust-0001")
	if !ok {
		t.Fatal("expected billing to be materialized")
	}
	if billing[TotalSpend] != 1200 {
		t.Fatalf("expected total_spend=1200, got %f", billing[TotalSpend])
	}

	// Check support
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

	// Wait for at least one materialization cycle
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Good — goroutine exited
	case <-time.After(time.Second):
		t.Fatal("materializer did not stop after context cancellation")
	}

	// Verify features were materialized
	_, ok := fs.Get(EntityUsage, "cust-0001")
	if !ok {
		t.Fatal("expected usage_metrics after materializer ran")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./feature/... -v`
Expected: FAIL — Materializer not defined

**Step 3: Write implementation**

```go
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

// RunOnce computes derived features from raw events for all customers.
func (m *Materializer) RunOnce() {
	for _, id := range m.customerIDs {
		m.materializeCustomer(id)
	}
}

func (m *Materializer) materializeCustomer(id string) {
	// Usage metrics from raw logins
	if raw, ok := m.store.Get(EntityRawLogins, id); ok {
		m.store.Set(EntityUsage, id, store.FeatureVector{
			LoginsLast30d:      raw["count_30d"],
			AvgSessionMinutes:  raw["avg_session_min"],
			DaysSinceLastLogin: raw["days_since_last"],
			FeatureAdoptionPct: raw["features_used_pct"],
		})
	}

	// Billing from raw payments
	if raw, ok := m.store.Get(EntityRawPayments, id); ok {
		m.store.Set(EntityBilling, id, store.FeatureVector{
			TotalSpend:        raw["total"],
			LatePaymentsCount: raw["late_count"],
			AvgMonthlySpend:   raw["monthly_avg"],
		})
	}

	// Support from raw tickets
	if raw, ok := m.store.Get(EntityRawTickets, id); ok {
		m.store.Set(EntitySupport, id, store.FeatureVector{
			TicketsLast90d:     raw["count_90d"],
			AvgResolutionHours: raw["avg_resolution"],
			EscalationCount:    raw["escalations"],
		})
	}
}

// Start runs materialization on a loop until ctx is cancelled.
func (m *Materializer) Start(ctx context.Context, interval time.Duration) {
	fmt.Printf("Materializer started (interval: %s, customers: %d)\n", interval, len(m.customerIDs))
	m.RunOnce() // run immediately on start
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
```

**Step 4: Run tests**

Run: `go test ./feature/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add feature/
git commit -m "feat: add materializer to compute derived features from raw events"
```

---

### Task 5: Extend HTTP Handlers

**Files:**
- Modify: `handler/handler.go`
- Modify: `handler/handler_test.go`

**Step 1: Write the tests for new endpoints**

Add to `handler/handler_test.go`:

```go
func setupChurnStore() *store.FeatureStore {
	fs := store.NewFeatureStore()
	// Profile
	fs.Set("customer_profile", "cust-0001", store.FeatureVector{
		"tenure_months": 60, "plan_tier": 3, "monthly_charge": 29.99,
	})
	// Usage
	fs.Set("usage_metrics", "cust-0001", store.FeatureVector{
		"logins_last_30d": 25, "avg_session_minutes": 45,
		"days_since_last_login": 1, "feature_adoption_pct": 85,
	})
	// Billing
	fs.Set("billing", "cust-0001", store.FeatureVector{
		"total_spend": 1800, "late_payments_count": 0, "avg_monthly_spend": 30,
	})
	// Support
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
```

**Step 2: Run test to verify new tests fail**

Run: `go test ./handler/... -v`
Expected: FAIL — new routes don't exist

**Step 3: Add new handlers to handler.go**

Add imports for `"github.com/rohil/gofun/feature"` and `"github.com/rohil/gofun/scoring"`.

Add two new routes in `New()`:
```go
mux.HandleFunc("GET /predict/{customer_id}", handlePredict(fs))
mux.HandleFunc("GET /customers/{customer_id}/features", handleCustomerFeatures(fs))
```

Add handler functions:
```go
func gatherFeatureGroups(fs *store.FeatureStore, customerID string) (map[string]store.FeatureVector, bool) {
	groups := make(map[string]store.FeatureVector)
	groupNames := []string{
		feature.EntityProfile, feature.EntityUsage,
		feature.EntityBilling, feature.EntitySupport,
	}
	found := false
	for _, g := range groupNames {
		if fv, ok := fs.Get(g, customerID); ok {
			groups[g] = fv
			found = true
		}
	}
	return groups, found
}

func handlePredict(fs *store.FeatureStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		customerID := r.PathValue("customer_id")
		groups, found := gatherFeatureGroups(fs, customerID)
		if !found {
			http.Error(w, `{"error":"customer not found"}`, http.StatusNotFound)
			return
		}
		result := scoring.Predict(customerID, groups)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func handleCustomerFeatures(fs *store.FeatureStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		customerID := r.PathValue("customer_id")
		groups, found := gatherFeatureGroups(fs, customerID)
		if !found {
			http.Error(w, `{"error":"customer not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(groups)
	}
}
```

**Step 4: Run tests**

Run: `go test ./handler/... -v`
Expected: PASS (all old + new tests)

**Step 5: Commit**

```bash
git add handler/
git commit -m "feat: add /predict and /customers endpoints"
```

---

### Task 6: Wire Up main.go

**Files:**
- Modify: `main.go`

**Step 1: Update main.go**

Replace `main.go` contents to:
- Use `seed.Generate()` instead of hardcoded data
- Start materializer goroutine with context
- Cancel materializer on shutdown

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rohil/gofun/feature"
	"github.com/rohil/gofun/handler"
	"github.com/rohil/gofun/seed"
	"github.com/rohil/gofun/store"
)

func main() {
	fs := store.NewFeatureStore()

	// Generate seed data
	customerIDs := seed.Generate(fs, 75)
	fmt.Printf("Seeded %d customers\n", len(customerIDs))

	// Start materializer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mat := feature.NewMaterializer(fs, customerIDs)
	go mat.Start(ctx, 10*time.Second)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: handler.New(fs),
	}

	go func() {
		fmt.Println("Feature store listening on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("\nShutting down...")

	// Stop materializer
	cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Server stopped.")
}
```

**Step 2: Verify build**

Run: `go build .`
Expected: compiles without errors

**Step 3: Run all tests**

Run: `go test -race ./...`
Expected: all pass, no race conditions

**Step 4: Commit**

```bash
git add main.go
git commit -m "feat: wire up seed data, materializer, and new endpoints in main"
```

---

### Task 7: Update README

**Files:**
- Modify: `README.md`

**Step 1: Update README with new endpoints and description**

Update to document the churn prediction system, new endpoints, and example curls:
- `curl localhost:8080/predict/cust-0001`
- `curl localhost:8080/customers/cust-0001/features`
- Keep existing generic endpoints documented

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: update README for churn prediction feature store"
```

---

## Verification Checklist

After all tasks:
1. `go build .` — compiles
2. `go test -race ./...` — all pass, no races
3. Manual test:
   - `curl localhost:8080/health`
   - `curl localhost:8080/predict/cust-0001`
   - `curl localhost:8080/customers/cust-0001/features`
   - `curl localhost:8080/features/customer_profile/cust-0001`
