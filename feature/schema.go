// feature/schema.go
package feature

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

var FeatureGroups = map[string][]string{
	"customer_profile": {TenureMonths, PlanTier, MonthlyCharge},
	"usage_metrics":    {LoginsLast30d, AvgSessionMinutes, DaysSinceLastLogin, FeatureAdoptionPct},
	"billing":          {TotalSpend, LatePaymentsCount, AvgMonthlySpend},
	"support":          {TicketsLast90d, AvgResolutionHours, EscalationCount},
}

const (
	EntityProfile     = "customer_profile"
	EntityUsage       = "usage_metrics"
	EntityBilling     = "billing"
	EntitySupport     = "support"
	EntityRawLogins   = "raw_logins"
	EntityRawPayments = "raw_payments"
	EntityRawTickets  = "raw_tickets"
)
