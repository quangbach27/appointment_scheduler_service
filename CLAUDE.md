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

Local run: `make up` (docker compose services `scheduler-app` + `scheduler-db`), `make down`, `make up-clean` (wipes volumes). App listens on `PORT` (default `4000`); Postgres is exposed on `DB_PORT` (default `5432`).

Pre-push hooks (`lefthook.yml`) shell out to `task test-unit`, `task lint`, `task fmt` — these will fail if `task` (go-task) isn't installed; use the `make` targets directly instead.

CI (`.github/workflows/commit-stage.yml`) runs golangci-lint, `make test`, `go build ./...`, then a container vulnerability scan and image publish on push to `main`.

## Architecture

Go service (module `scheduler`, Go 1.26) built as **ports-and-adapters (hexagonal)** per bounded context, with Echo for HTTP and pgx/sqlc for Postgres. Dependency direction is strict: `domain` ← `app` ← `adapters`/`ports`. Inner layers never import outer ones.

- `cmd/main.go` — entrypoint (currently a stub).
- `internal/svc.go` — service bootstrap: builds the Echo router, holds a `[]module.Module`, wires each module's `Init` → `RegisterContracts` → contract `Verify()` → `RegisterHttp`, then runs the HTTP server with graceful shutdown via `errgroup`. Bounded contexts implement `module.Module` (`internal/common/module/module.go`: `Name`/`Init`/`RegisterHttp(ctx, common.EchoRouter)`/`RegisterContracts`) and are added to this slice to get wired up; `appointments.NewModule(config, pgxDb)` is in the slice, so that context is mounted and serving.
- `internal/common/` — shared infra, split between the `common` package root and subpackages:
  - Package root (`errors.go`, `echo.go`, `migrations.go`, `enum.go`, `generic.go`, `uuid.go`, `db.go`): typed `common.Error` (`HttpErrorCode`, `ErrorSlug`, `PublicError`, `Details`, constructed via `NewInvalidInputError`/`NewNotFoundError`/`NewConflictError`/`NewExpiredError`/`NewUnauthorizedError`/`NewForbiddenError`); the `common.EchoRouter` interface that `module.Module.RegisterHttp` and generated OpenAPI servers depend on; `MigrateDatabaseUp` (golang-migrate + pgx, one Postgres schema per module, embedded migration `fs.FS` passed in by each module); `UpdateInTx`/`UpdateInReadCommittedTx` (retrying, isolation-level-aware transaction wrapper with exponential backoff on serialization failures); `IsUniqueViolationError(err, constraint)` (pgconn error inspection, used for insert-first idempotency patterns); a generic `Enum[T]` (driver.Valuer/Scanner + validated `UnmarshalText`) used for domain enums; `Must`/`ToPtr`/`SafeDeref` generics.
  - `config/` — env-driven `AppConfig`/`AppointmentConfig`/`DBConfig` (see `.env.example`); `getEnv`/`getIntEnv` helpers with logged fallbacks on parse failure.
  - `http/` — Echo bootstrap, CORS, correlation-ID + request logging middleware, error-to-HTTP mapping (`errors_handler.go`'s `EchoErrorHandler` maps `common.Error`/`echo.HTTPError` to a `{message, slug, details}` JSON body using `common.Error.HttpErrorCode`/`ErrorSlug`/`Details`). Middleware sets/echoes a `Correlation-ID` response header and honors an inbound `TestName` header for request logging — don't break this when touching `middlewares.go`.
  - `log/` — context-scoped `slog.Logger`, correlation ID propagation.
  - `module/` — the `Module` interface and cross-module `contracts.Contracts` (modules publish/consume shared contracts here, verified once via `Verify()` before any HTTP registration).
  - `shared/` — cross-cutting validated value objects usable by any bounded context (e.g. `Email`).
  - `testutils/` — test-only helpers (e.g. spinning up a DB connection) shared by integration/component tests.

### Bounded context layout (`internal/appointments/` is the template for future contexts)

- `module.go` at the bounded-context root wires the whole thing together: implements `module.Module`, `//go:embed`s its own `adapters/db/migrations/*.sql`, runs them via `common.MigrateDatabaseUp` in `Init`, constructs the domain factory + repository adapter + read model + `app.Service`, builds `ports/http.Handlers`, and registers HTTP routes in `RegisterHttp`.
- `domain/` — business rules and validation only; no HTTP/DB/transport imports. Value objects use `NewX(...)` constructors returning `common.Error` variants; state is read through getters, not exported fields. Trusted DB rows are hydrated via `UnmarshalX(...)` in `unmarshal.go`, which skips re-validation (DB is treated as source of truth).
- `app/` — use-case orchestration. `Service` (`service.go`) is built from `*domain.BookingFactory` and `domain.AppointmentRepository`. `request_appointment.go`/`confirm_appointment.go`/`cancel_appointment.go` are the three use cases (`RequestBooking`/`ConfirmAppointment`/`CancelAppointment`): `ConfirmAppointment` and `CancelAppointment` are implemented (both pure delegation to the corresponding `AppointmentRepository` method, which already contains the full domain confirm/cancel + persist logic); `RequestBooking` is still a no-op stub. `CatalogService`/`ServiceCatalogEntry` are defined in `request_appointment.go` but currently unused by `Service`.
- `adapters/db/` — sqlc-generated Postgres access. `sqlc.yaml` config + `migrations/` (golang-migrate SQL files) + `queries/` are hand-written sources of truth; generated output goes to `dbmodels/` via `go generate` (`sqlc_generate.go` → `go tool sqlc generate`). Never hand-edit generated files — if a domain type referenced by a `sqlc.yaml` column override is renamed, update the override and rerun `go tool sqlc generate` (run from `internal/appointments/adapters/db/`, or via `make gen`). `AppointmentRepository` (`appointment_repository.go`) is a real implementation of `domain.AppointmentRepository`, using `common.UpdateInTx` for the multi-statement `RequestAppointment`/`ConfirmAppointment`/`CancelAppointment` flows and an insert-first/catch-conflict pattern (`common.IsUniqueViolationError`) for reservation idempotency. Migrations deliberately have no performance indexes yet, except the `uq_reservation_idempotency_key` UNIQUE constraint (a correctness constraint, not a query-optimization index).
- `ports/http/` — OpenAPI spec (`openapi.yaml`) is the source artifact; server/client code is generated via `go generate` (`openapi_generate.go` → `go tool oapi-codegen`, strict-server + models, output `openapi.gen.go`). A separate client generator lives under `ports/http/client/`. Convention: request/response schema fields that represent a domain identifier or enum (UUIDs, statuses) are declared as named `components/schemas` entries bound directly to the real domain Go type via `x-go-type`/`x-go-type-import`, e.g. an `AppointmentUUID` schema with `x-go-type: domain.AppointmentUUID` and `x-go-type-import: {path: scheduler/internal/appointments/domain}` — not inlined as `type: string, format: uuid`. This makes the generated Go types *be* the domain types (no manual conversion needed in handlers, since the domain types already implement `encoding.TextMarshaler`/`TextUnmarshaler` via `common.UUID`/`common.Enum[T]`), and is the pattern to follow for new endpoints. `Register(router, handlers)` wires the generated `RegisterHandlers`/`NewStrictHandler`. `handlers.go` also declares a `ReservationReadModel` outbound port (`GetReservationByID(ctx, userID, ReservationUUID) (GetReservationResponse, error)`) for query-side endpoints — a CQRS-style read side that bypasses the domain layer entirely, implemented by `adapters/db.ReservationReadModel` via a single joined sqlc query and wired in through `module.go`. It returns the OpenAPI-generated response DTO directly, so no manual mapping is needed in the handler.
- External service adapters follow `stub_<service>_service.go` next to the real `<service>_service.go` implementing the same interface, so component tests avoid real network/infra calls (DB and message queues are the exception — those aren't stubbed this way).

### Appointment domain model

A booking has two related-but-distinct objects, deliberately kept separate:

- **`Appointment`** (`appointment.go`) owns the actual booking details (`AppointmentDetails`: dealer, service bay, technician, user, service type, vehicle, start/end time) and a 3-value status enum — `pending` / `confirmed` / `cancelled` (`appointment_status.go`). It's created directly in `Pending` via `BookingFactory.CreateAppointment` (`factory.go`), which validates required fields and enforces `MinStartLeadTime`.
- **`Reservation`** (`reservation.go`) is a short-lived token: a `ReservationUUID`, the `AppointmentUUID` it guards, a TTL-based `expiredAt`, and a caller-supplied `idempotencyKey` (prevents duplicate/spam booking requests — enforced by a DB `UNIQUE` constraint, with an insert-first/catch-conflict pattern that returns the original reservation on replay). Minted via `BookingFactory.CreateReservation(appointmentUUID, idempotencyKey)` (`factory.go`), which requires a non-empty `idempotencyKey` and stamps `ReservationTTL` (default 5m, `defaultReservationTTL`). `AppointmentFactory`/`ReservationFactory` were merged into this single `BookingFactory` — there is no separate factory per object anymore.
- **`Appointment.Confirm(reservation *Reservation)`** transitions `Pending` → `Confirmed`. It checks the reservation actually belongs to that appointment (`reservation-mismatch`), that the appointment is still `Pending` (`appointment-not-pending`), and that the reservation hasn't expired (`reservation-expired`, via `common.NewExpiredError`) — a failed confirm never mutates status.
- **`Appointment.Cancel()`** only allows `Confirmed` → `Cancelled`, checked in order: already-cancelled (`appointment-already-cancelled`, 409), then not-confirmed (`appointment-not-confirmed`, 409), then start time already passed (`appointment-in-the-past`, 400) — all three are typed `common.Error`s.
- **"Expired" is not a stored status.** It's computed on read via `Appointment.IsExpired()` (`pending && startTime already passed`) — this is a no-show safety net independent of any specific `Reservation`'s own `IsExpired()` (the short checkout-window expiry checked inside `Confirm`). Nothing ever persists an "expired" value to the DB status column.
- `domain.AppointmentRepository` (in `appointment.go`) is the outbound port: `RequestAppointment(ctx, userID, *Vehicle, *Appointment, *Reservation) (ReservationUUID, error)`, `ConfirmAppointment(ctx, userID, ReservationUUID) (AppointmentUUID, error)`, `CancelAppointment(ctx, userID, AppointmentUUID) error`. It now has a real Postgres implementation (`adapters/db/appointment_repository.go`).

## Testing conventions

- Unit tests live under `internal/**`, named `*_test.go` alongside the code; use `github.com/go-openapi/testify/v2` (`assert`/`require`), and table-driven cases for validation logic (valid/invalid/boundary). By convention, `app/` and `ports/http/` are not unit-tested — **all business-logic coverage for those two layers is expected to come from component tests instead**.
- Integration tests use the `integration` build tag and load env from `.env.test` (only reachable via `make test-integration`, which sources that file first).
- Component tests live under `tests/component/`. `setup_test.go`'s `TestMain` boots the real wired service via `internal.New(ctx, config, dbPgx, internal.ExternalService{})` against a real Postgres (`testutils.NewDB()`) and runs the real HTTP server (`:9090`); tests drive it end-to-end through the generated OpenAPI client (`ports/http/client`, see `newTestClients` in `helpers_test.go`) rather than calling handlers or `app.Service` directly.
- For external services consumed by component tests (excluding DB and message queues, which are always real), add a `stub_<service>_service.go` beside the real `<service>_service.go` implementing the same interface, and inject it via `internal.ExternalService` — this is what keeps component tests hitting a real service and a real DB while avoiding real network calls. `internal.ExternalService` is currently an empty struct since no external service adapters exist yet.
