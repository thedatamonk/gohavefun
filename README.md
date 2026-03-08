# Customer Churn Prediction Feature Store

An ML feature store for customer churn prediction, built with Go. Features SQLite persistence, a Feature Registry for metadata management, strict write validation, and a materialization pipeline. Simulates a real feature serving workflow: raw events → materialization → feature store → prediction.

## Architecture

- **Store** (`store/`) — `Store` interface with two backends: `MemoryStore` (in-memory) and `SQLiteStore` (persistent)
- **Registry** (`registry/`) — Feature view definitions with metadata (owner, description, tags). SQLite-backed with CRUD API
- **Feature Schema** (`feature/schema.go`) — 13 churn features across 4 groups (profile, usage, billing, support)
- **Materializer** (`feature/materializer.go`) — Background goroutine computing derived features from raw events every 10s
- **Scorer** (`scoring/`) — Logistic regression model returning churn probability + risk factors
- **Seed Data** (`seed/`) — Generates 75 customers across 3 personas + seeds 7 feature view definitions

## Persistence

Data is stored in SQLite databases under `data/`:
- `data/gofun.db` — Feature values (entity_type + entity_id → JSON feature vector)
- `data/registry.db` — Feature view metadata (name, owner, features, tags, timestamps)

Data persists across server restarts.

## Run

```bash
go run .
```

Server starts on `:8080` with 5,000 seeded customers, 7 registered feature views, and a background materializer.

## Test

```bash
go test -race ./...
```

## API

### Health check
```bash
curl localhost:8080/health
```

### Predict churn
```bash
curl localhost:8080/predict/cust-0001
```

### Get all features for a customer
```bash
curl localhost:8080/customers/cust-0001/features
```

### Get a specific feature group
```bash
curl localhost:8080/features/customer_profile/cust-0001
```

### Set features (validated against registry)
```bash
# Valid — all features defined in customer_profile view
curl -X POST localhost:8080/features/customer_profile/cust-0099 \
  -d '{"tenure_months": 12, "plan_tier": 2, "monthly_charge": 29.99}'

# Rejected — "bad_field" not in customer_profile view
curl -X POST localhost:8080/features/customer_profile/cust-0099 \
  -d '{"bad_field": 99}'
```

### Batch get
```bash
curl -X POST localhost:8080/features/batch \
  -d '{"keys": [{"entity_type":"customer_profile","entity_id":"cust-0001"},{"entity_type":"billing","entity_id":"cust-0001"}]}'
```

### Feature Registry

```bash
# List all feature views
curl localhost:8080/registry/feature-views

# Get a specific view
curl localhost:8080/registry/feature-views/customer_profile

# Create a new view
curl -X POST localhost:8080/registry/feature-views \
  -d '{"name":"my_features","entity_type":"user","features":[{"name":"f1","dtype":"float64","description":"My feature"}],"owner":"my-team"}'

# Update a view
curl -X PUT localhost:8080/registry/feature-views/my_features \
  -d '{"name":"my_features","entity_type":"user","features":[{"name":"f1","dtype":"float64"},{"name":"f2","dtype":"float64"}],"owner":"my-team"}'

# Delete a view
curl -X DELETE localhost:8080/registry/feature-views/my_features
```

## Seeded Customers

5,000 customers (`cust-0001` through `cust-5000`) generated with deterministic seed:
- ~30% loyal — long tenure, high engagement, no issues
- ~40% mixed — varying signals
- ~30% at-risk — short tenure, low engagement, many support tickets

## Load Testing

Requires [k6](https://k6.io/):

```bash
brew install k6
k6 run loadtest.js
```

Ramps up to 1,300 virtual users across 3 scenarios (reads, writes, registry). Thresholds: p95 < 500ms, error rate < 1%.

## Scoring Model

Logistic regression with 13 features. Returns:
- `churn_probability` (0-1)
- `risk_level` (low/medium/high)
- `top_risk_factors` (top 3 contributors to churn risk)
