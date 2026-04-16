---
sidebar_position: 5
slug: /sse
---

# Server-Sent Events (SSE)

Tainha supports proxying Server-Sent Events from your backend to clients. When a route is marked as `isSSE: true`, the gateway streams the response directly without buffering.

## Configuration

```yaml
routes:
  - method: GET
    route: /events
    service: localhost:3001
    path: /sse
    isSSE: true
    public: true
```

## How It Works

For SSE routes, the gateway:

1. Receives the client connection
2. Sets CORS headers for cross-origin streaming
3. Proxies the request directly to the backend
4. Streams the response in real-time (no buffering or recording)

```
Client                    Gateway                   Backend SSE
  │                         │                          │
  │ GET /api/events         │                          │
  │────────────────────────>│ GET /sse                 │
  │                         │─────────────────────────>│
  │                         │                          │
  │  data: {"event":1}      │  data: {"event":1}       │
  │<────────────────────────│<─────────────────────────│
  │                         │                          │
  │  data: {"event":2}      │  data: {"event":2}       │
  │<────────────────────────│<─────────────────────────│
  │                         │                          │
  │  ...stream continues... │  ...stream continues...  │
```

## Key Differences from Regular Routes

| | Regular Routes | SSE Routes |
|---|---|---|
| Response | Buffered, then sent | Streamed in real-time |
| Mapping | Supported | Not supported |
| CORS | Via middleware | Extra headers added |
| Connection | Short-lived | Long-lived |

## Example Backend

A minimal SSE server in Go:

```go
func sseHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher := w.(http.Flusher)

    for i := 0; i < 10; i++ {
        fmt.Fprintf(w, "data: {\"count\": %d}\n\n", i)
        flusher.Flush()
        time.Sleep(1 * time.Second)
    }
}
```

## Notes

- SSE routes skip response mapping (the response is streamed, not buffered)
- Authentication still works on SSE routes (unless marked as `public: true`)
- The gateway adds CORS headers to support cross-origin SSE clients
