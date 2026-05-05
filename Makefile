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
	@echo ">> proto drift check"
	$(MAKE) generate-proto
	git diff --exit-code -- internal/ai/aigrpc internal/document/documentgrpc apps/ai-sidecar/src/ai_sidecar/proto apps/document-sidecar/src/document_sidecar/proto || (echo "Proto stubs are stale — run 'make generate-proto' and commit." && exit 1)

typecheck:
	@command -v staticcheck >/dev/null && staticcheck ./... || echo "(skipping staticcheck — not installed)"
	cd apps/ai-sidecar && uv run pyright
	cd apps/document-sidecar && uv run pyright
	cd apps/web && pnpm typecheck

test:
	go test ./...
	cd apps/ai-sidecar && uv run pytest
	cd apps/document-sidecar && uv run pytest
	cd apps/web && pnpm test

format:
	gofmt -w .
	cd apps/ai-sidecar && uv run ruff check --fix . && uv run ruff format .
	cd apps/document-sidecar && uv run ruff check --fix . && uv run ruff format .

generate-types:
	cd apps/web && pnpm exec openapi-typescript ../../schemas/openapi.yaml -o src/api/types.ts
	@echo "Generated apps/web/src/api/types.ts"

generate-proto:
	@echo ">> Go stubs (buf)"
	buf generate
	@echo ">> Python stubs (grpcio-tools, ai-sidecar)"
	cd apps/ai-sidecar && uv run python -m grpc_tools.protoc \
		-I../../proto \
		--python_out=src/ai_sidecar/proto \
		--grpc_python_out=src/ai_sidecar/proto \
		--pyi_out=src/ai_sidecar/proto \
		../../proto/ai.proto ../../proto/document.proto
	@echo ">> Python stubs (grpcio-tools, document-sidecar)"
	cd apps/document-sidecar && uv run python -m grpc_tools.protoc \
		-I../../proto \
		--python_out=src/document_sidecar/proto \
		--grpc_python_out=src/document_sidecar/proto \
		--pyi_out=src/document_sidecar/proto \
		../../proto/document.proto
	@echo ">> Patch generated python imports to be package-relative"
	sed -i 's|^import ai_pb2 |from . import ai_pb2 |' apps/ai-sidecar/src/ai_sidecar/proto/ai_pb2_grpc.py
	sed -i 's|^import document_pb2 |from . import document_pb2 |' apps/ai-sidecar/src/ai_sidecar/proto/document_pb2_grpc.py 2>/dev/null || true
	sed -i 's|^import document_pb2 |from . import document_pb2 |' apps/document-sidecar/src/document_sidecar/proto/document_pb2_grpc.py

migrate:
	@echo "Migrations land in Phase 1; goose dir is /migrations."

clean:
	rm -rf bin coverage.out
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	rm -rf apps/web/.next apps/web/node_modules .pytest_cache .ruff_cache
