---
sidebar_position: 3
slug: /authentication
---

# Authentication

Tainha supports three authentication modes:

1. **[Local JWT (HS256)](/docs/authentication/local-jwt)** — validate tokens with a shared secret
2. **[JWKS (RS256/ES256)](/docs/authentication/jwks)** — validate tokens with public keys from a JWKS endpoint
3. **[Auth Delegation](/docs/authentication/delegation)** — delegate validation to your own auth service

All modes protect routes marked as non-public and forward user claims as `X-` headers to your backend services.

## Quick Comparison

| | Local JWT | JWKS | Auth Delegation |
|---|---|---|---|
| **Setup** | Secret in config | JWKS URL in config | Running auth service |
| **Algorithms** | HS256 | RS256, ES256, etc. | Any |
| **Performance** | Fastest | Fast (keys cached) | Extra hop per request |
| **Providers** | Custom | Auth0, Keycloak, Firebase, Cognito | Any |
| **Use case** | Prototyping, simple apps | Standard identity providers | Full custom auth |

## Priority

If multiple options are configured:

```
authService > jwksUrl > secret
```

## Public Routes

Mark routes as `public: true` to skip authentication:

```yaml
routes:
  - method: GET
    route: /products
    service: localhost:3000
    path: /products
    public: true

  - method: GET
    route: /orders
    service: localhost:3000
    path: /orders
    # requires auth (default)
```

## Claim Forwarding

In all modes, string claims are forwarded to your backend as `X-` headers:

| JWT Claim | Forwarded Header |
|-----------|-----------------|
| `username` | `X-Username` |
| `role` | `X-Role` |
| `sub` | `X-Sub` |
| `email` | `X-Email` |

Your backend reads these headers to identify the user — no need to parse the token again.
