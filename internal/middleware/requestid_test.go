package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID(t *testing.T) {
	t.Run("generates ID when not present", func(t *testing.T) {
		handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(HeaderRequestID) == "" {
				t.Error("Expected request ID to be set on request")
			}
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Header().Get(HeaderRequestID) == "" {
			t.Error("Expected request ID in response header")
		}
	})

	t.Run("preserves existing ID", func(t *testing.T) {
		handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(HeaderRequestID) != "my-custom-id" {
				t.Errorf("Expected preserved ID, got %q", r.Header.Get(HeaderRequestID))
			}
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(HeaderRequestID, "my-custom-id")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Header().Get(HeaderRequestID) != "my-custom-id" {
			t.Errorf("Expected 'my-custom-id' in response, got %q", rr.Header().Get(HeaderRequestID))
		}
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		for i := 0; i < 100; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			id := rr.Header().Get(HeaderRequestID)
			if ids[id] {
				t.Fatalf("Duplicate request ID generated: %s", id)
			}
			ids[id] = true
		}
	})
}
