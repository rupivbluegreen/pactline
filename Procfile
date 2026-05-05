api: go run ./cmd/api
worker: go run ./cmd/worker
ai-sidecar: bash -c "cd apps/ai-sidecar && uv run python -m ai_sidecar.main"
document-sidecar: bash -c "cd apps/document-sidecar && uv run python -m document_sidecar.main"
web: bash -c "cd apps/web && pnpm dev"
