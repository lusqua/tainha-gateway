package e2e

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	gatewayURL = envOrDefault("GATEWAY_URL", "http://localhost:8080")
	mockURL    = envOrDefault("MOCK_URL", "http://localhost:4000")
)

const jwtSecret = "e2e-test-secret"

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestMain(m *testing.M) {
	if os.Getenv("E2E") != "true" {
		fmt.Println("Skipping e2e tests (set E2E=true to run)")
		os.Exit(0)
	}

	// Wait for gateway to be ready
	if err := waitForService(gatewayURL+"/api/products", 30*time.Second); err != nil {
		fmt.Printf("Gateway not ready: %v\n", err)
		os.Exit(1)
	}

	// Reset mock request log before tests
	http.Post(mockURL+"/_requests/reset", "", nil)

	os.Exit(m.Run())
}

// --- Public routes ---

func TestPublicRoutes(t *testing.T) {
	t.Run("GET /api/products without auth returns 200", func(t *testing.T) {
		resp, err := http.Get(gatewayURL + "/api/products")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		var products []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&products)

		if len(products) != 3 {
			t.Fatalf("Expected 3 products, got %d", len(products))
		}
	})

	t.Run("GET /api/products/{id} without auth returns 200", func(t *testing.T) {
		resp, err := http.Get(gatewayURL + "/api/products/1")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		var product map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&product)

		if product["name"] != "Laptop" {
			t.Errorf("product name = %v, want Laptop", product["name"])
		}
	})

	t.Run("GET /api/categories without auth returns 200", func(t *testing.T) {
		resp, err := http.Get(gatewayURL + "/api/categories")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
	})
}

// --- Auth protection ---

