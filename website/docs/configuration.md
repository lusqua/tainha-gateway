---
sidebar_position: 2
slug: /configuration
---

# Configuration

Tainha is configured through a single YAML file. By default, it looks for `./config/config.yaml`, but you can specify a custom path:

```bash
./tainha -config /path/to/config.yaml
```

Send `SIGHUP` to reload config without restarting:

```bash
kill -HUP $(pgrep tainha)
```

## Full Reference

```yaml
config:
  port: 8000
  basePath: /api
  readTimeoutSec: 15
  writeTimeoutSec: 30
  idleTimeoutSec: 60

  auth:
    secret: "your-secret-key"       # HS256 shared secret
    jwksUrl: "https://.../.well-known/jwks.json"  # RS256/ES256 via JWKS
    algorithm: RS256                # Algorithm hint (optional)
    defaultProtected: true
    authService: localhost:5000     # Or delegate to your own service
    authPath: /auth/validate

  rateLimit:
    enabled: true
    requestsPerSec: 100
    burst: 200

  mappingCache:
    enabled: true
    ttlSec: 60
    maxSize: 1000

  circuitBreaker:
    enabled: true
    maxFailures: 5
    timeoutSec: 30
    halfOpenRequests: 1

  telemetry:
    enabled: true
    serviceName: tainha-gateway
    exporterEndpoint: localhost:4317

routes:
  - method: GET
    route: /products
    service: localhost:3000
    path: /products
    public: true
    isSSE: false
    isWebSocket: false
    mapping:
      - path: /categories?id={categoryId}
        service: localhost:3000
        tag: category
        removeKeyMapping: true
```

## Config Section

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | int | `8080` | Port the gateway listens on |
| `basePath` | string | `/api` | Prefix prepended to all route paths |
| `readTimeoutSec` | int | `15` | HTTP server read timeout |
| `writeTimeoutSec` | int | `30` | HTTP server write timeout |
| `idleTimeoutSec` | int | `60` | HTTP server idle timeout |

### Auth

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `auth.secret` | string | — | HMAC secret for HS256 validation |
| `auth.jwksUrl` | string | — | JWKS endpoint URL for RS256/ES256 |
| `auth.algorithm` | string | — | Algorithm hint (optional) |
| `auth.defaultProtected` | bool | `false` | Require auth on all routes by default |
| `auth.authService` | string | — | External auth service (enables delegation) |
| `auth.authPath` | string | `/validate` | Auth service endpoint path |

**Priority:** `authService` > `jwksUrl` > `secret`. Only one is used.

### Rate Limiting

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `rateLimit.enabled` | bool | `false` | Enable per-IP rate limiting |
| `rateLimit.requestsPerSec` | int | `100` | Max requests per second per IP |
| `rateLimit.burst` | int | `requestsPerSec * 2` | Burst allowance |

### Mapping Cache

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `mappingCache.enabled` | bool | `false` | Cache mapping responses |
| `mappingCache.ttlSec` | int | `60` | Time-to-live in seconds |
| `mappingCache.maxSize` | int | `1000` | Max cached entries (LRU eviction) |

### Circuit Breaker

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `circuitBreaker.enabled` | bool | `false` | Enable per-service circuit breaker |
| `circuitBreaker.maxFailures` | int | `5` | Failures before opening circuit |
| `circuitBreaker.timeoutSec` | int | `30` | Seconds before trying half-open |
| `circuitBreaker.halfOpenRequests` | int | `1` | Probe requests in half-open state |

### Telemetry

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `telemetry.enabled` | bool | `false` | Enable OTEL metrics + traces |
| `telemetry.serviceName` | string | `tainha-gateway` | Service name in OTEL |
| `telemetry.exporterEndpoint` | string | — | OTLP gRPC endpoint for traces |

## Route Section

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `method` | string | — | HTTP method (GET, POST, PUT, DELETE, PATCH) |
| `route` | string | — | Gateway path (what clients hit) |
| `service` | string | — | Backend host (with or without protocol) |
| `path` | string | — | Backend path (supports `{param}` placeholders) |
| `public` | bool | `false` | Skip authentication for this route |
| `isSSE` | bool | `false` | Enable SSE passthrough |
| `isWebSocket` | bool | `false` | Enable WebSocket proxy |
| `mapping` | array | — | Response mapping rules |

## Route Mapping

| Field | Type | Description |
|-------|------|-------------|
| `path` | string | Endpoint to fetch, with `{param}` from response data |
| `service` | string | Service host for this mapping |
| `tag` | string | Key name to add in the response |
| `removeKeyMapping` | bool | Remove the original param key from response |

## Path Parameters

Both `route` and `path` support dynamic parameters with `{paramName}` syntax:

```yaml
routes:
  - method: GET
    route: /users/{userId}
    service: localhost:3000
    path: /users/{userId}
```

## Service URLs

```yaml
service: localhost:3000             # Defaults to http://
service: http://localhost:3000
service: https://api.example.com
```
