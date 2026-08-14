SERVICE ?= scheduler-app
LOAD_TEST_ENV = set -a && . ./.env.test && set +a &&

.PHONY: up
up:
	COMPOSE_MENU=false docker compose up $(SERVICE)

.PHONY: down
down:
	docker compose down

.PHONY: up-clean
up-clean:
	$(MAKE) down-volumes
	$(MAKE) up

.PHONY: down-volumes
down-volumes:
	docker compose down -v

## Run all tests (unit, integration, component)
.PHONY: test
test:
	$(LOAD_TEST_ENV) \
	go test -tags integration ./internal/... -count=1 && \
	go test ./tests/... -count=1

## Run unit tests
.PHONY: test-unit
test-unit:
	go test ./internal/...

## Run integration tests
.PHONY: test-integration
test-integration:
	$(LOAD_TEST_ENV) go test -tags integration ./internal/... -count=1

## Run component tests
.PHONY: test-component
test-component:
	$(LOAD_TEST_ENV) go test ./tests/... -count=1

## Run go generate
.PHONY: gen
gen:
	go generate ./internal/...
	$(MAKE) fmt

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: fmt
fmt:
	go tool gofumpt -l -w .

## Update Go module dependencies
.PHONY: go-mod-update
go-mod-update:
	go get -u ./...
