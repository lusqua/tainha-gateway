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
