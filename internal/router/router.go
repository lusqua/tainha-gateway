package router

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/aguiar-sh/tainha/internal/auth"
	"github.com/aguiar-sh/tainha/internal/config"
	"github.com/aguiar-sh/tainha/internal/mapper"
	"github.com/aguiar-sh/tainha/internal/middleware"
	"github.com/aguiar-sh/tainha/internal/proxy"
	"github.com/aguiar-sh/tainha/internal/util"
	"github.com/gorilla/mux"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-CSRF-Token, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type, X-Request-ID")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func SetupRouter(cfg *config.Config) (*mux.Router, error) {
	r := mux.NewRouter()

	// Health check (before any middleware)
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods("GET")

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(corsMiddleware)

	if cfg.BaseConfig.RateLimit.Enabled {
		limiter := middleware.NewRateLimiter(
			cfg.BaseConfig.RateLimit.RequestsPerSec,
			cfg.BaseConfig.RateLimit.Burst,
		)
		r.Use(limiter.Middleware)
		slog.Info("rate limiting enabled",
			"requestsPerSec", cfg.BaseConfig.RateLimit.RequestsPerSec,
			"burst", cfg.BaseConfig.RateLimit.Burst,
		)
	}

	for _, route := range cfg.Routes {

		path, protocol := util.PathProtocol(route.Service)
		servicePath := fmt.Sprintf("%s://%s", protocol, path)

		reverseProxy, err := proxy.NewReverseProxy(servicePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy for %s: %w", route.Path, err)
		}

		fullPath := fmt.Sprintf("%s%s", cfg.BaseConfig.BasePath, route.Route)

		handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			reqID := req.Header.Get(middleware.HeaderRequestID)
			slog.Info("request received",
				"path", req.URL.Path,
				"method", req.Method,
				"requestId", reqID,
			)

			// Extract path parameters using the utility function
			params := util.ExtractPathParams(route.Path)
			vars := mux.Vars(req)

			// Construct the target path dynamically
			targetPath := route.Path
			for _, param := range params {
				value, ok := vars[param]
				if !ok {
					http.Error(w, fmt.Sprintf("Parameter %s not found in request path", param), http.StatusBadRequest)
					return
				}
				// Replace the placeholder with the actual value
				targetPath = strings.Replace(targetPath, fmt.Sprintf("{%s}", param), value, -1)
			}

			// Parse the target path to separate path and query
			parsedURL, err := url.Parse(targetPath)
			if err != nil {
				slog.Error("error parsing target path", "error", err, "requestId", reqID)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			proxyReq := req.Clone(req.Context())
			proxyReq.URL.Path = parsedURL.Path
			proxyReq.URL.RawQuery = parsedURL.RawQuery
			proxyReq.URL.Host = path
			proxyReq.URL.Scheme = protocol
			proxyReq.Host = path

			// For routes without mapping, use direct proxy
			if route.IsSSE {
				reverseProxy.ServeHTTP(w, proxyReq)
				return
			}

			// Only use recorder if we need to map the response
			rec := httptest.NewRecorder()
			reverseProxy.ServeHTTP(rec, proxyReq)

			// Read the response body
			respBody := rec.Body.Bytes()

			if rec.Code < 200 || rec.Code >= 300 {
				for k, v := range rec.Header() {
					w.Header()[k] = v
				}
				w.WriteHeader(rec.Code)
				w.Write(respBody)
				return
			}

			// Check if the response body is empty
			if len(respBody) == 0 {
				slog.Warn("response body is empty", "path", req.URL.Path, "requestId", reqID)
			}

			response, err := mapper.Map(route, respBody)
			if err != nil {
				slog.Error("error mapping response", "error", err, "requestId", reqID)
				http.Error(w, "Failed to map response", http.StatusInternalServerError)
				return
			}

			// Copy headers from the recorder to the response writer
			for k, v := range rec.Header() {
				w.Header()[k] = v
			}

			// Set the correct Content-Length
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(response)))

			// Write the status code
			w.WriteHeader(rec.Code)

			// Write the final response
			n, err := w.Write(response)
			if err != nil {
				slog.Error("error writing response", "error", err, "requestId", reqID)
				return
			} else if n != len(response) {
				slog.Warn("not all bytes written", "expected", len(response), "wrote", n, "requestId", reqID)
			}
		})

		if route.IsSSE {
			originalHandler := handler
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

				// Handle SSE stream
				originalHandler.ServeHTTP(w, r)
			})
		}

		if cfg.BaseConfig.Auth.DefaultProtected && !route.Public {
			if cfg.BaseConfig.Auth.AuthService != "" {
				authPath := cfg.BaseConfig.Auth.AuthPath
				if authPath == "" {
					authPath = "/validate"
				}
				_, authProtocol := util.PathProtocol(cfg.BaseConfig.Auth.AuthService)
				authHost, _ := util.PathProtocol(cfg.BaseConfig.Auth.AuthService)
				authURL := fmt.Sprintf("%s://%s%s", authProtocol, authHost, authPath)
				handler = auth.ValidateWithService(authURL, handler).ServeHTTP
			} else {
				handler = auth.ValidateJWT(cfg.BaseConfig.Auth.Secret, handler).ServeHTTP
			}
		}

		r.Handle(fullPath, handler).Methods(route.Method, "OPTIONS")
	}

	return r, nil
}
