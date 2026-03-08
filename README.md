# Customer Churn Prediction Feature Store

An in-memory ML feature store for customer churn prediction, built with Go's standard library. Simulates a real feature serving workflow: raw events → materialization → feature store → prediction.

## Architecture

- **Store** (`store/`) — Generic feature store with RWMutex-protected concurrent access
- **Feature Schema** (`feature/schema.go`) — 13 churn features across 4 groups (profile, usage, billing, support)
- **Materializer** (`feature/materializer.go`) — Background goroutine computing derived features from raw events every 10s
- **Scorer** (`scoring/`) — XGBoost model (with logistic regression fallback) returning churn probability + risk factors
- **Training** (`training/`) — Python script to train XGBoost from feature store data
- **Seed Data** (`seed/`) — Generates 75 customers across 3 personas (loyal, mixed, at-risk)

## Run

```bash
go run .
```

Server starts on `:8080` with 75 seeded customers and a background materializer.

## Test

```bash
# Run all tests with race detector
go test -race ./...

# Run tests for a specific package
go test ./handler/... -v
go test ./store/... -race -v
go test ./scoring/... -v
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
curl localhost:8080/features/usage_metrics/cust-0001
curl localhost:8080/features/billing/cust-0001
curl localhost:8080/features/support/cust-0001
```

### Set features
```bash
curl -X POST localhost:8080/features/customer_profile/cust-0099 \
  -d '{"tenure_months": 12, "plan_tier": 2, "monthly_charge": 29.99}'
```

### Batch get
```bash
curl -X POST localhost:8080/features/batch \
  -d '{"keys": [{"entity_type":"customer_profile","entity_id":"cust-0001"},{"entity_type":"billing","entity_id":"cust-0001"}]}'
```

## Seeded Customers

75 customers (`cust-0001` through `cust-0075`) generated with deterministic seed:
- ~30% loyal — long tenure, high engagement, no issues
- ~40% mixed — varying signals
- ~30% at-risk — short tenure, low engagement, many support tickets

## Load Testing

Requires [k6](https://k6.io/):

```bash
brew install k6
```

Run the load test (server must be running):

```bash
k6 run loadtest.js
```

The test ramps up to 1000 virtual users and exercises all endpoints:
- Health check
- Single feature fetch
- Batch feature fetch
- Churn prediction

Thresholds: p99 latency < 100ms, error rate < 1%.

## Training the XGBoost Model

Requires Python 3.8+ and a running Go server:

```bash
# Terminal 1: Start the server
go run .

# Terminal 2: Train the model
cd training
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
python train.py
```

This fetches features for all 5,000 customers, generates synthetic churn labels, trains an XGBoost classifier, and exports `models/model.json`.

Restart the Go server to load the new model:

```bash
# The server will print: "Loaded XGBoost model: 100 trees, 13 features"
go run .
curl localhost:8080/predict/cust-0001
```

The response now includes `"model_type": "xgboost"`. Without a trained model, predictions fall back to logistic regression (`"model_type": "logistic_regression"`).

## Scoring Model

XGBoost binary classifier (or logistic regression fallback) with 13 features. Returns:
- `churn_probability` (0-1)
- `risk_level` (low/medium/high)
- `top_risk_factors` (top 3 contributors to churn risk)
- `model_type` (`xgboost` or `logistic_regression`)
