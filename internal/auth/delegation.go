package auth

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

var delegationClient = &http.Client{
	Timeout: 5 * time.Second,
}

// ValidateWithService delegates token validation to an external auth service.
// The gateway sends the original Authorization header to the auth service.
// If the service responds with 200, the request is forwarded to the backend
// with any JSON response fields added as X- headers.
// Any non-200 response is treated as unauthorized.
func ValidateWithService(serviceURL string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get(HeaderAuthorization)
		if authHeader == "" {
			jsonError(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), "GET", serviceURL, nil)
		if err != nil {
			log.Printf("Failed to create auth delegation request: %v", err)
			jsonError(w, "Internal auth error", http.StatusInternalServerError)
			return
		}
		req.Header.Set(HeaderAuthorization, authHeader)

		resp, err := delegationClient.Do(req)
		if err != nil {
			log.Printf("Auth service unavailable: %v", err)
			jsonError(w, "Auth service unavailable", http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			w.Write(body)
			return
		}

		// Parse response body for claims to forward as headers
		body, err := io.ReadAll(resp.Body)
		if err == nil && len(body) > 0 {
			var claims map[string]interface{}
			if err := json.Unmarshal(body, &claims); err == nil {
				for key, value := range claims {
					if str, ok := value.(string); ok {
						r.Header.Set(HeaderPrefix+key, str)
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}
