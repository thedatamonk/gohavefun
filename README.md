# Go Feature Store

An in-memory ML feature store HTTP service built with Go's standard library. Demonstrates concurrency primitives (sync.RWMutex), HTTP serving, and graceful shutdown.

## Run

```bash
go run .
```

Server starts on `:8080`.

## Test

```bash
# Run all tests
go test -race ./...

# Run test for handler 
go test ./handler/... -v

# Run test for concurrency on feature store
go test ./store/... -race -v
```

## API

### Health check
```bash
curl localhost:8080/health
```

### Get features
```bash
curl localhost:8080/features/user/123
```

### Set features
```bash
curl -X POST localhost:8080/features/user/456 \
  -d '{"age": 30, "score": 0.95}'
```

### Batch get
```bash
curl -X POST localhost:8080/features/batch \
  -d '{"keys": [{"entity_type":"user","entity_id":"123"},{"entity_type":"item","entity_id":"abc"}]}'
```

## Seeded Data

The server starts with sample data:
- `user:123` — age, score, active_days
- `user:456` — age, score, active_days
- `item:abc` — price, popularity
- `item:def` — price, popularity
