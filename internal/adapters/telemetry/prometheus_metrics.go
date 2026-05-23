package telemetry

import (
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTPRequestDuration tracks response latency percentiles.
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of response latencies for HTTP requests.",
			Buckets: []float64{.001, .002, .005, .01, .02, .05, .1, .2, .5, 1, 2, 5},
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestsTotal tracks request throughput.
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total count of processed HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)

	// CircuitBreakerState tracks active gobreaker status (0 = Closed, 1 = HalfOpen, 2 = Open).
	CircuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Current state of circuit breakers (0 = Closed, 1 = HalfOpen, 2 = Open).",
		},
		[]string{"name"},
	)

	// JWKSCacheHits tracks cache hits for signature verification.
	JWKSCacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "jwks_cache_hits_total",
			Help: "Total number of JWKS caching hits.",
		},
	)
)

func init() {
	prometheus.MustRegister(HTTPRequestDuration)
	prometheus.MustRegister(HTTPRequestsTotal)
	prometheus.MustRegister(CircuitBreakerState)
	prometheus.MustRegister(JWKSCacheHits)
}

// MetricsHandler returns the standard HTTP controller for exposing Prometheus metrics.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// MetricsMiddleware tracks throughput and latency metrics for all HTTP router requests.
// Time Complexity: O(1)
// Space Complexity: O(1)
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)

		duration := time.Since(start).Seconds()
		path := r.URL.Path

		// Merge variable parameters to prevent label cardinality explosion in Prometheus
		if strings.HasPrefix(path, "/private/transfers/") {
			path = "/private/transfers/:id"
		}

		statusStr := http.StatusText(sw.status)
		HTTPRequestDuration.WithLabelValues(r.Method, path, statusStr).Observe(duration)
		HTTPRequestsTotal.WithLabelValues(r.Method, path, statusStr).Inc()
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
