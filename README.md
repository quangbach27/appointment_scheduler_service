# Unified Service Scheduler

An appointment-booking backend for dealership service scheduling: it checks real-time availability of technicians and service bays and prevents double-bookings. See [docs/SYSTEM_DESIGN.md](docs/SYSTEM_DESIGN.md) for the full domain model and architecture.

## Prerequisites

- Docker + Docker Compose
- Go 1.26.1 (only needed if running outside Docker)

## Quick start

```bash
cp .env.example .env
make up
```

`make up` starts `scheduler-db` (Postgres 17) and `scheduler-app` (hot-reloading on source changes) via `docker compose`. Once healthy:

- App: http://localhost:4000
- Postgres: localhost:5432 (user `user`, password `password`, db `scheduler_service`)

Stop everything with `make down`. To wipe the database volume and start clean, use `make up-clean`.

## Verify it's running

```bash
curl http://localhost:4000/healthz
```

## Configuration

All environment variables are documented in [.env.example](.env.example) (server port, CORS, reservation/lead-time settings, DB connection). Copy it to `.env` and adjust as needed; `.env` is gitignored.

## Running tests

```bash
make test-unit          # go test ./internal/...
make test-integration   # loads .env.test, requires -tags integration
make test-component     # go test ./tests/...
make test               # integration + component + unit
```

## More

- [docs/SYSTEM_DESIGN.md](docs/SYSTEM_DESIGN.md) — domain model, architecture, data flow, observability.
- [CLAUDE.md](CLAUDE.md) — full command reference and codebase conventions.
