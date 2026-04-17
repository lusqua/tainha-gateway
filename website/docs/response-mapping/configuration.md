---
sidebar_position: 2
slug: /response-mapping/configuration
---

# Mapping Configuration

## Basic Structure

Mappings are defined per route in the YAML config:

```yaml
routes:
  - method: GET
    route: /products
    service: localhost:3000
    path: /products
    mapping:
      - path: /categories?id={categoryId}
        service: localhost:3000
        tag: category
        removeKeyMapping: true
```

## Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Endpoint to call. `{param}` is replaced with the value from the response item |
| `service` | string | Yes | Host of the service to call (with or without protocol) |
| `tag` | string | Yes | Key name under which the mapped data is added to the response |
| `removeKeyMapping` | bool | No | If `true`, removes the original parameter key from the item |

## The `path` Field

The path defines **what to fetch** and **which field to use** from the response item.

### Query Parameter Style

```yaml
path: /comments?postId={id}
```

Given item `{"id": "1", ...}`, calls: `GET /comments?postId=1`

### Path Parameter Style

```yaml
path: /users/{authorId}
```

Given item `{"authorId": "5", ...}`, calls: `GET /users/5`

### The parameter name must match a field in the response item.

If the item doesn't have the field, the mapping is **skipped** silently for that item.

## The `tag` Field

The tag is the key name where the mapped data appears in the response:

```yaml
tag: category
```

Result: `{"id": "1", ..., "category": {"name": "Electronics"}}`

Choose descriptive tag names — they become part of your API response.

## The `removeKeyMapping` Field

Controls whether the original foreign key is kept or removed:

### `removeKeyMapping: false` (default)

```json
{"id": "1", "categoryId": "c1", "category": {"name": "Electronics"}}
```

Both `categoryId` and `category` are present.

### `removeKeyMapping: true`

```json
{"id": "1", "category": {"name": "Electronics"}}
```

`categoryId` is removed — cleaner response, no redundant data.

## The `service` Field

The service can be a different host from the primary route service:

```yaml
routes:
  - method: GET
    route: /products
    service: product-service:3000         # Primary service
    path: /products
    mapping:
      - path: /categories/{categoryId}
        service: catalog-service:3001     # Different service
        tag: category
      - path: /reviews?productId={id}
        service: review-service:3002      # Another service
        tag: reviews
```

This is how you aggregate data from multiple microservices in one request.

## Multiple Mappings

A single route can have multiple mapping rules. They all run in parallel:

```yaml
mapping:
  - path: /categories/{categoryId}
    service: catalog-service:3001
    tag: category
    removeKeyMapping: true
  - path: /reviews?productId={id}
    service: review-service:3002
    tag: reviews
    removeKeyMapping: false
  - path: /stock/{id}
    service: inventory-service:3003
    tag: stock
    removeKeyMapping: false
```

Result: each product gets `category`, `reviews`, and `stock` data attached.

## Mapping Cache

Enable caching to avoid repeated HTTP calls for the same URL:

```yaml
config:
  mappingCache:
    enabled: true
    ttlSec: 60        # Entries expire after 60 seconds
    maxSize: 1000     # Max cached entries
```

Useful when many items share the same foreign key (e.g., 100 products in 5 categories = 5 HTTP calls instead of 100).

## Validation

The gateway validates mapping config on startup and **fails fast** if:
- `path` is empty
- `service` is empty
- `tag` is empty
