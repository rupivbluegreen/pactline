.PHONY: setup install up down logs dev lint typecheck test format generate-types generate-proto migrate clean

setup: install up
	@echo ""
	@echo "Setup complete. Run 'make dev' to start the stack."

install:
	go mod download
	cd apps/ai-sidecar && uv sync
	cd apps/document-sidecar && uv sync
	cd apps/web && pnpm install

up:
	docker compose -f infra/docker-compose.yml up -d

down:
	docker compose -f infra/docker-compose.yml down

logs:
	docker compose -f infra/docker-compose.yml logs -f

dev:
	honcho start

lint:
	gofmt -l . | tee /tmp/pactline-gofmt && test ! -s /tmp/pactline-gofmt
	go vet ./...
	@command -v golangci-lint >/dev/null && golangci-lint run ./... || echo "(skipping golangci-lint — not installed)"
	cd apps/ai-sidecar && uv run ruff check . && uv run ruff format --check .
	cd apps/document-sidecar && uv run ruff check . && uv run ruff format --check .
	cd apps/web && pnpm lint

typecheck:
	@command -v staticcheck >/dev/null && staticcheck ./... || echo "(skipping staticcheck — not installed)"
	cd apps/ai-sidecar && uv run pyright
	cd apps/document-sidecar && uv run pyright
	cd apps/web && pnpm typecheck

test:
	go test ./...
	cd apps/ai-sidecar && uv run pytest
	cd apps/document-sidecar && uv run pytest
	cd apps/web && pnpm test --run

format:
	gofmt -w .
	cd apps/ai-sidecar && uv run ruff check --fix . && uv run ruff format .
	cd apps/document-sidecar && uv run ruff check --fix . && uv run ruff format .

generate-types:
	cd apps/web && pnpm exec openapi-typescript ../../schemas/openapi.yaml -o src/api/types.ts
	@echo "Generated apps/web/src/api/types.ts"

generate-proto:
	@echo "Proto generation lands in Phase 1 once the buf/protoc toolchain is wired in."
	@echo "Phase 0 ships proto schemas only; sidecars are idle stubs."

migrate:
	@echo "Migrations land in Phase 1; goose dir is /migrations."

clean:
	rm -rf bin coverage.out
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	rm -rf apps/web/.next apps/web/node_modules .pytest_cache .ruff_cache
