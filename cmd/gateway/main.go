package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aguiar-sh/tainha/internal/config"
	"github.com/aguiar-sh/tainha/internal/hotreload"
	"github.com/aguiar-sh/tainha/internal/mapper"
	"github.com/aguiar-sh/tainha/internal/router"
	"github.com/aguiar-sh/tainha/internal/telemetry"
)

func main() {
	configPath := flag.String("config", "./config/config.yaml", "Path to configuration file")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize OpenTelemetry
	if cfg.BaseConfig.Telemetry.Enabled {
		ctx := context.Background()
		shutdown, err := telemetry.Setup(ctx, cfg.BaseConfig.Telemetry)
		if err != nil {
			slog.Error("failed to initialize telemetry", "error", err)
			os.Exit(1)
		}
		defer shutdown(ctx)
	}

	// Initialize mapping cache
	if cfg.BaseConfig.MappingCache.Enabled {
		ttl := time.Duration(cfg.BaseConfig.MappingCache.TTLSec) * time.Second
		cache := mapper.NewCache(ttl, cfg.BaseConfig.MappingCache.MaxSize)
		mapper.SetCache(cache)
		slog.Info("mapping cache enabled",
			"ttl", cfg.BaseConfig.MappingCache.TTLSec,
			"maxSize", cfg.BaseConfig.MappingCache.MaxSize,
		)
	}

	r, err := router.SetupRouter(cfg)
	if err != nil {
		slog.Error("failed to setup router", "error", err)
		os.Exit(1)
	}

	logRoutes(cfg)

	// Wrap router in hot-reloadable handler
	handler := hotreload.NewHandler(r)

	addr := fmt.Sprintf(":%d", cfg.BaseConfig.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  time.Duration(cfg.BaseConfig.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.BaseConfig.WriteTimeoutSec) * time.Second,
		IdleTimeout:  time.Duration(cfg.BaseConfig.IdleTimeoutSec) * time.Second,
	}

	// Signal handling
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		slog.Info("gateway started", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	for {
		sig := <-done
		if sig == syscall.SIGHUP {
			slog.Info("received SIGHUP, reloading config...")
			newCfg, err := hotreload.Reload(handler, *configPath)
			if err != nil {
				slog.Error("reload failed, keeping current config", "error", err)
				continue
			}
			// Update cache if config changed
			if newCfg.BaseConfig.MappingCache.Enabled {
				ttl := time.Duration(newCfg.BaseConfig.MappingCache.TTLSec) * time.Second
				mapper.SetCache(mapper.NewCache(ttl, newCfg.BaseConfig.MappingCache.MaxSize))
			} else {
				mapper.SetCache(nil)
			}
			logRoutes(newCfg)
			continue
		}

		// SIGTERM / SIGINT — graceful shutdown
		slog.Info("shutting down gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("forced shutdown", "error", err)
			os.Exit(1)
		}
		slog.Info("gateway stopped")
		return
	}
}

func logRoutes(cfg *config.Config) {
	for _, route := range cfg.Routes {
		fullPath := fmt.Sprintf("%s%s", cfg.BaseConfig.BasePath, route.Route)
		attrs := []any{
			"method", route.Method,
			"path", fullPath,
			"service", route.Service + route.Path,
			"public", route.Public,
		}
		if route.IsSSE {
			attrs = append(attrs, "sse", true)
		}
		if route.IsWebSocket {
			attrs = append(attrs, "websocket", true)
		}
		if len(route.Mapping) > 0 {
			tags := make([]string, len(route.Mapping))
			for i, m := range route.Mapping {
				tags[i] = m.Tag
			}
			attrs = append(attrs, "mappings", tags)
		}
		slog.Info("route registered", attrs...)
	}
}
