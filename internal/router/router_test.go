package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aguiar-sh/tainha/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

func TestCORSMiddleware(t *testing.T) {
	cfg := &config.Config{
		BaseConfig: config.BaseConfig{
			BasePath: "/api",
			Auth:     config.AuthConfig{Secret: "secret", DefaultProtected: false},
		},
		Routes: []config.Route{
			{Method: "GET", Route: "/test", Service: "localhost:9999", Path: "/test", Public: true},
		},
	}

	backend := startMockBackend(t, `{"ok":true}`)
	cfg.Routes[0].Service = stripScheme(backend.URL)

	r, err := SetupRouter(cfg)
	if err != nil {
		t.Fatalf("SetupRouter() error = %v", err)
	}

	t.Run("CORS headers present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
		}
		if got := rr.Header().Get("Access-Control-Allow-Methods"); got == "" {
			t.Error("Access-Control-Allow-Methods header missing")
		}
	})

	t.Run("OPTIONS preflight returns 200", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/test", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("OPTIONS status = %d, want 200", rr.Code)
		}
	})
}

func TestRouterProxying(t *testing.T) {
	t.Run("basic proxy to backend", func(t *testing.T) {
		backend := startMockBackend(t, `{"message":"hello"}`)

		cfg := &config.Config{
			BaseConfig: config.BaseConfig{
				BasePath: "/api",
				Auth:     config.AuthConfig{DefaultProtected: false},
			},
			Routes: []config.Route{
				{Method: "GET", Route: "/hello", Service: stripScheme(backend.URL), Path: "/hello", Public: true},
			},
		}

		r, _ := SetupRouter(cfg)
		req := httptest.NewRequest("GET", "/api/hello", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rr.Code)
		}

		var body map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &body)
		if body["message"] != "hello" {
			t.Errorf("body = %v, want message=hello", body)
		}
	})

	t.Run("path params forwarded", func(t *testing.T) {
		var receivedPath string
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedPath = r.URL.Path
			json.NewEncoder(w).Encode(map[string]string{"id": "42"})
		}))
		t.Cleanup(backend.Close)

		cfg := &config.Config{
			BaseConfig: config.BaseConfig{
				BasePath: "/api",
				Auth:     config.AuthConfig{DefaultProtected: false},
			},
			Routes: []config.Route{
				{Method: "GET", Route: "/users/{userId}", Service: stripScheme(backend.URL), Path: "/users/{userId}", Public: true},
			},
		}

		r, _ := SetupRouter(cfg)
		req := httptest.NewRequest("GET", "/api/users/42", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rr.Code)
		}
		if receivedPath != "/users/42" {
			t.Errorf("backend received path = %q, want /users/42", receivedPath)
		}
	})

	t.Run("backend error forwarded", func(t *testing.T) {
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		}))
		t.Cleanup(backend.Close)

		cfg := &config.Config{
			BaseConfig: config.BaseConfig{
				BasePath: "/api",
				Auth:     config.AuthConfig{DefaultProtected: false},
			},
			Routes: []config.Route{
				{Method: "GET", Route: "/missing", Service: stripScheme(backend.URL), Path: "/missing", Public: true},
			},
		}

		r, _ := SetupRouter(cfg)
		req := httptest.NewRequest("GET", "/api/missing", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rr.Code)
		}
	})
}

func TestRouterWithAuth(t *testing.T) {
	const secret = "test-secret-key"

	backend := startMockBackend(t, `{"data":"protected"}`)

	cfg := &config.Config{
		BaseConfig: config.BaseConfig{
			BasePath: "/api",
			Auth:     config.AuthConfig{Secret: secret, DefaultProtected: true},
		},
		Routes: []config.Route{
			{Method: "GET", Route: "/protected", Service: stripScheme(backend.URL), Path: "/protected"},
			{Method: "GET", Route: "/public", Service: stripScheme(backend.URL), Path: "/public", Public: true},
		},
	}

	r, _ := SetupRouter(cfg)

	t.Run("protected route without token returns 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/protected", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rr.Code)
		}
	})

	t.Run("protected route with valid token returns 200", func(t *testing.T) {
		token := generateTestToken(secret, map[string]interface{}{
			"username": "testuser",
			"role":     "admin",
		})

		req := httptest.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rr.Code)
		}
	})

	t.Run("protected route with invalid token returns 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rr.Code)
		}
	})

	t.Run("public route without token returns 200", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/public", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rr.Code)
		}
	})
}

func TestRouterWithMapping(t *testing.T) {
	// Backend that serves both products and categories
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/products":
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "1", "name": "Product A", "categoryId": "c1"},
			})
		case r.URL.Path == "/categories":
			catId := r.URL.Query().Get("id")
			json.NewEncoder(w).Encode(map[string]string{"id": catId, "name": "Electronics"})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(backend.Close)

	host := stripScheme(backend.URL)
	cfg := &config.Config{
		BaseConfig: config.BaseConfig{
			BasePath: "/api",
			Auth:     config.AuthConfig{DefaultProtected: false},
		},
		Routes: []config.Route{
			{
				Method:  "GET",
				Route:   "/products",
				Service: host,
				Path:    "/products",
				Public:  true,
				Mapping: []config.RouteMapping{
					{
						Path:             "/categories?id={categoryId}",
						Service:          backend.URL,
						Tag:              "category",
						RemoveKeyMapping: true,
					},
				},
			},
		},
	}

	r, _ := SetupRouter(cfg)
	req := httptest.NewRequest("GET", "/api/products", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 product, got %d", len(result))
	}

	if _, ok := result[0]["category"]; !ok {
		t.Error("Expected 'category' mapping in response")
	}
	if _, ok := result[0]["categoryId"]; ok {
		t.Error("Expected 'categoryId' to be removed (removeKeyMapping: true)")
	}
}

// helpers

func startMockBackend(t *testing.T, responseBody string) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(responseBody))
	}))
	t.Cleanup(s.Close)
	return s
}

func stripScheme(url string) string {
	// Remove http:// prefix for the config (PathProtocol will add it back)
	if len(url) > 7 && url[:7] == "http://" {
		return url[7:]
	}
	return url
}

func generateTestToken(secret string, claims map[string]interface{}) string {
	if claims["exp"] == nil {
		claims["exp"] = time.Now().Add(time.Hour).Unix()
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}
