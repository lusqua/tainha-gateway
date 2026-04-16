package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		limiter := NewRateLimiter(10, 10)
		handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("Request %d: status = %d, want 200", i, rr.Code)
			}
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		limiter := NewRateLimiter(5, 5)
		handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Exhaust the burst
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "10.0.0.1:1234"
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}

		// Next request should be rate limited
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want 429", rr.Code)
		}

		var body map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &body)
		if body["error"] != "rate limit exceeded" {
			t.Errorf("error = %v, want 'rate limit exceeded'", body["error"])
		}
		if rr.Header().Get("Retry-After") != "1" {
			t.Error("Expected Retry-After header")
		}
	})

	t.Run("different IPs have separate limits", func(t *testing.T) {
		limiter := NewRateLimiter(2, 2)
		handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Exhaust IP 1
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "1.1.1.1:1234"
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}

		// IP 2 should still work
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "2.2.2.2:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Different IP should not be rate limited, got %d", rr.Code)
		}
	})

	t.Run("respects X-Forwarded-For", func(t *testing.T) {
		limiter := NewRateLimiter(1, 1)
		handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// First request uses XFF
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "3.3.3.3, 4.4.4.4")
		req.RemoteAddr = "127.0.0.1:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("First request should pass, got %d", rr.Code)
		}

		// Second request from same XFF IP should be limited
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.Header.Set("X-Forwarded-For", "3.3.3.3")
		req2.RemoteAddr = "127.0.0.1:1234"
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)

		if rr2.Code != http.StatusTooManyRequests {
			t.Fatalf("Second request from same XFF should be limited, got %d", rr2.Code)
		}
	})
}
