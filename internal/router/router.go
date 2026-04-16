package router

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/aguiar-sh/tainha/internal/auth"
	"github.com/aguiar-sh/tainha/internal/config"
	"github.com/aguiar-sh/tainha/internal/mapper"
	"github.com/aguiar-sh/tainha/internal/proxy"
	"github.com/aguiar-sh/tainha/internal/util"
	"github.com/gorilla/mux"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func SetupRouter(cfg *config.Config) (*mux.Router, error) {
	r := mux.NewRouter()
	r.Use(corsMiddleware)

	for _, route := range cfg.Routes {

		path, protocol := util.PathProtocol(route.Service)
		servicePath := fmt.Sprintf("%s://%s", protocol, path)

		reverseProxy, err := proxy.NewReverseProxy(servicePath)
		if err != nil {
			log.Fatalf("Erro ao criar proxy para %s: %v", route.Path, err)
		}

		fullPath := fmt.Sprintf("%s%s", cfg.BaseConfig.BasePath, route.Route)

		handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			log.Println("Request received for:", req.URL.Path)

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
				log.Printf("Error parsing target path: %v", err)
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
				log.Println("Warning: Response body is empty")
			}

			response, err := mapper.Map(route, respBody)
			if err != nil {
				log.Printf("Error mapping response: %v", err)
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
				log.Printf("Error writing response: %v", err)
				return
			} else if n != len(response) {
				log.Printf("Warning: not all bytes were written. Expected %d, wrote %d", len(response), n)
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
