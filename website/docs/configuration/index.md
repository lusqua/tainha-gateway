---
sidebar_position: 2
slug: /configuration
---

# Configuration

Tainha is configured through a single YAML file. Everything — routing, auth, rate limiting, observability — is defined declaratively.

## Quick Start

```bash
# Default path
./tainha

# Custom path
./tainha -config /path/to/config.yaml

# Reload without restart
kill -HUP $(pgrep tainha)
```

## Minimal Config

The simplest working config — one route, no auth:

```yaml
config:
  port: 8000
  basePath: /api

routes:
  - method: GET
    route: /users
    service: localhost:3000
    path: /users
```

This proxies `GET /api/users` to `http://localhost:3000/users`.

## Full Example

A production-ready config using all features:

```yaml
config:
  port: 8000
  basePath: /api
  readTimeoutSec: 15
  writeTimeoutSec: 30
  idleTimeoutSec: 60

  auth:
    jwksUrl: "https://auth.example.com/.well-known/jwks.json"
    defaultProtected: true

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

  telemetry:
    enabled: true
    serviceName: tainha-gateway
    exporterEndpoint: jaeger:4317

routes:
  - method: GET
    route: /products
    service: product-service:3000
    path: /products
    public: true
    mapping:
      - path: /categories?id={categoryId}
        service: catalog-service:3001
        tag: category
        removeKeyMapping: true

  - method: GET
    route: /orders
    service: order-service:3002
    path: /orders

  - method: GET
    route: /events
    service: event-service:3003
    path: /stream
    isSSE: true
    public: true

  - method: GET
    route: /chat
    service: chat-service:3004
    path: /ws
    isWebSocket: true
```

## Sections

The config is divided into two top-level blocks:

### `config` — Gateway Settings

Global settings that apply to the entire gateway. See sub-pages for details:

- [Server](/docs/configuration/server) — port, basePath, timeouts
- [Auth](/docs/configuration/auth) — JWT, JWKS, delegation
- [Rate Limiting](/docs/configuration/rate-limiting) — per-IP throttling
- [Resilience](/docs/configuration/resilience) — circuit breaker, mapping cache
- [Telemetry](/docs/configuration/telemetry) — OTEL traces, Prometheus metrics

### `routes` — Route Definitions

Each route maps a client-facing URL to a backend service. See:

- [Routes](/docs/configuration/routes) — method, path, service, params
- [Response Mapping](/docs/response-mapping/configuration) — enrich responses from multiple services

## Config Validation

The gateway validates the config on startup and **fails fast** with clear error messages:

- Missing required fields (`method`, `route`, `service`, `path`)
- Invalid HTTP methods
- Duplicate routes
- Auth enabled without secret, JWKS URL, or auth service
- Mapping rules without `path`, `service`, or `tag`

## Hot Reload

Send `SIGHUP` to reload config without dropping connections:

```bash
kill -HUP $(pgrep tainha)
```

- Routes are swapped atomically
- If the new config is invalid, the old one stays active
- Zero downtime — in-flight requests complete with the old config
