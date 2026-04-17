---
sidebar_position: 3
slug: /response-mapping/examples
---

# Mapping Examples

Real-world patterns for response mapping.

## E-commerce: Products with Categories

Enrich a product list with full category data from a separate service:

```yaml
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
```

**Backend returns:**
```json
[
  {"id": "1", "name": "Laptop", "price": 2500, "categoryId": "c1"},
  {"id": "2", "name": "Mouse", "price": 49, "categoryId": "c2"}
]
```

**Gateway returns:**
```json
[
  {
    "id": "1",
    "name": "Laptop",
    "price": 2500,
    "category": {"id": "c1", "name": "Electronics"}
  },
  {
    "id": "2",
    "name": "Mouse",
    "price": 49,
    "category": {"id": "c2", "name": "Accessories"}
  }
]
```

## Blog: Posts with Author and Comments

Enrich posts with data from two different services:

```yaml
routes:
  - method: GET
    route: /posts
    service: blog-service:3000
    path: /posts
    mapping:
      - path: /users/{authorId}
        service: user-service:3001
        tag: author
        removeKeyMapping: true
      - path: /comments?postId={id}
        service: comment-service:3002
        tag: comments
        removeKeyMapping: false
```

**Gateway returns:**
```json
[
  {
    "id": "1",
    "title": "Getting Started with Go",
    "author": {"name": "Alice", "avatar": "alice.jpg"},
    "comments": [
      {"text": "Great article!", "user": "Bob"},
      {"text": "Very helpful", "user": "Charlie"}
    ]
  }
]
```

The frontend renders a complete blog post card with a single API call.

## User Profile: Single Object with Orders

Mapping works on single objects too, not just arrays:

```yaml
routes:
  - method: GET
    route: /users/{userId}
    service: user-service:3000
    path: /users/{userId}
    mapping:
      - path: /orders?userId={id}
        service: order-service:3001
        tag: orders
        removeKeyMapping: false
      - path: /addresses?userId={id}
        service: address-service:3002
        tag: addresses
        removeKeyMapping: false
```

**Gateway returns:**
```json
{
  "id": "1",
  "name": "Alice",
  "email": "alice@example.com",
  "orders": [
    {"id": "o1", "product": "Laptop", "status": "shipped"},
    {"id": "o2", "product": "Mouse", "status": "delivered"}
  ],
  "addresses": [
    {"type": "home", "city": "Florianopolis"}
  ]
}
```

## Dashboard: Companies with Users

Recursive-style mapping — companies with their users:

```yaml
routes:
  - method: GET
    route: /companies
    service: company-service:3000
    path: /companies
    mapping:
      - path: /users?companyId={id}
        service: user-service:3001
        tag: employees
        removeKeyMapping: false
```

**Gateway returns:**
```json
[
  {
    "id": "1",
    "name": "Acme Corp",
    "employees": [
      {"name": "Alice", "role": "Engineer"},
      {"name": "Bob", "role": "Designer"}
    ]
  }
]
```

## Inventory: Products with Stock and Supplier

Three mappings from three different services:

```yaml
routes:
  - method: GET
    route: /inventory
    service: product-service:3000
    path: /products
    mapping:
      - path: /stock/{id}
        service: inventory-service:3001
        tag: stock
        removeKeyMapping: false
      - path: /suppliers/{supplierId}
        service: supplier-service:3002
        tag: supplier
        removeKeyMapping: true
      - path: /pricing/{id}
        service: pricing-service:3003
        tag: pricing
        removeKeyMapping: false
```

All three mappings run in parallel for each product. With 100 products, that's up to 300 goroutines running concurrently — but with connection pooling and mapping cache, the actual HTTP calls are minimized.

## Tips

**Keep mapping targets fast.** The gateway's response time = backend response time + slowest mapping. If a mapping target is slow, consider:
- Enabling mapping cache (`mappingCache.enabled: true`)
- Adding an index on the queried field in your database
- Using a lighter endpoint that returns only the fields you need

**Use `removeKeyMapping: true` for clean APIs.** Foreign keys like `categoryId` are implementation details — your API consumers care about the category name, not the ID.

**One route can have many mappings.** They all run in parallel. Adding more mappings adds latency only if the new mapping target is slower than the existing ones.