func TestAuthProtection(t *testing.T) {
	t.Run("GET /api/users without token returns 401", func(t *testing.T) {
		resp, err := http.Get(gatewayURL + "/api/users")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 401 {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}

		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)

		if errResp["success"] != false {
			t.Error("Expected success=false in error response")
		}
	})

	t.Run("GET /api/orders without token returns 401", func(t *testing.T) {
		resp, err := http.Get(gatewayURL + "/api/orders")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 401 {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})

	t.Run("GET /api/users with invalid token returns 401", func(t *testing.T) {
		req, _ := http.NewRequest("GET", gatewayURL+"/api/users", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.value")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 401 {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})

	t.Run("GET /api/users with wrong secret returns 401", func(t *testing.T) {
		token := generateToken("wrong-secret", map[string]interface{}{
			"username": "hacker",
		})

		req, _ := http.NewRequest("GET", gatewayURL+"/api/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 401 {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})

	t.Run("GET /api/users with valid token returns 200", func(t *testing.T) {
		token := generateToken(jwtSecret, map[string]interface{}{
			"username": "alice",
			"role":     "admin",
		})

		req, _ := http.NewRequest("GET", gatewayURL+"/api/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		var users []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&users)

		if len(users) != 2 {
			t.Errorf("Expected 2 users, got %d", len(users))
		}
	})

	t.Run("GET /api/users with expired token returns 401", func(t *testing.T) {
		token := generateToken(jwtSecret, map[string]interface{}{
			"username": "alice",
			"exp":      time.Now().Add(-1 * time.Hour).Unix(),
		})

		req, _ := http.NewRequest("GET", gatewayURL+"/api/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 401 {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})
}

// --- JWT claim forwarding ---

func TestClaimForwarding(t *testing.T) {
	t.Run("JWT claims forwarded as X- headers to backend", func(t *testing.T) {
		// Reset request log
		http.Post(mockURL+"/_requests/reset", "", nil)

		token := generateToken(jwtSecret, map[string]interface{}{
			"username": "alice",
			"role":     "admin",
		})

		req, _ := http.NewRequest("GET", gatewayURL+"/api/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		// Check the mock service's request log to verify headers were forwarded
		logResp, err := http.Get(mockURL + "/_requests")
		if err != nil {
			t.Fatalf("Failed to get request log: %v", err)
		}
		defer logResp.Body.Close()

		var requests []struct {
			Method  string            `json:"method"`
			Path    string            `json:"path"`
			Headers map[string]string `json:"headers"`
		}
		json.NewDecoder(logResp.Body).Decode(&requests)

		// Find the /users request
		found := false
		for _, r := range requests {
			if r.Path == "/users" {
				found = true
				if r.Headers["X-Username"] != "alice" {
					t.Errorf("X-Username = %q, want alice", r.Headers["X-Username"])
				}
				if r.Headers["X-Role"] != "admin" {
					t.Errorf("X-Role = %q, want admin", r.Headers["X-Role"])
				}
				break
			}
		}
		if !found {
			t.Error("No /users request found in mock service log")
		}
	})
}

// --- Response mapping ---

func TestResponseMapping(t *testing.T) {
	t.Run("products have category mapping", func(t *testing.T) {
		resp, err := http.Get(gatewayURL + "/api/products")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		var products []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&products)

		for _, product := range products {
			// categoryId should be removed (removeKeyMapping: true)
			if _, ok := product["categoryId"]; ok {
				t.Errorf("Product %v still has categoryId (should be removed by mapping)", product["id"])
			}

			// category should be present
			cat, ok := product["category"]
			if !ok {
				t.Errorf("Product %v missing 'category' mapping", product["id"])
				continue
			}

			catMap, ok := cat.(map[string]interface{})
			if !ok {
				t.Errorf("Product %v category is not an object: %T", product["id"], cat)
				continue
			}

			if catMap["name"] == nil || catMap["name"] == "" {
				t.Errorf("Product %v category has no name", product["id"])
			}
		}
	})

	t.Run("user has orders mapping", func(t *testing.T) {
		token := generateToken(jwtSecret, map[string]interface{}{
			"username": "alice",
			"role":     "admin",
		})

		req, _ := http.NewRequest("GET", gatewayURL+"/api/users/1", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		var user map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&user)

		orders, ok := user["orders"]
		if !ok {
			t.Fatal("User response missing 'orders' mapping")
		}

		orderList, ok := orders.([]interface{})
		if !ok {
			t.Fatalf("Orders is not an array: %T", orders)
		}

		if len(orderList) != 2 {
			t.Errorf("Expected 2 orders for user 1, got %d", len(orderList))
		}

		// id should still be present (removeKeyMapping: false)
		if _, ok := user["id"]; !ok {
			t.Error("User 'id' should remain when removeKeyMapping is false")
		}
	})
}

// --- SSE ---

func TestSSE(t *testing.T) {
	t.Run("SSE endpoint streams events", func(t *testing.T) {
		resp, err := http.Get(gatewayURL + "/api/events")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		scanner := bufio.NewScanner(resp.Body)
		eventCount := 0
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				eventCount++
				var event map[string]interface{}
				if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
					t.Errorf("Failed to parse SSE event: %v", err)
				}
				if event["message"] != "stock update" {
					t.Errorf("event message = %v, want 'stock update'", event["message"])
				}
			}
		}

		if eventCount < 1 {
			t.Errorf("Expected at least 1 SSE event, got %d", eventCount)
		}
	})
}

// --- Backend not found ---

func TestBackendErrors(t *testing.T) {
	t.Run("non-existent product returns 404", func(t *testing.T) {
		resp, err := http.Get(gatewayURL + "/api/products/999")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 404 {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, want 404, body = %s", resp.StatusCode, body)
		}
	})

	t.Run("non-existent gateway route returns 405 or 404", func(t *testing.T) {
		resp, err := http.Get(gatewayURL + "/api/nonexistent")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 404 && resp.StatusCode != 405 {
			t.Fatalf("status = %d, want 404 or 405", resp.StatusCode)
		}
	})
}

// --- Helpers ---

func generateToken(secret string, claims map[string]interface{}) string {
	if _, ok := claims["exp"]; !ok {
		claims["exp"] = time.Now().Add(time.Hour).Unix()
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func waitForService(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("service at %s not ready after %v", url, timeout)
}
