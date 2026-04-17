package auth

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ValidateJWKS validates JWT tokens using a remote JWKS endpoint.
// Supports RS256, RS384, RS512, ES256, ES384, ES512, and other asymmetric algorithms.
func ValidateJWKS(jwksURL string, next http.Handler) http.Handler {
	jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{})
	if err != nil {
		slog.Error("failed to fetch JWKS", "url", jwksURL, "error", err)
		// Return handler that always rejects — don't crash the gateway
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			jsonError(w, "Auth configuration error", http.StatusInternalServerError)
		})
	}

	slog.Info("JWKS loaded", "url", jwksURL)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get(HeaderAuthorization)
		if authHeader == "" {
			jsonError(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || strings.ToLower(bearerToken[0]) != "bearer" {
			jsonError(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(bearerToken[1], jwks.Keyfunc)
		if err != nil || !token.Valid {
			jsonError(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if issuer := r.Header.Get(HeaderJWTIssuer); issuer != "" {
				claimIssuer, ok := claims["iss"].(string)
				if !ok || claimIssuer != issuer {
					jsonError(w, "Invalid issuer", http.StatusUnauthorized)
					return
				}
			}

			if audience := r.Header.Get(HeaderJWTAudience); audience != "" {
				claimAud, ok := claims["aud"].(string)
				if !ok || claimAud != audience {
					jsonError(w, "Invalid audience", http.StatusUnauthorized)
					return
				}
			}

			caser := cases.Title(language.Und)
			for key, value := range claims {
				if str, ok := value.(string); ok {
					headerKey := HeaderPrefix + caser.String(key)
					r.Header.Set(headerKey, str)
				}
			}
		}

		r.Header.Del(HeaderJWTIssuer)
		r.Header.Del(HeaderJWTAudience)

		next.ServeHTTP(w, r)
	})
}
