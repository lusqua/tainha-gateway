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
	"github.com/aguiar-sh/tainha/internal/router"
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

	r, err := router.SetupRouter(cfg)
	if err != nil {
		slog.Error("failed to setup router", "error", err)
		os.Exit(1)
	}

	// Log routes
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
		if len(route.Mapping) > 0 {
			tags := make([]string, len(route.Mapping))
			for i, m := range route.Mapping {
				tags[i] = m.Tag
			}
			attrs = append(attrs, "mappings", tags)
		}
		slog.Info("route registered", attrs...)
	}

	addr := fmt.Sprintf(":%d", cfg.BaseConfig.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.BaseConfig.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.BaseConfig.WriteTimeoutSec) * time.Second,
		IdleTimeout:  time.Duration(cfg.BaseConfig.IdleTimeoutSec) * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("gateway started", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("gateway stopped")
}
