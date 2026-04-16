package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type circuitState int

const (
	stateClosed   circuitState = iota // normal operation
	stateOpen                         // failing, reject requests
	stateHalfOpen                     // testing if backend recovered
)

type serviceCircuit struct {
	mu               sync.Mutex
	state            circuitState
	failures         int
	maxFailures      int
	timeout          time.Duration
	halfOpenRequests int
	lastFailure      time.Time
	halfOpenCount    int
}

type CircuitBreaker struct {
	mu               sync.RWMutex
	circuits         map[string]*serviceCircuit
	maxFailures      int
	timeout          time.Duration
	halfOpenRequests int
}

func NewCircuitBreaker(maxFailures, timeoutSec, halfOpenRequests int) *CircuitBreaker {
	return &CircuitBreaker{
		circuits:         make(map[string]*serviceCircuit),
		maxFailures:      maxFailures,
		timeout:          time.Duration(timeoutSec) * time.Second,
		halfOpenRequests: halfOpenRequests,
	}
}

func (cb *CircuitBreaker) getCircuit(service string) *serviceCircuit {
	cb.mu.RLock()
	sc, ok := cb.circuits[service]
	cb.mu.RUnlock()
	if ok {
		return sc
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Double check
	if sc, ok = cb.circuits[service]; ok {
		return sc
	}

	sc = &serviceCircuit{
		maxFailures:      cb.maxFailures,
		timeout:          cb.timeout,
		halfOpenRequests: cb.halfOpenRequests,
	}
	cb.circuits[service] = sc
	return sc
}

// Allow checks if a request to the service should be allowed.
func (cb *CircuitBreaker) Allow(service string) bool {
	sc := cb.getCircuit(service)
	sc.mu.Lock()
	defer sc.mu.Unlock()

	switch sc.state {
	case stateClosed:
		return true
	case stateOpen:
		if time.Since(sc.lastFailure) > sc.timeout {
			sc.state = stateHalfOpen
			sc.halfOpenCount = 0
			slog.Info("circuit breaker half-open", "service", service)
			return true
		}
		return false
	case stateHalfOpen:
		if sc.halfOpenCount < sc.halfOpenRequests {
			sc.halfOpenCount++
			return true
		}
		return false
	}
	return true
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess(service string) {
	sc := cb.getCircuit(service)
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.state == stateHalfOpen {
		sc.state = stateClosed
		sc.failures = 0
		slog.Info("circuit breaker closed", "service", service)
	}
	sc.failures = 0
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure(service string) {
	sc := cb.getCircuit(service)
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.failures++
	sc.lastFailure = time.Now()

	if sc.state == stateHalfOpen {
		sc.state = stateOpen
		slog.Warn("circuit breaker re-opened", "service", service, "failures", sc.failures)
		return
	}

	if sc.failures >= sc.maxFailures {
		sc.state = stateOpen
		slog.Warn("circuit breaker opened", "service", service, "failures", sc.failures)
	}
}

// Middleware wraps an HTTP handler with circuit breaker logic.
// It uses the backend host from X-Backend-Service header (set by the router).
func (cb *CircuitBreaker) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		service := r.Header.Get("X-Backend-Service")
		if service == "" {
			next.ServeHTTP(w, r)
			return
		}

		if !cb.Allow(service) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "service unavailable (circuit breaker open)",
				"success": false,
			})
			return
		}

		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)

		if rec.statusCode >= 500 {
			cb.RecordFailure(service)
		} else {
			cb.RecordSuccess(service)
		}
	})
}
