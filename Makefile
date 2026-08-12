.PHONY: dev up down build test test-unit test-integration test-security test-e2e test-race test-integration test-ratelimit test-pagination test-idempotency test-jobs test-auth test-perf bench-all integration-ratelimit integration-idempotency integration-jobs migrate migrate-org migrate-platform migrate-workflow migrate-audit migrate-integrations migrate-notifications migrate-jobs coverage lint vet tidy tidy-check fmt fmt-check run-auth run-gateway run-worker docker-build docker-build-all docker-run release-check integration-ratelimit integration-idempotency integration-jobs migrate coverage lint vet tidy tidy-check fmt fmt-check run-auth run-gateway run-worker docker-build-auth docker-build-gateway ci security security-vuln validate-openapi validate-migrations sqlc-generate proto-generate run-relay

COMPOSE_FILE := infra/docker/compose.yml
AUTH_DIR := services/auth
ORG_DIR := services/organization
WORKFLOW_DIR := services/workflow
AUDIT_DIR := services/audit
INTEGRATIONS_DIR := services/integrations
NOTIFICATIONS_DIR := services/notifications
JOBS_DIR := services/jobs
GATEWAY_DIR := services/gateway
PLATFORM_DIR := pkg/platform
RELAY_DIR := cmd/outbox-relay
INTEGRATION_DIR := tests/integration
IMAGE_TAG := halcyon/auth:local
GATEWAY_IMAGE_TAG := halcyon/gateway:local
GO_MODULES := pkg/platform services/auth services/organization services/workflow services/audit services/integrations services/notifications services/jobs services/gateway cmd/outbox-relay cmd/organization-consumer cmd/event-topics tests/integration tests/security tests/e2e
INTEGRATION_TAGS := -tags=integration
GOLANGCI_LINT_VERSION := v1.62.2
LINT_PACKAGES := ./pkg/platform/... ./services/... ./cmd/... ./tests/integration/...

dev: up

up:
	docker compose -f $(COMPOSE_FILE) up -d

down:
	docker compose -f $(COMPOSE_FILE) down

build:
	cd $(AUTH_DIR) && go build -o ../../bin/auth ./cmd/auth
	cd $(ORG_DIR) && go build -o ../../bin/organization ./cmd/organization
	cd $(WORKFLOW_DIR) && go build -o ../../bin/workflow ./cmd/workflow
	cd $(WORKFLOW_DIR) && go build -o ../../bin/workflow-worker ./cmd/workflow-worker
	cd $(AUDIT_DIR) && go build -o ../../bin/audit ./cmd/audit
	cd $(AUDIT_DIR) && go build -o ../../bin/audit-consumer ./cmd/consumer
	cd $(INTEGRATIONS_DIR) && go build -o ../../bin/integrations ./cmd/integrations
	cd $(INTEGRATIONS_DIR) && go build -o ../../bin/webhook-dispatcher ./cmd/dispatcher
	cd $(INTEGRATIONS_DIR) && go build -o ../../bin/webhook-worker ./cmd/worker
	cd $(NOTIFICATIONS_DIR) && go build -o ../../bin/notifications ./cmd/notifications
	cd $(NOTIFICATIONS_DIR) && go build -o ../../bin/notification-dispatcher ./cmd/dispatcher
	cd $(NOTIFICATIONS_DIR) && go build -o ../../bin/email-worker ./cmd/worker
	cd $(JOBS_DIR) && go build -o ../../bin/jobs ./cmd/jobs
	cd $(JOBS_DIR) && go build -o ../../bin/job-worker ./cmd/worker
	cd $(GATEWAY_DIR) && go build -o ../../bin/gateway ./cmd/gateway
	cd $(RELAY_DIR) && go build -o ../../bin/outbox-relay .

test:
	$(MAKE) test-unit

test-unit:
	cd $(PLATFORM_DIR) && go test ./... -count=1
	cd $(AUTH_DIR) && go test ./... -count=1
	cd $(ORG_DIR) && go test ./... -count=1
	cd $(WORKFLOW_DIR) && go test ./... -count=1
	cd $(AUDIT_DIR) && go test ./... -count=1
	cd $(INTEGRATIONS_DIR) && go test ./... -count=1
	cd $(NOTIFICATIONS_DIR) && go test ./... -count=1
	cd $(JOBS_DIR) && go test ./... -count=1
	cd $(GATEWAY_DIR) && go test ./... -count=1
	cd tests/security && go test ./... -count=1
	cd $(INTEGRATION_DIR) && go test ./... -count=1

test-security:
	cd tests/security && go test ./... -count=1 -timeout=2m
	cd $(PLATFORM_DIR) && go test ./auth/... ./config/... ./httperrors/... -count=1
	cd $(ORG_DIR) && go test ./internal/application/organization/... ./internal/domain/... -run 'Security|Tenant|IDOR|Authz|Invitation' -count=1
	cd $(INTEGRATIONS_DIR) && go test ./internal/api/http/middleware/... -count=1

