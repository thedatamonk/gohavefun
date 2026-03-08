# Customer Churn Prediction Feature Store — Design

## Goal

Extend the generic Go feature store into a customer churn prediction system that simulates a real ML feature serving workflow: raw events → materialization → feature store → prediction.

## Domain Model

### Feature Groups (per customer)

**Customer Profile** (slow-changing, set at seed time)
- `tenure_months` — months since signup
- `plan_tier` — 1=basic, 2=pro, 3=enterprise
- `monthly_charge` — subscription price

**Usage Metrics** (derived by materializer)
- `logins_last_30d` — login count
- `avg_session_minutes` — average session length
- `days_since_last_login` — inactivity measure
- `feature_adoption_pct` — % of product features used

**Billing History** (derived by materializer)
- `total_spend` — lifetime spend
- `late_payments_count` — missed/late payments
- `avg_monthly_spend` — average monthly billing

**Support Interactions** (derived by materializer)
- `tickets_last_90d` — recent support tickets
- `avg_resolution_hours` — support response quality
- `escalation_count` — escalated complaints

### Storage Keys

Feature groups stored as entity types in the existing store:
- `customer_profile:123`, `usage_metrics:123`, `billing:123`, `support:123`

Raw events stored similarly:
- `raw_logins:123`, `raw_payments:123`, `raw_tickets:123`

## Architecture

```
gofun/
├── main.go                    # wiring, seed data, graceful shutdown
├── store/
│   └── store.go               # generic feature store (unchanged)
├── feature/
│   ├── schema.go              # feature group constants, weight definitions
│   └── materializer.go        # background goroutine: raw events → derived features
├── scoring/
│   └── scorer.go              # logistic regression: sigmoid(w·x + bias)
├── handler/
│   └── handler.go             # extended with /predict and /customers endpoints
├── seed/
│   └── seed.go                # generateSeedData() for 50-100 customers
└── go.mod
```

### Data Flow

1. `seed.go` generates ~75 customers with raw events and profile data at startup
2. Materializer goroutine runs every 10 seconds, computes derived features from raw events
3. HTTP API serves feature lookups and predictions
4. Graceful shutdown stops materializer via context cancellation

## Scoring Model

Logistic regression: `P(churn) = sigmoid(w · x + bias)`

| Feature | Weight | Reasoning |
|---|---|---|
| tenure_months | -0.05 | Longer tenure → less churn |
| plan_tier | -0.3 | Higher tier → more invested |
| monthly_charge | +0.02 | Price sensitivity |
| logins_last_30d | -0.15 | Engagement signal |
| avg_session_minutes | -0.03 | Engagement depth |
| days_since_last_login | +0.1 | Inactivity signal |
| feature_adoption_pct | -0.02 | Product stickiness |
| total_spend | -0.001 | Sunk cost |
| late_payments_count | +0.4 | Dissatisfaction |
| avg_monthly_spend | -0.01 | Value perception |
| tickets_last_90d | +0.2 | Frustration |
| avg_resolution_hours | +0.05 | Support quality |
| escalation_count | +0.5 | Severe dissatisfaction |
| bias | -0.5 | Default toward not churning |

Risk levels: low (<0.3), medium (0.3–0.7), high (>0.7)

## API Endpoints

Existing (unchanged):
- `GET /health`
- `GET /features/{entity_type}/{entity_id}`
- `POST /features/{entity_type}/{entity_id}`
- `POST /features/batch`

New:
- `GET /predict/{customer_id}` — returns `{"customer_id", "churn_probability", "risk_level", "top_risk_factors"}`
- `GET /customers/{customer_id}/features` — returns all 4 feature groups for a customer

## Seed Data

~75 customers generated with `math/rand`, distributed across personas:
- Loyal (low churn signals): long tenure, high engagement, no support issues
- At-risk (high churn signals): short tenure, low engagement, many tickets
- Mixed: varying combinations to create realistic spread

## Go Concepts Exercised

- Background goroutines with context cancellation (materializer)
- sync.RWMutex (existing store)
- math/rand for data generation
- Struct composition and package organization
- Graceful shutdown coordination across goroutines
