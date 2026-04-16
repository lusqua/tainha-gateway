package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	HeaderAuthorization = "Authorization"
	HeaderJWTSecret     = "X-JWT-Secret"
	HeaderJWTIssuer     = "X-JWT-Issuer"
	HeaderJWTAudience   = "X-JWT-Audience"
	HeaderPrefix        = "X-"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Success bool   `json:"success"`
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(ErrorResponse{Error: message, Success: false}); err != nil {
		log.Printf("Failed to encode JSON error response: %v", err)
	}
	log.Println("Invalid JWT token", message)
}

func ValidateJWT(secret string, next http.Handler) http.Handler {
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

		token, err := jwt.Parse(bearerToken[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

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

		r.Header.Del(HeaderJWTSecret)
		r.Header.Del(HeaderJWTIssuer)
		r.Header.Del(HeaderJWTAudience)

		next.ServeHTTP(w, r)
	})
}
