---
sidebar_position: 2
slug: /configuration
---

# Configuration

Tainha is configured through a single YAML file. By default, it looks for `./config/config.yaml`, but you can specify a custom path:

```bash
./tainha -config /path/to/config.yaml
```

## Full Reference

```yaml
config:
  port: 8000                        # Gateway listen port
  basePath: /api                    # Prefix for all routes
  auth:
    secret: "your-secret-key"       # JWT signing secret (HS256)
    defaultProtected: true          # Require auth on all routes by default
    authService: localhost:5000     # (Optional) External auth service
    authPath: /auth/validate        # (Optional) Auth service endpoint

routes:
  - method: GET                     # HTTP method
    route: /products                # Client-facing path (gateway)
    service: localhost:3000         # Backend service host
    path: /products                 # Backend endpoint path
    public: true                    # Override auth (skip validation)
    isSSE: false                    # Enable SSE streaming
    mapping:                        # Response enrichment (optional)
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
| `auth.secret` | string | — | HMAC secret for local JWT validation |
| `auth.defaultProtected` | bool | `false` | Whether routes require auth by default |
| `auth.authService` | string | — | External auth service host (enables delegation) |
| `auth.authPath` | string | `/validate` | Endpoint path on the auth service |

## Route Section

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `method` | string | — | HTTP method (GET, POST, PUT, DELETE) |
| `route` | string | — | Gateway path (what clients hit) |
| `service` | string | — | Backend host (with or without protocol) |
| `path` | string | — | Backend path (supports `{param}` placeholders) |
| `public` | bool | `false` | Skip authentication for this route |
| `isSSE` | bool | `false` | Enable SSE passthrough |
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
    route: /users/{userId}          # Client hits /api/users/42
    service: localhost:3000
    path: /users/{userId}           # Proxied to backend as /users/42
```

## Service URLs

The `service` field accepts URLs with or without protocol:

```yaml
# All valid:
service: localhost:3000             # Defaults to http://
service: http://localhost:3000
service: https://api.example.com
```

## Examples

### Simple proxy

```yaml
routes:
  - method: GET
    route: /products
    service: localhost:3000
    path: /products
    public: true
```

### Protected route with path params

```yaml
routes:
  - method: GET
    route: /users/{userId}
    service: localhost:3000
    path: /users/{userId}
    # public: false (default — requires JWT)
```

### Route with response mapping

```yaml
routes:
  - method: GET
    route: /posts
    service: localhost:3000
    path: /posts
    public: true
    mapping:
      - path: /comments?postId={id}
        service: localhost:3000
        tag: comments
        removeKeyMapping: false
      - path: /users/{userId}
        service: localhost:3000
        tag: author
        removeKeyMapping: true
```
