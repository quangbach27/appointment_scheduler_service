# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go mod tidy                # sync dependencies
make test-unit              # go test ./internal/...
make test-integration       # loads .env.test, go test -tags integration ./internal/...
make test-component         # go test ./tests/...
make test                   # test-integration + test-component (NOT test-unit — run that separately)
make lint                   # golangci-lint run ./...
make fmt                    # go tool gofumpt -l -w .
make gen                    # go generate ./internal/... && make fmt (regenerates sqlc + OpenAPI code)
go build ./...
```

Run a single test: `go test ./internal/appointments/domain/... -run TestName -v`

Local run: `make up` (docker compose service `scheduler-app`), `make down`, `make up-clean` (wipes volumes). App listens on `PORT` (default `4000`).

Pre-push hooks (`lefthook.yml`) shell out to `task test-unit`, `task lint`, `task fmt` — these will fail if `task` (go-task) isn't installed; use the `make` targets directly instead.

CI (`.github/workflows/commit-stage.yml`) runs golangci-lint, `make test`, `go build ./...`, then a container vulnerability scan and image publish on push to `main`.

## Architecture

Go service (module `scheduler`, Go 1.26) built as **ports-and-adapters (hexagonal)** per bounded context, with Echo for HTTP and pgx/sqlc for Postgres. Dependency direction is strict: `domain` ← `app` ← `adapters`/`ports`. Inner layers never import outer ones.

- `cmd/main.go` — entrypoint (currently a stub).
- `internal/svc.go` — service bootstrap: builds the Echo router, holds a `[]module.Module`, wires each module's `Init` → `RegisterContracts` → contract `Verify()` → `RegisterHttp`, then runs the HTTP server with graceful shutdown via `errgroup`. The module list is currently hardcoded empty — bounded contexts implement `module.Module` (`internal/common/module/module.go`: `Name`/`Init`/`RegisterHttp(ctx, common.EchoRouter)`/`RegisterContracts`) and must be added to that slice to actually get wired up.
- `internal/common/` — shared infra, split between the `common` package root and subpackages:
  - Package root (`errors.go`, `echo.go`, `migrations.go`, `enum.go`, `generic.go`, `uuid.go`, `db.go`): typed `common.Error` (`HttpErrorCode`, `ErrorSlug`, `PublicError`, `Details`, constructed via `NewInvalidInputError`/`NewNotFoundError`/`NewConflictError`/`NewExpiredError`/`NewUnauthorizedError`/`NewForbiddenError`); the `common.EchoRouter` interface that `module.Module.RegisterHttp` and generated OpenAPI servers depend on; `MigrateDatabaseUp` (golang-migrate + pgx, one Postgres schema per module, embedded migration `fs.FS` passed in by each module); `UpdateInTx`/`UpdateInReadCommittedTx` (retrying, isolation-level-aware transaction wrapper with exponential backoff on serialization failures); a generic `Enum[T]` (driver.Valuer/Scanner + validated `UnmarshalText`) used for domain enums; `Must`/`ToPtr`/`SafeDeref` generics.
  - `config/` — env-driven `AppConfig`/`AppointmentConfig`/`DBConfig` (see `.env.example`); `getEnv`/`getIntEnv` helpers with logged fallbacks on parse failure.
  - `http/` — Echo bootstrap, CORS, correlation-ID + request logging middleware, error-to-HTTP mapping. Middleware sets/echoes a `Correlation-ID` response header and honors an inbound `TestName` header for request logging — don't break this when touching `middlewares.go`.
  - `log/` — context-scoped `slog.Logger`, correlation ID propagation.
  - `module/` — the `Module` interface and cross-module `contracts.Contracts` (modules publish/consume shared contracts here, verified once via `Verify()` before any HTTP registration).
  - `shared/` — cross-cutting validated value objects usable by any bounded context (e.g. `Email`).
  - `testutils/` — test-only helpers (e.g. spinning up a DB connection) shared by integration/component tests.

### Bounded context layout (`internal/appointments/` is the template for future contexts)

- `module.go` at the bounded-context root wires the whole thing together: implements `module.Module`, `//go:embed`s its own `adapters/db/migrations/*.sql`, runs them via `common.MigrateDatabaseUp` in `Init`, builds `ports/http.Handlers`, and registers HTTP routes in `RegisterHttp`.
- `domain/` — business rules and validation only; no HTTP/DB/transport imports. Value objects use `NewX(...)` constructors returning `common.Error` variants; state is read through getters, not exported fields. Trusted DB rows are hydrated via `UnmarshalX(...)` in `unmarshal.go`, which skips re-validation (DB is treated as source of truth).
- `app/` — use-case orchestration. `Service` (`service.go`) is built from `*domain.AppointmentFactory`, `*domain.ReservationFactory`, `domain.AppointmentRepository`, and a `CatalogService` port it defines itself (`GetServiceEntry`). `request_appointment.go`/`confirm_appointment.go`/`cancel_appointment.go` are the three use cases (`RequestBooking`/`ConfirmAppointment`/`CancelAppointment`) — **all three are still no-op stubs** (empty command structs, zero-value returns); no business logic, repository calls, or HTTP wiring exists yet.
- `adapters/db/` — sqlc-generated Postgres access. `sqlc.yaml` config + `migrations/` (golang-migrate SQL files) + `queries/` are hand-written sources of truth; generated output goes to `dbmodels/` via `go generate` (`sqlc_generate.go` → `go tool sqlc generate`). Never hand-edit generated files — if a domain type referenced by a `sqlc.yaml` column override is renamed, update the override and rerun `go tool sqlc generate` (run from `internal/appointments/adapters/db/`, or via `make gen`).
- `ports/http/` — OpenAPI spec (`openapi.yaml`) is the source artifact; server/client code is generated via `go generate` (`openapi_generate.go` → `go tool oapi-codegen`). A separate client generator lives under `ports/http/client/`. `Register(router, handlers)` is currently a no-op — no routes are wired yet.
- External service adapters follow `stub_<service>_service.go` next to the real `<service>_service.go` implementing the same interface, so component tests avoid real network/infra calls (DB and message queues are the exception — those aren't stubbed this way).

### Appointment domain model

A booking has two related-but-distinct objects, deliberately kept separate:

- **`Appointment`** (`appointment.go`) owns the actual booking details (`AppointmentDetails`: dealer, service bay, technician, user, service type, vehicle, start/end time) and a 3-value status enum — `pending` / `confirmed` / `cancelled` (`appointment_status.go`). It's created directly in `Pending` via `AppointmentFactory.CreateAppointment` (`factory.go`), which validates required fields and enforces `MinStartLeadTime`.
- **`Reservation`** (`reservation.go`) is a short-lived, detail-free token: just a `ReservationUUID`, the `AppointmentUUID` it guards, and a TTL-based `expiredAt`. Minted via `ReservationFactory.CreateReservation` (`factory.go`), which does no validation — it only stamps `ReservationTTL` (default 5m, `defaultReservationTTL`).
- **`Appointment.Confirm(reservation *Reservation)`** transitions `Pending` → `Confirmed`. It checks the reservation actually belongs to that appointment (`reservation-mismatch`), that the appointment is still `Pending` (`appointment-not-pending`), and that the reservation hasn't expired (`reservation-expired`, via `common.NewExpiredError`) — a failed confirm never mutates status.
- **`Appointment.Cancel()`** only allows `Confirmed` → `Cancelled` (`appointment-not-confirmed` otherwise) and rejects appointments whose start time has passed.
- **"Expired" is not a stored status.** It's computed on read via `Appointment.IsExpired()` (`pending && startTime already passed`) — this is a no-show safety net independent of any specific `Reservation`'s own `IsExpired()` (the short checkout-window expiry checked inside `Confirm`). Nothing ever persists an "expired" value to the DB status column.
- `domain.AppointmentRepository` (in `appointment.go`) is the outbound port: `RequestAppointment(ctx, userID, vehicle, *Appointment, *Reservation) error`, `ConfirmAppointment(ctx, userID, ReservationUUID) (AppointmentUUID, error)`, `CancelAppointment(ctx, userID, AppointmentUUID) error`. No implementation exists yet — this is pure interface, waiting on an adapter.

## Testing conventions

- Unit tests live under `internal/**`, named `*_test.go` alongside the code; use `github.com/go-openapi/testify/v2` (`assert`/`require`), and table-driven cases for validation logic (valid/invalid/boundary).
- Integration tests use the `integration` build tag and load env from `.env.test` (only reachable via `make test-integration`, which sources that file first).
- Component tests live under `tests/component/`.
- For external services consumed by component tests (excluding DB and message queues, which are never stubbed), add a `stub_<service>_service.go` beside the real `<service>_service.go`, implementing the same interface.
