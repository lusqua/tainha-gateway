---
sidebar_position: 6
slug: /configuration/routes
---

# Routes Configuration

Routes define how client requests are mapped to backend services.

## Basic Route

```yaml
routes:
  - method: GET
    route: /products
    service: localhost:3000
    path: /products
```

This proxies `GET /api/products` → `GET http://localhost:3000/products`.

## Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `method` | string | required | HTTP method: GET, POST, PUT, DELETE, PATCH, HEAD |
| `route` | string | required | Gateway path (what clients hit) |
| `service` | string | required | Backend host |
| `path` | string | required | Backend endpoint path |
| `public` | bool | `false` | Skip [authentication](/docs/authentication) |
| `isSSE` | bool | `false` | Enable [SSE passthrough](/docs/sse) |
| `isWebSocket` | bool | `false` | Enable [WebSocket proxy](/docs/websocket) |
| `mapping` | array | — | [Response mapping](/docs/response-mapping) rules |

## Path Parameters

Use `{paramName}` for dynamic segments. The same parameter must appear in both `route` and `path`:

```yaml
routes:
  - method: GET
    route: /users/{userId}          # Client hits /api/users/42
    service: localhost:3000
    path: /users/{userId}           # Backend receives /users/42
```

Multiple parameters:

```yaml
routes:
  - method: GET
    route: /companies/{companyId}/users/{userId}
    service: localhost:3000
    path: /companies/{companyId}/users/{userId}
```

## Query Parameters

Query parameters from the client are forwarded automatically. You can also define them in the `path`:

```yaml
routes:
  - method: GET
    route: /company/{companyId}/users
    service: localhost:3000
    path: /users?companyId={companyId}
```

`GET /api/company/5/users` → `GET http://localhost:3000/users?companyId=5`

## Service URLs

The `service` field accepts different formats:

```yaml
service: localhost:3000             # → http://localhost:3000
service: http://localhost:3000      # → http://localhost:3000
service: https://api.example.com   # → https://api.example.com
```

Without a protocol prefix, `http://` is assumed.

## Route Types

### Standard Route

Regular HTTP proxy — request is forwarded, response is returned:

```yaml
- method: GET
  route: /users
  service: user-service:3000
  path: /users
```

### Route with Mapping

Response is enriched with data from other services. See [Response Mapping](/docs/response-mapping):

```yaml
- method: GET
  route: /products
  service: product-service:3000
  path: /products
  mapping:
    - path: /categories?id={categoryId}
      service: catalog-service:3001
      tag: category
```

### SSE Route

Server-Sent Events are streamed directly without buffering. See [SSE](/docs/sse):

```yaml
- method: GET
  route: /events
  service: event-service:3001
  path: /stream
  isSSE: true
```

### WebSocket Route

WebSocket connections are tunneled to the backend. See [WebSocket](/docs/websocket):

```yaml
- method: GET
  route: /chat
  service: chat-service:3002
  path: /ws
  isWebSocket: true
```

## Public vs Protected

With `auth.defaultProtected: true`, all routes require JWT unless marked `public`:

```yaml
routes:
  # Public — no token needed
  - method: GET
    route: /products
    service: localhost:3000
    path: /products
    public: true

  # Protected — requires valid JWT
  - method: POST
    route: /orders
    service: localhost:3000
    path: /orders
```

## Validation

On startup, the gateway rejects configs with:

- Missing `method`, `route`, `service`, or `path`
- Invalid HTTP method (must be GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)
- Duplicate routes (same method + route combination)
