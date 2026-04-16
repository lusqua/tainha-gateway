package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateWithService(t *testing.T) {
	t.Run("valid token - auth service returns 200 with claims", func(t *testing.T) {
		authService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer valid-token" {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid", Success: false})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"userId":   "123",
				"username": "alice",
				"role":     "admin",
			})
		}))
		defer authService.Close()

		var capturedHeaders http.Header
		handler := ValidateWithService(authService.URL, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedHeaders = r.Header.Clone()
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if capturedHeaders.Get("X-userId") != "123" {
			t.Errorf("X-userId = %q, want 123", capturedHeaders.Get("X-userId"))
		}
		if capturedHeaders.Get("X-username") != "alice" {
			t.Errorf("X-username = %q, want alice", capturedHeaders.Get("X-username"))
		}
		if capturedHeaders.Get("X-role") != "admin" {
			t.Errorf("X-role = %q, want admin", capturedHeaders.Get("X-role"))
		}
	})

	t.Run("invalid token - auth service returns 401", func(t *testing.T) {
		authService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "invalid token",
				"success": false,
			})
		}))
		defer authService.Close()

		handler := ValidateWithService(authService.URL, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("next handler should not be called for invalid token")
		}))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer bad-token")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rr.Code)
		}

		var body map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &body)
		if body["error"] != "invalid token" {
			t.Errorf("error = %v, want 'invalid token'", body["error"])
		}
	})

	t.Run("missing authorization header", func(t *testing.T) {
		authService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("auth service should not be called without auth header")
		}))
		defer authService.Close()

		handler := ValidateWithService(authService.URL, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("next handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/protected", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rr.Code)
		}
	})

	t.Run("auth service unavailable", func(t *testing.T) {
		handler := ValidateWithService("http://localhost:1/validate", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("next handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer some-token")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rr.Code)
		}
	})

	t.Run("auth service returns 200 with empty body", func(t *testing.T) {
		authService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer authService.Close()

		handler := ValidateWithService(authService.URL, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
	})

	t.Run("auth service forwards original authorization header", func(t *testing.T) {
		var receivedAuth string
		authService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAuth = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusOK)
		}))
		defer authService.Close()

		handler := ValidateWithService(authService.URL, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer my-specific-token")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if receivedAuth != "Bearer my-specific-token" {
			t.Errorf("auth service received Authorization = %q, want 'Bearer my-specific-token'", receivedAuth)
		}
	})
}
