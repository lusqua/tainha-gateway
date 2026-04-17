![Tainha logo](https://github.com/user-attachments/assets/a1286b71-5b0b-4d1e-90c6-177dd9ca5fe5)

# Tainha Gateway

A lightweight, open-source API Gateway written in Go. Route requests, validate auth, aggregate responses from multiple services, and monitor everything — all from a single YAML config.

[![Tests](https://github.com/lusqua/tainha-gateway/actions/workflows/tests.yml/badge.svg)](https://github.com/lusqua/tainha-gateway/actions/workflows/tests.yml)
[![Docs](https://github.com/lusqua/tainha-gateway/actions/workflows/docs.yml/badge.svg)](https://lusqua.github.io/tainha-gateway)

**[Documentation](https://lusqua.github.io/tainha-gateway)** | **[Getting Started](https://lusqua.github.io/tainha-gateway/docs/getting-started)** | **[Configuration](https://lusqua.github.io/tainha-gateway/docs/configuration)**

## Features

- **Routing** — dynamic path params, query strings, configurable base path
- **Authentication** — local JWT validation (HS256) or delegate to your own auth service
- **Response Mapping** — enrich responses by aggregating data from multiple services in parallel
- **SSE** — Server-Sent Events passthrough for real-time streaming
- **Rate Limiting** — token bucket per IP with X-Forwarded-For support
- **Observability** — OpenTelemetry traces, Prometheus metrics (`/metrics`), structured JSON logs
- **Health Check** — `GET /health` for load balancers and k8s probes
- **Request Tracing** — `X-Request-ID` generated per request, propagated to backends
- **Graceful Shutdown** — SIGTERM handling with connection draining
- **Config Validation** — fail fast on startup with clear error messages

## Quick Start

```bash
git clone https://github.com/lusqua/tainha-gateway.git
cd tainha-gateway
```

Create `config/config.yaml`:

```yaml
config:
  port: 8000
  basePath: /api
  auth:
    secret: "your-secret"
    defaultProtected: false

routes:
  - method: GET
    route: /users
    service: localhost:3000
    path: /users
    public: true
```

Run:

```bash
go run cmd/gateway/main.go
```

```bash
curl http://localhost:8000/api/users
```

## Configuration

```yaml
config:
  port: 8000
  basePath: /api
  readTimeoutSec: 15
  writeTimeoutSec: 30
  idleTimeoutSec: 60

  auth:
    secret: "your-secret"           # Local JWT (HS256)
    defaultProtected: true
    # Or delegate to your auth service:
    # authService: localhost:5000
    # authPath: /auth/validate

  rateLimit:
    enabled: true
    requestsPerSec: 100
    burst: 200

  telemetry:
    enabled: true
    serviceName: tainha-gateway
    exporterEndpoint: localhost:4317   # OTLP gRPC (Jaeger, Tempo)

routes:
  - method: GET
    route: /products
    service: localhost:3000
    path: /products
    public: true
    mapping:
      - path: /categories?id={categoryId}
        service: localhost:3000
        tag: category
        removeKeyMapping: true
```

See the full [Configuration Reference](https://lusqua.github.io/tainha-gateway/docs/configuration).

## Response Mapping

Aggregate data from multiple services in a single response:

```yaml
routes:
  - method: GET
    route: /posts
    service: localhost:3000
    path: /posts
    mapping:
      - path: /comments?postId={id}
        service: localhost:3000
        tag: comments
      - path: /users/{userId}
        service: localhost:3000
        tag: author
        removeKeyMapping: true
```

The gateway fetches posts, then enriches each one with comments and author data in parallel.

## Auth Delegation

Bring your own auth service. The gateway forwards the `Authorization` header to your service — you validate with any strategy (RS256, OAuth2, API keys, etc.):

```yaml
config:
  auth:
    authService: localhost:5000
    authPath: /auth/validate
    defaultProtected: true
```

Your service returns `200` with claims → forwarded as `X-` headers. Non-200 → forwarded to client.

See the [Auth Delegation docs](https://lusqua.github.io/tainha-gateway/docs/authentication/delegation) for the full contract and examples in Go and Node.js.

## Observability

With `telemetry.enabled: true`:

- **`GET /metrics`** — Prometheus metrics (request count, latency, error rate, rate limit hits)
- **OTLP Traces** — spans for each request with child spans for proxy and mapping calls
- **Structured Logs** — JSON via `slog` with request IDs
- **W3C Trace Context** — propagated to backends automatically

## Testing

```bash
# Unit tests (102 test cases)
make test

# E2E tests with Docker (inventory system scenario)
make e2e
```

## Roadmap

- [x] Routing with path params and query strings
- [x] JWT authentication (HS256)
- [x] Auth delegation to external services
- [x] Response mapping (parallel aggregation)
- [x] Server-Sent Events
- [x] Rate limiting
- [x] OpenTelemetry traces + Prometheus metrics
- [x] Graceful shutdown
- [x] Health check
- [x] Structured logging
- [x] Config validation
- [x] Request ID tracing
- [x] Mapping cache
- [x] Circuit breaker
- [x] WebSocket support
- [x] Hot config reload (SIGHUP)
- [x] JWT with RS256 / JWK (via JWKS endpoint)

## Contributing

Contributions are welcome. Open an issue or submit a pull request.

## License

[MIT](LICENSE)
