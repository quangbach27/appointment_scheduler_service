# AGENTS

## Purpose
This repository is a Go service scaffold for unified service scheduling. Keep changes minimal, preserve existing architecture boundaries, and prefer adding behavior inside the existing module/layer structure.

## First Commands To Run
- Install/sync dependencies: `go mod tidy`
- Fast unit tests: `make test-unit`
- Integration tests: `make test-integration`
- Component tests: `make test-component`
- Lint: `make lint`
- Format: `make fmt`
- Full CI-like check in this repo: `make lint && make test && go build ./...`

## Build and Runtime
- Local container run: `make up` (uses service `scheduler-app` from `docker-compose.yaml`)
- Stop containers: `make down`
- Recreate with clean volumes: `make up-clean`
- App port defaults to `4000` via env (`PORT` in config and compose mapping)

## Architecture Map
- Entrypoint: `cmd/main.go` (currently minimal)
- Service bootstrap and lifecycle: `internal/svc.go`
- Shared infrastructure:
  - HTTP bootstrap and middleware: `internal/common/http/`
  - Logging and correlation context: `internal/common/log/`
  - Config/env parsing: `internal/common/config/config.go`
  - Module contracts/interfaces: `internal/common/module/`
- Domain area follows layered structure per bounded context:
  - `internal/appointments/domain/`
  - `internal/appointments/app/`
  - `internal/appointments/adapters/`
  - `internal/appointments/ports/`

## Conventions To Preserve
- Keep domain logic in `domain`, orchestration/use-cases in `app`, IO/transport/storage in `adapters` + `ports`.
- Keep HTTP handlers and middleware in `internal/common/http` or module adapters, not in domain packages.
- Prefer constructor-style setup and explicit dependency wiring in service/module bootstrap code.
- Use existing error model (`internal/common/errors.go`) for public error messages/slugs and HTTP mapping.
- Keep logs structured and correlation-aware via context logger helpers in `internal/common/log`.

## Testing Guidance
- Unit tests are primarily under `internal/**` with Go test naming conventions.
- Integration tests use build tag `integration` and load env from `.env.test` in `make test-integration`.
- Component tests live under `tests/component/` and are currently scaffolded; avoid assuming broad component coverage.
- For external services in component tests (excluding DB and message queue), create a stub implementation inside the adapter package.
- Stub file naming convention: `stub_<service>_service.go` next to the real adapter implementation.
- Example: in `internal/appointments/adapters/catalog/`, keep `catalog_service.go` as the real implementation and add `stub_catalog_service.go` for test usage.
- Use `github.com/go-openapi/testify/v2` (`assert`/`require`) for new tests.
- For validation logic, prefer table-driven test cases to cover valid, invalid, and boundary scenarios.

## Important Pitfalls
- `make test` runs integration + component tests, not `make test-unit`. Run `make test-unit` explicitly when changing internal logic.
- Pre-push hooks are defined in `lefthook.yml` and currently call `task ...`; this may fail if `task` is not installed/configured locally.
- HTTP middleware sets and echoes `Correlation-ID` and supports `TestName` header; do not break this behavior when touching middleware/logging.
- `internal/svc.go` currently initializes an empty module list; when adding a module, implement `module.Module` and wire it into service bootstrap.

## High-Value Files To Read Before Large Changes
- `Makefile`
- `.github/workflows/commit-stage.yml`
- `internal/svc.go`
- `internal/common/http/echo.go`
- `internal/common/http/middlewares.go`
- `internal/common/http/errors_handler.go`
- `internal/common/module/module.go`
