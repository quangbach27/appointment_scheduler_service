# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make tidy                   # sync dependencies
make test-unit              # go test ./internal/...
make test-integration       # loads .env.test, go test -tags integration ./internal/... (+ unit test)
make test-component         # go test ./tests/...
make test                   # test-integration + test-component + unit test
make lint                   # golangci-lint run ./...
make fmt                    # go tool gofumpt -l -w .
make gen                    # go generate ./internal/... && make fmt (regenerates sqlc + OpenAPI code)
go build ./...
```

Run a single test: `go test ./internal/appointments/domain/... -run TestName -v`

Local run: `make up` (docker compose services `scheduler-app` + `scheduler-db`), `make down`, `make up-clean` (wipes volumes). App listens on `PORT` (default `4000`); Postgres is exposed on `DB_PORT` (default `5432`).

Pre-push hooks (`lefthook.yml`) shell out to `task test-unit`, `task lint`, `task fmt`

CI (`.github/workflows/commit-stage.yml`) runs golangci-lint, `make test`, `go build ./...`, then a container vulnerability scan and image publish on push to `main`.

## Architecture

Go 1.26 service (module `scheduler`), **ports-and-adapters per bounded context**, Echo for HTTP and pgx/sqlc for Postgres. Dependency direction is strict: `domain` ← `app` ← `adapters`/`ports`. Inner layers never import outer ones.

- `cmd/main.go` — builds the config and pgx pool, fills `internal.ExternalService` with the external-service adapters, runs the service under a signal-cancelled context.
- `internal/svc.go` — bootstrap: an Echo router and a `[]module.Module`, each wired `Init` → `RegisterContracts` → `Verify()` → `RegisterHttp`, then HTTP serve with graceful shutdown. A bounded context joins the app by implementing `module.Module` and being added to that slice.
- `internal/common/` — shared infra:
  - `common.Error` — typed error (`HttpErrorCode`, `ErrorSlug`, `PublicError`, `Details`), built via `NewInvalidInputError` / `NewNotFoundError` / `NewConflictError` / `NewExpiredError` / `NewUnauthorizedError` / `NewForbiddenError`.
  - `UpdateInTx` (RepeatableRead) / `UpdateInReadCommittedTx` / `UpdateInSerializableTx` — transaction wrappers that retry serialization failures. RepeatableRead does **not** prevent write-skew: any "check a set is free, then insert into it" flow needs `UpdateInSerializableTx` *and* the conflicting read inside the transaction.
  - Also `MigrateDatabaseUp` (one Postgres schema per module), `IsUniqueViolationError`, generic `Enum[T]` for domain enums, `EchoRouter`, `Must`/`ToPtr`/`SafeDeref`.
  - Subpackages: `config/` (env-driven, see `.env.example`), `http/` (Echo bootstrap, CORS, correlation-ID + request-logging middleware, `EchoErrorHandler` mapping `common.Error` to `{message, slug, details}`), `log/`, `module/` (the `Module` interface + cross-module contracts), `shared/`, `testutils/`.

### Bounded context layout (`internal/appointments/` is the template)

- `module.go` — implements `module.Module`: embeds and runs its own migrations, then constructs the domain factory, adapters, `app.Service`, and `ports/http.Handlers`.
- `domain/` — rules and validation only, no HTTP/DB imports. `NewX(...)` constructors return `common.Error`; state is read through getters. Trusted DB rows hydrate via `UnmarshalX(...)`, skipping re-validation.
- `app/` — one file per use case. `Service` holds the domain factory, the repository port, and any outbound service ports (declared in `app`, not `adapters`). Commands return identifiers rather than whole aggregates. External calls go **before** the transaction, which is retried on serialization failure and would otherwise re-issue them.
- `adapters/db/` — sqlc. `sqlc.yaml`, `migrations/`, and `queries/` are the hand-written sources; `dbmodels/` is generated, never hand-edited. Renaming a domain type used in a column override means updating the override and rerunning `make gen`.
  - **Concurrency pattern to copy** (`appointment_repository.go`): in one SERIALIZABLE transaction, resolve the idempotency key, run the availability `SELECT`, then insert. That `SELECT` is load-bearing, not a guard — SSI needs a read to detect write-skew, or concurrent bookings both commit; a component test proves it. `IsUniqueViolationError` after the transaction backstops identical requests racing past the key lookup, and migration `0004`'s indexes keep predicate locks narrow (a seq scan locks the whole relation).
- `ports/http/` — `openapi.yaml` is the source artifact; server and client code are generated. Two conventions: bind identifier/enum schemas to the real domain types via `x-go-type`/`x-go-type-import` so handlers need no conversion, and give query-side endpoints a read-model port implemented in `adapters/db` that returns the generated DTO directly (a CQRS read side, bypassing the domain).
- External services get their own `adapters/<service>/` package, with `stub_<service>_service.go` beside the real implementation and injection via `internal.ExternalService`. DB and message queues are never stubbed this way.

Appointment domain rules — the two-phase booking lifecycle, state transitions, the two expiries, availability — live in `internal/appointments/CLAUDE.md`.

## Testing conventions

- Unit tests sit beside the code, use `github.com/go-openapi/testify/v2`, and are table-driven for validation logic. `app/` and `ports/http/` are **not** unit-tested — component tests cover them.
- Integration tests use the `integration` build tag and `.env.test`. Component tests (`tests/component/`) boot the real service against real Postgres on `:9090` and drive it through the generated OpenAPI client, never handlers or `app.Service` directly.
- **The test database is long-lived and never truncated**, so every run must carve out its own data: component tests take a fresh future day per booking (`nextStart()`, valid only because `.env.test` widens `APPOINTMENT_MAX_START_LEAD_TIME_DAYS` — keep the two in step), and DB integration tests take unique technician/bay IDs and run serially, since concurrent SERIALIZABLE bookings on a small table just exhaust the retry budget.
- Stub fixtures should make every interesting branch reachable deterministically — see `adapters/workforce`'s permanently-off dealerships.
