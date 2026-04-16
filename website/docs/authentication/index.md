---
sidebar_position: 3
slug: /authentication
---

# Authentication

Tainha supports two authentication modes:

1. **[Local JWT](/docs/authentication/local-jwt)** — the gateway validates tokens directly using a shared secret (HS256)
2. **[Auth Delegation](/docs/authentication/delegation)** — the gateway delegates validation to your own auth service

Both modes protect routes marked as non-public and forward user claims as headers to your backend services.

## Quick Comparison

| | Local JWT | Auth Delegation |
|---|---|---|
| **Setup** | Just a secret in config | Requires running auth service |
| **Algorithms** | HS256 only | Any (your service decides) |
| **Performance** | Faster (no network call) | Extra hop per request |
| **Flexibility** | Limited to JWT claims | Full control over validation |
| **Use case** | Simple apps, prototyping | Production, custom auth |

## Public Routes

Both modes respect the `public` flag. Mark routes as `public: true` to skip authentication entirely:

```yaml
routes:
  - method: GET
    route: /products
    service: localhost:3000
    path: /products
    public: true          # No token required

  - method: GET
    route: /orders
    service: localhost:3000
    path: /orders
    # public: false       # Token required (default)
```

## Claim Forwarding

In both modes, authenticated user data is forwarded to your backend as `X-` headers. Your backend reads these headers to identify the user — no need to parse the token again.

| Source Claim | Forwarded Header |
|-----------|-----------------|
| `username` | `X-Username` |
| `role` | `X-Role` |
| `sub` | `X-Sub` |
| `email` | `X-Email` |
