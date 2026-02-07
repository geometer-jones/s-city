# Quickstart: Nostr Relay with Group Projections

## Prerequisites

- Go 1.25.5
- PostgreSQL 18.1
- Local environment variables for database access

## Setup

1. Create a database and user.
2. Set required environment variables:

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/nostr_relay"
export RELAY_PUBKEY="<hex>"
export RELAY_PRIVKEY="<hex>"
export HTTP_ADDR=":8080"
export LOG_LEVEL="INFO"
export RATE_LIMIT_BURST="30"
export RATE_LIMIT_PER_MIN="120"
export MAX_EVENT_SKEW_SECONDS="300"
```

3. Start the relay service (migrations in
   `relay/src/storage/migrations/*.sql` are applied on startup):

```bash
go run ./relay/cmd/relay
```

4. Verify health and metrics endpoints:

```bash
curl -s http://localhost:8080/health
curl -s http://localhost:8080/metrics
```

## Smoke Tests

1. Publish a valid event and confirm it is accepted:

```bash
curl -s -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{"id":"<event-id>","pubkey":"<pubkey>","created_at":<unix>,"kind":1,"tags":[],"content":"hello","sig":"<signature>"}'
```

2. Query by author and kind and confirm the event is returned:

```bash
curl -s "http://localhost:8080/events?author=<pubkey>&kind=1&limit=20"
```

3. Submit a group metadata event (`kind=39000` or `kind=9007`) and verify
   projection:

```bash
curl -s "http://localhost:8080/groups"
curl -s "http://localhost:8080/groups/<group-id>"
```

4. Submit a deletion request and verify the event is excluded from active
   queries:

```bash
curl -s -X POST http://localhost:8080/events/<event-id>/delete \
  -H "Content-Type: application/json" \
  -d '{"deleted_at":<unix>,"deleted_by":"<pubkey>","reason":"cleanup"}'
```

## Validation Results (2026-02-05)

- `go mod tidy` completed successfully and generated `go.sum`.
- `go test ./...` completed successfully (all packages build; no test files yet).
- Full runtime smoke test against PostgreSQL is pending local DB setup using
  the env vars above.
