run:
	go run cmd/gateway/main.go

# Run SSE test server
sse-server:
	@echo "Starting SSE test server on :3000/sse"
	@go run test/sseServer/sse_server.go

run-sse:
	@go run test/sseServer/sse_server.go -port=${PORT}

# Unit tests
test:
	go test ./internal/... -v

# E2E tests (requires Docker)
e2e:
	docker compose -f e2e/docker-compose.yml up --build --abort-on-container-exit --exit-code-from test-runner

e2e-down:
	docker compose -f e2e/docker-compose.yml down -v

e2e-logs:
	docker compose -f e2e/docker-compose.yml logs -f

.PHONY: run run-sse test e2e e2e-down e2e-logs
