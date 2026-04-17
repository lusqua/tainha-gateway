---
sidebar_position: 4
slug: /response-mapping
---

# Response Mapping

## The Problem

In a microservices architecture, data is spread across multiple services. A single page on your frontend might need data from 3 or 4 services to render. Without an API gateway, your frontend has to do this:

```mermaid
sequenceDiagram
    participant F as Frontend
    participant P as Products API
    participant C as Categories API
    participant R as Reviews API

    F->>P: GET /products
    P-->>F: [{id:1, categoryId:"c1", ...}]
    F->>C: GET /categories/c1
    C-->>F: {name:"Electronics"}
    F->>C: GET /categories/c2
    C-->>F: {name:"Accessories"}
    F->>R: GET /reviews?productId=1
    R-->>F: [{rating:5, ...}]
    F->>R: GET /reviews?productId=2
    R-->>F: [{rating:4, ...}]
    Note over F: 6 requests to render one page
```

This creates problems:
- **Slow** — sequential requests add up. 6 requests at 50ms each = 300ms minimum
- **Complex frontend** — your app needs to orchestrate multiple calls, handle partial failures, merge data
- **Over-fetching** — every client (web, mobile, CLI) repeats the same aggregation logic
- **Chatty network** — especially bad on mobile networks with high latency

## The Solution

Response mapping moves the aggregation logic into the gateway. Your frontend makes **one request**, and the gateway fetches and merges everything:

```mermaid
sequenceDiagram
    participant F as Frontend
    participant G as Gateway
    participant P as Products API
    participant C as Categories API
    participant R as Reviews API

    F->>G: GET /api/products
    G->>P: GET /products
    P-->>G: [{id:1, categoryId:"c1", ...}]
    par Parallel
        G->>C: GET /categories?id=c1
        G->>C: GET /categories?id=c2
        G->>R: GET /reviews?productId=1
        G->>R: GET /reviews?productId=2
    end
    C-->>G: {name:"Electronics"}
    C-->>G: {name:"Accessories"}
    R-->>G: [{rating:5}]
    R-->>G: [{rating:4}]
    G-->>F: Complete enriched response
    Note over F: 1 request, all data included
```

**Benefits:**
- **One request** — frontend gets everything in a single call
- **Parallel** — all mapping requests run concurrently, so total time = slowest mapping
- **Simple frontend** — no orchestration logic, just render the data
- **Cacheable** — mapping cache avoids refetching data that doesn't change
- **Resilient** — if a mapping fails, the item is returned without it (no error to the client)

## How It Looks

A product list without mapping:

```json
[
  {"id": "1", "name": "Laptop", "categoryId": "c1", "price": 2500},
  {"id": "2", "name": "Mouse", "categoryId": "c2", "price": 49}
]
```

The same product list with mapping (category enriched, foreign key removed):

```json
[
  {
    "id": "1",
    "name": "Laptop",
    "price": 2500,
    "category": {"id": "c1", "name": "Electronics", "description": "Electronic devices"}
  },
  {
    "id": "2",
    "name": "Mouse",
    "price": 49,
    "category": {"id": "c2", "name": "Accessories", "description": "Computer accessories"}
  }
]
```

The `categoryId` foreign key is gone and replaced with the full category object. One request, complete data.

## Next Steps

- [How It Works](/docs/response-mapping/how-it-works) — understand the mechanics step by step
- [Configuration](/docs/response-mapping/configuration) — YAML reference and field descriptions
- [Examples](/docs/response-mapping/examples) — real-world patterns and use cases
