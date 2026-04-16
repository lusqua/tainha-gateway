---
sidebar_position: 3
slug: /authentication
---

# Authentication

Tainha supports two authentication modes:

1. **Local JWT validation** — the gateway validates tokens directly using a shared secret (HS256)
2. **Auth delegation** — the gateway delegates validation to your own auth service

Both modes protect routes marked as non-public and forward user claims as headers to your backend services.

## Local JWT Validation

The simplest setup. The gateway validates JWT tokens using a shared HMAC secret.

### Configuration

```yaml
config:
  auth:
    secret: "your-secret-key"
    defaultProtected: true
```

### How it works

1. Client sends `Authorization: Bearer <token>`
2. Gateway validates the token signature with the configured secret (HS256)
3. Gateway extracts claims and forwards them as `X-` headers
4. Backend receives the request with claim headers

```mermaid
sequenceDiagram
    participant C as Client
    participant G as Gateway
    participant B as Backend

    C->>G: GET /api/users<br/>Authorization: Bearer xxx
    G->>G: Validate JWT (HS256)<br/>Extract claims
    G->>B: GET /users<br/>X-Username: alice<br/>X-Role: admin
    B-->>G: 200 OK
    G-->>C: 200 OK
```

### Public routes

Mark routes as `public: true` to skip authentication:

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

### Claim forwarding

String claims from the JWT payload are forwarded as headers with the `X-` prefix:

| JWT Claim | Forwarded Header |
|-----------|-----------------|
| `username` | `X-Username` |
| `role` | `X-Role` |
| `sub` | `X-Sub` |
| `email` | `X-Email` |

Your backend reads these headers to identify the authenticated user — no need to parse the JWT again.

### Issuer and audience validation

Optional validation can be triggered by sending headers in the request:

| Header | Validates |
|--------|-----------|
| `X-JWT-Issuer` | Checks the `iss` claim matches |
| `X-JWT-Audience` | Checks the `aud` claim matches |

These headers are removed before forwarding to the backend.

---

## Auth Delegation

For more control, delegate token validation to your own auth service. This lets you use **any auth strategy** — RS256, OAuth2, API keys, sessions, or anything else.

### Configuration

```yaml
config:
  auth:
    authService: localhost:5000
    authPath: /auth/validate
    defaultProtected: true
```

When `authService` is configured, the gateway calls your service instead of validating the token locally. The `secret` field is ignored.

### How it works

1. Client sends `Authorization: Bearer <token>`
2. Gateway forwards the Authorization header to your auth service
3. Your service validates the token and responds with claims (200) or an error
4. Gateway forwards claims as `X-` headers to the backend

```mermaid
sequenceDiagram
    participant C as Client
    participant G as Gateway
    participant A as Auth Service
    participant B as Backend

    C->>G: GET /api/users<br/>Authorization: Bearer xxx
    G->>A: GET /auth/validate<br/>Authorization: Bearer xxx
    A->>A: Validate token
    A-->>G: 200 OK<br/>{"userId":"1", "role":"admin"}
    G->>B: GET /users<br/>X-userId: 1<br/>X-role: admin
    B-->>G: 200 OK
    G-->>C: 200 OK
```

### Auth Service Contract

Your auth service must implement a single endpoint:

#### Request

```
GET /auth/validate
Authorization: Bearer <token>
```

The gateway forwards the exact `Authorization` header from the original client request.

#### Success Response (200 OK)

Return a JSON object with user claims. String values will be forwarded as `X-` headers:

```json
{
  "userId": "123",
  "username": "alice",
  "role": "admin"
}
```

This results in headers: `X-userId: 123`, `X-username: alice`, `X-role: admin`.

Returning an empty body or `{}` is also valid — the request will be forwarded without extra headers.

#### Error Response (any non-200 status)

The gateway forwards the auth service's response directly to the client:

```json
HTTP/1.1 401 Unauthorized

{
  "error": "Token expired",
  "success": false
}
```

### Example Auth Service

Here's a minimal auth service in Go that validates JWT tokens:

```go
package main

import (
    "encoding/json"
    "net/http"
    "strings"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

const secret = "your-secret-key"

func main() {
    http.HandleFunc("/auth/validate", validateHandler)
    http.HandleFunc("/auth/login", loginHandler)
    http.HandleFunc("/auth/register", registerHandler)
    http.ListenAndServe(":5000", nil)
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
    auth := r.Header.Get("Authorization")
    parts := strings.Split(auth, " ")
    if len(parts) != 2 {
        http.Error(w, `{"error":"missing token"}`, 401)
        return
    }

    token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
        return []byte(secret), nil
    })
    if err != nil || !token.Valid {
        http.Error(w, `{"error":"invalid token"}`, 401)
        return
    }

    claims := token.Claims.(jwt.MapClaims)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "userId":   claims["sub"].(string),
        "username": claims["username"].(string),
        "role":     claims["role"].(string),
    })
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
    // Your login logic here — validate credentials, return JWT
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "sub":      "1",
        "username": "alice",
        "role":     "admin",
        "exp":      time.Now().Add(24 * time.Hour).Unix(),
    })
    tokenString, _ := token.SignedString([]byte(secret))

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "token": tokenString,
    })
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
    // Your registration logic here
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "message": "User registered",
    })
}
```

### Example Auth Service in Node.js

```javascript
const express = require('express');
const jwt = require('jsonwebtoken');
const app = express();

const SECRET = 'your-secret-key';

app.get('/auth/validate', (req, res) => {
  const token = req.headers.authorization?.split(' ')[1];
  if (!token) return res.status(401).json({ error: 'Missing token' });

  try {
    const decoded = jwt.verify(token, SECRET);
    res.json({
      userId: decoded.sub,
      username: decoded.username,
      role: decoded.role,
    });
  } catch (err) {
    res.status(401).json({ error: 'Invalid token' });
  }
});

app.post('/auth/login', express.json(), (req, res) => {
  // Your login logic here
  const token = jwt.sign(
    { sub: '1', username: 'alice', role: 'admin' },
    SECRET,
    { expiresIn: '24h' }
  );
  res.json({ token });
});

app.listen(5000, () => console.log('Auth service on :5000'));
```

## Choosing a Mode

| | Local JWT | Auth Delegation |
|---|---|---|
| **Setup** | Just a secret in config | Requires running auth service |
| **Algorithms** | HS256 only | Any (your service decides) |
| **Performance** | Faster (no network call) | Extra hop per request |
| **Flexibility** | Limited to JWT claims | Full control over validation |
| **Use case** | Simple apps, prototyping | Production, custom auth |
