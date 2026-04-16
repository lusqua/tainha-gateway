package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

// Mock services for e2e testing: inventory system with products, categories, and users.
// Each service runs on a different port, configured via environment variables.

// --- Data ---

var (
	products = []map[string]interface{}{
		{"id": "1", "name": "Laptop", "price": 2500.00, "categoryId": "1", "stock": 15},
		{"id": "2", "name": "Mouse", "price": 49.90, "categoryId": "2", "stock": 200},
		{"id": "3", "name": "Monitor", "price": 1200.00, "categoryId": "1", "stock": 30},
	}

	categories = []map[string]interface{}{
		{"id": "1", "name": "Electronics", "description": "Electronic devices"},
		{"id": "2", "name": "Accessories", "description": "Computer accessories"},
	}

	users = []map[string]interface{}{
		{"id": "1", "name": "Alice", "email": "alice@example.com", "role": "admin"},
		{"id": "2", "name": "Bob", "email": "bob@example.com", "role": "viewer"},
	}

	orders = []map[string]interface{}{
		{"id": "1", "productId": "1", "userId": "1", "quantity": 2, "status": "shipped"},
		{"id": "2", "productId": "2", "userId": "1", "quantity": 5, "status": "pending"},
		{"id": "3", "productId": "3", "userId": "2", "quantity": 1, "status": "delivered"},
	}

	mu               sync.RWMutex
	receivedHeaders  = make(map[string]map[string]string) // track headers per request path
	requestLog       []RequestLogEntry
	requestLogMu     sync.Mutex
)

type RequestLogEntry struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}

	mux := http.NewServeMux()

	// Products
	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		logRequest(r)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(products)
	})
	mux.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		logRequest(r)
		id := strings.TrimPrefix(r.URL.Path, "/products/")
		for _, p := range products {
			if p["id"] == id {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(p)
				return
			}
		}
		http.Error(w, `{"error":"product not found"}`, http.StatusNotFound)
	})

	// Categories
	mux.HandleFunc("/categories", func(w http.ResponseWriter, r *http.Request) {
		logRequest(r)
		w.Header().Set("Content-Type", "application/json")
		id := r.URL.Query().Get("id")
		if id != "" {
			for _, c := range categories {
				if c["id"] == id {
					json.NewEncoder(w).Encode(c)
					return
				}
			}
			http.Error(w, `{"error":"category not found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(categories)
	})
	mux.HandleFunc("/categories/", func(w http.ResponseWriter, r *http.Request) {
		logRequest(r)
		id := strings.TrimPrefix(r.URL.Path, "/categories/")
		for _, c := range categories {
			if c["id"] == id {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(c)
				return
			}
		}
		http.Error(w, `{"error":"category not found"}`, http.StatusNotFound)
	})

	// Users
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		logRequest(r)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	})
	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		logRequest(r)
		id := strings.TrimPrefix(r.URL.Path, "/users/")
		for _, u := range users {
			if u["id"] == id {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(u)
				return
			}
		}
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
	})

	// Orders
	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		logRequest(r)
		w.Header().Set("Content-Type", "application/json")
		userId := r.URL.Query().Get("userId")
		if userId != "" {
			var filtered []map[string]interface{}
			for _, o := range orders {
				if o["userId"] == userId {
					filtered = append(filtered, o)
				}
			}
			json.NewEncoder(w).Encode(filtered)
			return
		}
		json.NewEncoder(w).Encode(orders)
	})

	// SSE endpoint
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		logRequest(r)
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		for i := 0; i < 3; i++ {
			fmt.Fprintf(w, "data: {\"event\":%d,\"message\":\"stock update\"}\n\n", i+1)
			flusher.Flush()
		}
	})

	// Introspection endpoints for e2e test assertions
	mux.HandleFunc("/_requests", func(w http.ResponseWriter, r *http.Request) {
		requestLogMu.Lock()
		defer requestLogMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(requestLog)
	})
	mux.HandleFunc("/_requests/reset", func(w http.ResponseWriter, r *http.Request) {
		requestLogMu.Lock()
		defer requestLogMu.Unlock()
		requestLog = nil
		w.WriteHeader(http.StatusOK)
	})

	log.Printf("Mock service starting on :%s\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func logRequest(r *http.Request) {
	headers := make(map[string]string)
	for key := range r.Header {
		if strings.HasPrefix(key, "X-") {
			headers[key] = r.Header.Get(key)
		}
	}

	mu.Lock()
	receivedHeaders[r.URL.Path] = headers
	mu.Unlock()

	entry := RequestLogEntry{
		Method:  r.Method,
		Path:    r.URL.RequestURI(),
		Headers: headers,
	}
	requestLogMu.Lock()
	requestLog = append(requestLog, entry)
	requestLogMu.Unlock()

	log.Printf("[%s] %s headers=%v", r.Method, r.URL.RequestURI(), headers)
}
