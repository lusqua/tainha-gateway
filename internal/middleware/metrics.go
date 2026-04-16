package middleware

import (
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter          = otel.Meter("tainha-gateway")
	requestCount   metric.Int64Counter
	requestLatency metric.Float64Histogram
	activeRequests metric.Int64UpDownCounter
	rateLimitHits  metric.Int64Counter
)

func init() {
	var err error

	requestCount, err = meter.Int64Counter("http.server.request.count",
		metric.WithDescription("Total HTTP requests"),
	)
	if err != nil {
		panic(err)
	}

	requestLatency, err = meter.Float64Histogram("http.server.request.duration",
		metric.WithDescription("HTTP request latency in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}

	activeRequests, err = meter.Int64UpDownCounter("http.server.active_requests",
		metric.WithDescription("Number of in-flight requests"),
	)
	if err != nil {
		panic(err)
	}

	rateLimitHits, err = meter.Int64Counter("http.server.rate_limit.hits",
		metric.WithDescription("Number of rate-limited requests"),
	)
	if err != nil {
		panic(err)
	}
}

// RecordRateLimitHit records a rate limit event for metrics.
func RecordRateLimitHit(ip string) {
	rateLimitHits.Add(nil, 1, metric.WithAttributes(
		attribute.String("client.ip", ip),
	))
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Metrics middleware records request count, latency, and active connections.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		attrs := metric.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", r.URL.Path),
		)

		activeRequests.Add(r.Context(), 1, attrs)
		defer activeRequests.Add(r.Context(), -1, attrs)

		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)

		duration := time.Since(start).Seconds()
		statusAttrs := metric.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", r.URL.Path),
			attribute.String("http.status_code", strconv.Itoa(rec.statusCode)),
		)

		requestCount.Add(r.Context(), 1, statusAttrs)
		requestLatency.Record(r.Context(), duration, statusAttrs)
	})
}
