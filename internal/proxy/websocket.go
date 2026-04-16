package proxy

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

// WebSocketProxy handles WebSocket upgrade requests by establishing a TCP tunnel
// to the backend service. No external WebSocket library needed — we hijack the
// connection and pipe bytes bidirectionally.
func WebSocketProxy(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isWebSocketUpgrade(r) {
			http.Error(w, "expected websocket upgrade", http.StatusBadRequest)
			return
		}

		// Connect to backend
		backendConn, err := net.DialTimeout("tcp", target, 10*time.Second)
		if err != nil {
			slog.Error("websocket: failed to connect to backend", "target", target, "error", err)
			http.Error(w, "backend unavailable", http.StatusBadGateway)
			return
		}
		defer backendConn.Close()

		// Hijack the client connection
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijacking not supported", http.StatusInternalServerError)
			return
		}
		clientConn, clientBuf, err := hijacker.Hijack()
		if err != nil {
			slog.Error("websocket: hijack failed", "error", err)
			return
		}
		defer clientConn.Close()

		// Forward the original upgrade request to backend
		if err := r.Write(backendConn); err != nil {
			slog.Error("websocket: failed to forward upgrade request", "error", err)
			return
		}

		// Bidirectional copy
		done := make(chan struct{}, 2)

		go func() {
			io.Copy(backendConn, clientBuf)
			done <- struct{}{}
		}()
		go func() {
			io.Copy(clientConn, backendConn)
			done <- struct{}{}
		}()

		// Wait for either direction to close
		<-done
	}
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Connection"), "upgrade") &&
		strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}