test-e2e:
	cd tests/e2e && go test -tags=e2e ./... -count=1 -timeout=5m

test-race:
	cd $(PLATFORM_DIR) && go test -race ./... -count=1
	cd $(AUTH_DIR) && go test -race ./... -count=1
	cd $(ORG_DIR) && go test -race ./... -count=1
	cd $(WORKFLOW_DIR) && go test -race ./... -count=1
	cd $(AUDIT_DIR) && go test -race ./... -count=1
	cd $(INTEGRATIONS_DIR) && go test -race ./... -count=1
	cd $(NOTIFICATIONS_DIR) && go test -race ./... -count=1
	cd $(JOBS_DIR) && go test -race ./... -count=1
	cd $(GATEWAY_DIR) && go test -race ./... -count=1

test-integration:
	cd $(PLATFORM_DIR) && go test $(INTEGRATION_TAGS) ./outbox/... -count=1 -timeout=5m
	cd $(PLATFORM_DIR) && go test $(INTEGRATION_TAGS) ./ratelimit/... -count=1 -timeout=5m
	cd $(AUTH_DIR) && go test $(INTEGRATION_TAGS) ./internal/infrastructure/postgres/... -count=1 -timeout=5m
	cd $(ORG_DIR) && go test $(INTEGRATION_TAGS) ./internal/infrastructure/postgres/... -count=1 -timeout=5m
	cd $(WORKFLOW_DIR) && go test $(INTEGRATION_TAGS) ./internal/infrastructure/postgres/... -count=1 -timeout=5m
	cd $(INTEGRATION_DIR) && go test $(INTEGRATION_TAGS) ./... -count=1 -timeout=10m

test-ratelimit:
	cd $(PLATFORM_DIR) && go test ./ratelimit/... -count=1

test-pagination:
	cd $(PLATFORM_DIR) && go test ./pagination/... -count=1 -bench=.

test-idempotency:
	cd $(PLATFORM_DIR) && go test ./idempotency/... -count=1 -bench=.

integration-idempotency:
	cd $(PLATFORM_DIR) && go test -tags=integration ./idempotency/... -count=1 -timeout=5m

test-jobs:
	cd $(PLATFORM_DIR) && go test ./jobs/... -count=1 -bench=.

test-config:
	cd $(PLATFORM_DIR) && go test ./config/... -count=1

test-auth:
	cd $(PLATFORM_DIR) && go test ./auth/... -count=1
	cd $(INTEGRATIONS_DIR) && go test ./internal/api/http/middleware/... -count=1
	cd $(ORG_DIR) && go test ./internal/application/organization/... -count=1

test-perf:
	cd $(PLATFORM_DIR) && go test ./perf/... -count=1
	cd $(PLATFORM_DIR) && go test ./pagination/... ./idempotency/... ./jobs/... ./auth/... ./ratelimit/... -count=1

bench-all:
	cd $(PLATFORM_DIR) && go test -bench=Benchmark -benchmem -count=1 ./pagination/... ./idempotency/... ./jobs/... ./auth/... ./ratelimit/... ./perf/...

integration-jobs:
	cd $(PLATFORM_DIR) && go test -tags=integration ./jobs/postgres/... -count=1 -timeout=5m

run-worker:
	cd $(JOBS_DIR) && go run ./cmd/worker

integration-ratelimit:
	cd $(PLATFORM_DIR) && go test $(INTEGRATION_TAGS) ./ratelimit/... -count=1 -timeout=5m

migrate:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	cd $(AUTH_DIR) && go run ./cmd/migrate

migrate-org:
	@test -n "$(ORGANIZATION_DATABASE_URL)" || (echo "ORGANIZATION_DATABASE_URL is required" && exit 1)
	cd $(ORG_DIR) && go run ./cmd/migrate

migrate-platform:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	cd $(PLATFORM_DIR) && go run ./cmd/migrate

migrate-workflow:
	@test -n "$(WORKFLOW_DATABASE_URL)" || (echo "WORKFLOW_DATABASE_URL is required" && exit 1)
	cd $(WORKFLOW_DIR) && go run ./cmd/migrate

migrate-audit:
	@test -n "$(AUDIT_DATABASE_URL)" || (echo "AUDIT_DATABASE_URL is required" && exit 1)
	cd $(AUDIT_DIR) && go run ./cmd/migrate

migrate-integrations:
	@test -n "$(INTEGRATIONS_DATABASE_URL)" || (echo "INTEGRATIONS_DATABASE_URL is required" && exit 1)
	cd $(INTEGRATIONS_DIR) && go run ./cmd/migrate

