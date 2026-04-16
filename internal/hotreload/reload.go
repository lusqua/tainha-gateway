package hotreload

import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/aguiar-sh/tainha/internal/config"
	"github.com/aguiar-sh/tainha/internal/router"
)

// Handler wraps a mux.Router that can be swapped at runtime.
// It implements http.Handler so it can be used as the server's handler.
type Handler struct {
	mu      sync.RWMutex
	handler http.Handler
}

func NewHandler(h http.Handler) *Handler {
	return &Handler{handler: h}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	handler := h.handler
	h.mu.RUnlock()
	handler.ServeHTTP(w, r)
}

func (h *Handler) Swap(newHandler http.Handler) {
	h.mu.Lock()
	h.handler = newHandler
	h.mu.Unlock()
}

// Reload loads a new config from path and swaps the router.
// Returns the new config or an error (old router stays active on error).
func Reload(h *Handler, configPath string) (*config.Config, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		slog.Error("hot reload: config load failed", "error", err)
		return nil, err
	}

	newRouter, err := router.SetupRouter(cfg)
	if err != nil {
		slog.Error("hot reload: router setup failed", "error", err)
		return nil, err
	}

	h.Swap(newRouter)
	slog.Info("hot reload: config reloaded successfully")
	return cfg, nil
}
