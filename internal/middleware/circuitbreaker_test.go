package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCircuitBreaker(t *testing.T) {
	t.Run("allows requests when closed", func(t *testing.T) {
		cb := NewCircuitBreaker(3, 10, 1)

		if !cb.Allow("backend") {
			t.Error("Expected request to be allowed when circuit is closed")
		}
	})

	t.Run("opens after max failures", func(t *testing.T) {
		cb := NewCircuitBreaker(3, 10, 1)

		for i := 0; i < 3; i++ {
			cb.RecordFailure("backend")
		}

		if cb.Allow("backend") {
			t.Error("Expected circuit to be open after max failures")
		}
	})

	t.Run("different services are independent", func(t *testing.T) {
		cb := NewCircuitBreaker(2, 10, 1)

		cb.RecordFailure("serviceA")
		cb.RecordFailure("serviceA")

		if cb.Allow("serviceA") {
			t.Error("Expected serviceA circuit to be open")
		}
		if !cb.Allow("serviceB") {
			t.Error("Expected serviceB circuit to still be closed")
		}
	})

	t.Run("success resets failure count", func(t *testing.T) {
		cb := NewCircuitBreaker(3, 10, 1)

		cb.RecordFailure("backend")
		cb.RecordFailure("backend")
		cb.RecordSuccess("backend")
		cb.RecordFailure("backend")

		// Should still be closed (only 1 failure since last success)
		if !cb.Allow("backend") {
			t.Error("Expected circuit to be closed after success reset")
		}
	})

	t.Run("middleware returns 503 when open", func(t *testing.T) {
		cb := NewCircuitBreaker(1, 10, 1)
		cb.RecordFailure("test-service")

		handler := cb.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called when circuit is open")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Backend-Service", "test-service")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rr.Code)
		}

		var body map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &body)
		if body["error"] != "service unavailable (circuit breaker open)" {
			t.Errorf("error = %v", body["error"])
		}
	})

	t.Run("middleware passes through without X-Backend-Service", func(t *testing.T) {
		cb := NewCircuitBreaker(1, 10, 1)
		called := false
		handler := cb.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if !called {
			t.Error("Handler should be called without X-Backend-Service")
		}
	})
}