migrate-notifications:
	@test -n "$(NOTIFICATIONS_DATABASE_URL)" || (echo "NOTIFICATIONS_DATABASE_URL is required" && exit 1)
	cd $(NOTIFICATIONS_DIR) && go run ./cmd/migrate

migrate-jobs:
	@test -n "$(JOBS_DATABASE_URL)" || (echo "JOBS_DATABASE_URL is required" && exit 1)
	cd $(JOBS_DIR) && go run ./cmd/migrate

run-workflow:
	cd $(WORKFLOW_DIR) && go run ./cmd/workflow

run-workflow-worker:
	cd $(WORKFLOW_DIR) && go run ./cmd/workflow-worker

build-workflow:
	cd $(WORKFLOW_DIR) && go build -o ../../bin/workflow ./cmd/workflow

build-workflow-worker:
	cd $(WORKFLOW_DIR) && go build -o ../../bin/workflow-worker ./cmd/workflow-worker

test-workflow:
	cd $(WORKFLOW_DIR) && go test ./... -count=1

docker-build-workflow:
	docker build -f services/workflow/Dockerfile -t halcyon/workflow:local .

docker-build-workflow-worker:
	docker build -f services/workflow/Dockerfile -t halcyon/workflow-worker:local .

sqlc-generate:
	cd $(AUTH_DIR) && sqlc generate

coverage:
	cd $(PLATFORM_DIR) && go test ./... -count=1 -coverprofile=coverage-platform.out
	cd $(AUTH_DIR) && go test ./... -count=1 -coverprofile=coverage-auth.out
	cd $(ORG_DIR) && go test ./... -count=1 -coverprofile=coverage-organization.out
	cd $(WORKFLOW_DIR) && go test ./... -count=1 -coverprofile=coverage-workflow.out
	@echo "Coverage reports: pkg/platform/coverage-platform.out, services/auth/coverage-auth.out, services/organization/coverage-organization.out, services/workflow/coverage-workflow.out"

fmt:
	@dirs=$$(bash scripts/ci/list-go-modules.sh); gofmt -w $$dirs

fmt-check:
	bash scripts/ci/fmt-check.sh

tidy:
	@for dir in $(GO_MODULES); do \
		(cd $$dir && go mod tidy); \
	done
	go work sync

tidy-check:
	bash scripts/ci/tidy-check.sh

vet:
	@for dir in $(GO_MODULES); do \
		echo "vet $$dir"; \
		(cd $$dir && go vet ./...); \
	done

lint: vet
	golangci-lint run --config .golangci.yml $(LINT_PACKAGES)

validate-openapi:
	cd scripts/ci/validate-openapi && go run .

validate-migrations:
	bash scripts/ci/validate-migrations.sh

security-vuln:
	@for dir in $(GO_MODULES); do \
		echo "govulncheck $$dir"; \
		(cd $$dir && go run golang.org/x/vuln/cmd/govulncheck@latest ./...); \
	done

security: security-vuln

run-auth:
	cd $(AUTH_DIR) && go run ./cmd/auth

run-organization:
	cd $(ORG_DIR) && go run ./cmd/organization

run-gateway:
	cd $(GATEWAY_DIR) && go run ./cmd/gateway

test-gateway:
	cd $(GATEWAY_DIR) && go test ./... -count=1

build-gateway:
	cd $(GATEWAY_DIR) && go build -o ../../bin/gateway ./cmd/gateway

run-relay:
	cd $(RELAY_DIR) && go run .

run-organization-consumer:
	cd cmd/organization-consumer && go run .

init-event-topics:
	cd cmd/event-topics && go run .

docker-build-auth:
	docker build -f services/auth/Dockerfile -t $(IMAGE_TAG) .

docker-build-gateway:
	docker build -f services/gateway/Dockerfile -t $(GATEWAY_IMAGE_TAG) .

docker-build-all:
	docker build -f services/auth/Dockerfile -t halcyon/auth:local .
	docker build -f services/organization/Dockerfile -t halcyon/organization:local .
	docker build -f services/gateway/Dockerfile -t halcyon/gateway:local .
	docker build -f services/workflow/Dockerfile -t halcyon/workflow:local .
	docker build -f services/audit/Dockerfile -t halcyon/audit:local .
	docker build -f services/integrations/Dockerfile -t halcyon/integrations:local .
	docker build -f services/notifications/Dockerfile -t halcyon/notifications:local .
	docker build -f services/jobs/Dockerfile -t halcyon/jobs:local .
	docker build -f cmd/outbox-relay/Dockerfile -t halcyon/outbox-relay:local .

docker-run: up

release-check: fmt-check tidy-check validate-migrations validate-openapi vet lint test test-security build
	@echo "Release validation passed (integration/docker optional — run make test-integration docker-build-all separately)."

ci: fmt-check tidy-check validate-migrations validate-openapi vet lint test test-race test-security coverage build
	@echo "CI checks passed."
