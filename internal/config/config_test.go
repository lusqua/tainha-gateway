package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		yaml := `
config:
  port: 9000
  basePath: /api
  auth:
    secret: "test-secret"
    defaultProtected: true

routes:
  - method: GET
    route: /products
    service: localhost:4000
    path: /products
    public: true
  - method: GET
    route: /users/{userId}
    service: localhost:4000
    path: /users/{userId}
    mapping:
      - path: /orders?userId={id}
        service: localhost:4001
        tag: orders
        removeKeyMapping: false
`
		path := writeTempConfig(t, yaml)
		cfg, err := LoadConfig(path)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if cfg.BaseConfig.Port != 9000 {
			t.Errorf("Port = %d, want 9000", cfg.BaseConfig.Port)
		}
		if cfg.BaseConfig.BasePath != "/api" {
			t.Errorf("BasePath = %q, want /api", cfg.BaseConfig.BasePath)
		}
		if cfg.BaseConfig.Auth.Secret != "test-secret" {
			t.Errorf("Auth.Secret = %q, want test-secret", cfg.BaseConfig.Auth.Secret)
		}
		if !cfg.BaseConfig.Auth.DefaultProtected {
			t.Error("Auth.DefaultProtected = false, want true")
		}
		if len(cfg.Routes) != 2 {
			t.Fatalf("len(Routes) = %d, want 2", len(cfg.Routes))
		}

		route := cfg.Routes[0]
		if route.Method != "GET" || route.Route != "/products" || route.Service != "localhost:4000" {
			t.Errorf("Route[0] = %+v, unexpected values", route)
		}
		if !route.Public {
			t.Error("Route[0].Public = false, want true")
		}

		route = cfg.Routes[1]
		if len(route.Mapping) != 1 {
			t.Fatalf("Route[1].Mapping len = %d, want 1", len(route.Mapping))
		}
		if route.Mapping[0].Tag != "orders" {
			t.Errorf("Route[1].Mapping[0].Tag = %q, want orders", route.Mapping[0].Tag)
		}
	})

	t.Run("sse route", func(t *testing.T) {
		yaml := `
config:
  port: 8000
  basePath: /api
  auth:
    secret: "secret"
    defaultProtected: false

routes:
  - method: GET
    route: /events
    service: localhost:3001
    path: /sse
    isSSE: true
    public: true
`
		path := writeTempConfig(t, yaml)
		cfg, err := LoadConfig(path)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if !cfg.Routes[0].IsSSE {
			t.Error("Route.IsSSE = false, want true")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/config.yaml")
		if err == nil {
			t.Error("LoadConfig() expected error for missing file")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		path := writeTempConfig(t, `{{{invalid`)
		_, err := LoadConfig(path)
		if err == nil {
			t.Error("LoadConfig() expected error for invalid yaml")
		}
	})

	t.Run("applies default timeouts", func(t *testing.T) {
		yaml := `
config:
  port: 8000
  basePath: /api
  auth:
    defaultProtected: false
routes:
  - method: GET
    route: /test
    service: localhost:3000
    path: /test
`
		path := writeTempConfig(t, yaml)
		cfg, err := LoadConfig(path)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if cfg.BaseConfig.ReadTimeoutSec != 15 {
			t.Errorf("ReadTimeoutSec = %d, want 15", cfg.BaseConfig.ReadTimeoutSec)
		}
		if cfg.BaseConfig.WriteTimeoutSec != 30 {
			t.Errorf("WriteTimeoutSec = %d, want 30", cfg.BaseConfig.WriteTimeoutSec)
		}
		if cfg.BaseConfig.IdleTimeoutSec != 60 {
			t.Errorf("IdleTimeoutSec = %d, want 60", cfg.BaseConfig.IdleTimeoutSec)
		}
	})

	t.Run("rate limit config", func(t *testing.T) {
		yaml := `
config:
  port: 8000
  basePath: /api
  auth:
    defaultProtected: false
  rateLimit:
    enabled: true
    requestsPerSec: 50
    burst: 100
routes:
  - method: GET
    route: /test
    service: localhost:3000
    path: /test
`
		path := writeTempConfig(t, yaml)
		cfg, err := LoadConfig(path)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if !cfg.BaseConfig.RateLimit.Enabled {
			t.Error("RateLimit.Enabled = false, want true")
		}
		if cfg.BaseConfig.RateLimit.RequestsPerSec != 50 {
			t.Errorf("RequestsPerSec = %d, want 50", cfg.BaseConfig.RateLimit.RequestsPerSec)
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("no routes", func(t *testing.T) {
		yaml := `
config:
  port: 8000
  basePath: /api
  auth:
    defaultProtected: false
routes: []
`
		path := writeTempConfig(t, yaml)
		_, err := LoadConfig(path)
		if err == nil {
			t.Error("Expected validation error for no routes")
		}
	})

	t.Run("missing route fields", func(t *testing.T) {
		yaml := `
config:
  port: 8000
  basePath: /api
  auth:
    defaultProtected: false
routes:
  - method: GET
    route: ""
    service: ""
    path: ""
`
		path := writeTempConfig(t, yaml)
		_, err := LoadConfig(path)
		if err == nil {
			t.Error("Expected validation error for missing fields")
		}
	})

	t.Run("invalid method", func(t *testing.T) {
		yaml := `
config:
  port: 8000
  basePath: /api
  auth:
    defaultProtected: false
routes:
  - method: INVALID
    route: /test
    service: localhost:3000
    path: /test
`
		path := writeTempConfig(t, yaml)
		_, err := LoadConfig(path)
		if err == nil {
			t.Error("Expected validation error for invalid method")
		}
	})

	t.Run("duplicate routes", func(t *testing.T) {
		yaml := `
config:
  port: 8000
  basePath: /api
  auth:
    defaultProtected: false
routes:
  - method: GET
    route: /test
    service: localhost:3000
    path: /test
  - method: GET
    route: /test
    service: localhost:3000
    path: /test
`
		path := writeTempConfig(t, yaml)
		_, err := LoadConfig(path)
		if err == nil {
			t.Error("Expected validation error for duplicate routes")
		}
	})

	t.Run("auth protected without secret or service", func(t *testing.T) {
		yaml := `
config:
  port: 8000
  basePath: /api
  auth:
    defaultProtected: true
routes:
  - method: GET
    route: /test
    service: localhost:3000
    path: /test
`
		path := writeTempConfig(t, yaml)
		_, err := LoadConfig(path)
		if err == nil {
			t.Error("Expected validation error for protected routes without auth config")
		}
	})

	t.Run("mapping missing tag", func(t *testing.T) {
		yaml := `
config:
  port: 8000
  basePath: /api
  auth:
    defaultProtected: false
routes:
  - method: GET
    route: /test
    service: localhost:3000
    path: /test
    mapping:
      - path: /other/{id}
        service: localhost:3000
        tag: ""
`
		path := writeTempConfig(t, yaml)
		_, err := LoadConfig(path)
		if err == nil {
			t.Error("Expected validation error for mapping without tag")
		}
	})
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
