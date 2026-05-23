package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	jwtcore "github.com/auth0/go-jwt-middleware/v3/core"
	"github.com/auth0/go-jwt-middleware/v3/validator"
)

type MockRateLimiterBackend struct {
	mu           sync.Mutex
	calls        []string
	evalCallback func(key string, capacity, refillRate, refillPeriodMs, nowMs int64) (bool, int64, error)
}

func (m *MockRateLimiterBackend) EvalTokenBucket(ctx context.Context, key string, capacity, refillRate, refillPeriodMs, nowMs int64) (bool, int64, error) {
	m.mu.Lock()
	m.calls = append(m.calls, key)
	m.mu.Unlock()

	if m.evalCallback != nil {
		return m.evalCallback(key, capacity, refillRate, refillPeriodMs, nowMs)
	}
	return true, capacity - 1, nil
}

func TestRateLimiterMiddleware_Allowed(t *testing.T) {
	backend := &MockRateLimiterBackend{
		evalCallback: func(key string, capacity, refillRate, refillPeriodMs, nowMs int64) (bool, int64, error) {
			return true, 4, nil
		},
	}

	logger := slog.New(slog.NewJSONHandler(&strings.Builder{}, nil))
	config := RateLimiterConfig{
		Capacity:     5,
		RefillRate:   1,
		RefillPeriod: 1 * time.Second,
	}

	middleware := NewRateLimiterMiddleware(backend, logger, config)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Handler(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", w.Code)
	}

	if limit := w.Header().Get("X-RateLimit-Limit"); limit != "5" {
		t.Errorf("expected X-RateLimit-Limit 5, got %s", limit)
	}

	if remaining := w.Header().Get("X-RateLimit-Remaining"); remaining != "4" {
		t.Errorf("expected X-RateLimit-Remaining 4, got %s", remaining)
	}

	backend.mu.Lock()
	defer backend.mu.Unlock()
	if len(backend.calls) != 1 {
		t.Errorf("expected 1 backend call, got %d", len(backend.calls))
	}
}

func TestRateLimiterMiddleware_Throttled(t *testing.T) {
	backend := &MockRateLimiterBackend{
		evalCallback: func(key string, capacity, refillRate, refillPeriodMs, nowMs int64) (bool, int64, error) {
			return false, 0, nil
		},
	}

	logger := slog.New(slog.NewJSONHandler(&strings.Builder{}, nil))
	config := RateLimiterConfig{
		Capacity:     5,
		RefillRate:   1,
		RefillPeriod: 2 * time.Second,
	}

	middleware := NewRateLimiterMiddleware(backend, logger, config)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("downstream handler should not be called when throttled")
	})

	handler := middleware.Handler(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", w.Code)
	}

	if retryAfter := w.Header().Get("Retry-After"); retryAfter != "2" {
		t.Errorf("expected Retry-After to be 2, got %s", retryAfter)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if resp["error"] != "too many requests" {
		t.Errorf("unexpected error msg: %s", resp["error"])
	}
}

func TestRateLimiterMiddleware_FailOpen(t *testing.T) {
	backend := &MockRateLimiterBackend{
		evalCallback: func(key string, capacity, refillRate, refillPeriodMs, nowMs int64) (bool, int64, error) {
			return false, 0, errors.New("valkey cluster timeout")
		},
	}

	logger := slog.New(slog.NewJSONHandler(&strings.Builder{}, nil))
	config := RateLimiterConfig{
		Capacity:     5,
		RefillRate:   1,
		RefillPeriod: 1 * time.Second,
	}

	middleware := NewRateLimiterMiddleware(backend, logger, config)

	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Handler(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 OK under fail-open strategy, got %d", w.Code)
	}

	if !nextCalled {
		t.Error("expected downstream handler to be called when rate limiter fails open")
	}
}

func TestRateLimiterMiddleware_ClientIdentification(t *testing.T) {
	backend := &MockRateLimiterBackend{}
	logger := slog.New(slog.NewJSONHandler(&strings.Builder{}, nil))
	config := RateLimiterConfig{Capacity: 5, RefillRate: 1, RefillPeriod: 1 * time.Second}
	middleware := NewRateLimiterMiddleware(backend, logger, config)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := middleware.Handler(nextHandler)

	// Case 1: Unauthenticated request (uses RemoteAddr IP)
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "1.2.3.4:5555"
	handler.ServeHTTP(httptest.NewRecorder(), req1)

	// Case 2: Unauthenticated request with X-Forwarded-For header
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-Forwarded-For", "5.6.7.8, 10.0.0.1")
	handler.ServeHTTP(httptest.NewRecorder(), req2)

	// Case 3: Authenticated request (uses OIDC Subject)
	req3 := httptest.NewRequest("GET", "/test", nil)
	claims := &validator.ValidatedClaims{
		RegisteredClaims: validator.RegisteredClaims{
			Subject: "auth0|user-999",
		},
	}
	ctx := jwtcore.SetClaims(req3.Context(), claims)
	req3 = req3.WithContext(ctx)
	handler.ServeHTTP(httptest.NewRecorder(), req3)

	backend.mu.Lock()
	defer backend.mu.Unlock()

	if len(backend.calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(backend.calls))
	}

	// Verify hashed key prefixes to protect PII
	// Check that calls map correctly. Note that the key contains "ratelimit:" + hashed ID.
	expectedHash1 := middleware.hashIdentifier("ip:1.2.3.4")
	if backend.calls[0] != "ratelimit:"+expectedHash1 {
		t.Errorf("expected key ratelimit:%s, got %s", expectedHash1, backend.calls[0])
	}

	expectedHash2 := middleware.hashIdentifier("ip:5.6.7.8")
	if backend.calls[1] != "ratelimit:"+expectedHash2 {
		t.Errorf("expected key ratelimit:%s, got %s", expectedHash2, backend.calls[1])
	}

	expectedHash3 := middleware.hashIdentifier("user:auth0|user-999")
	if backend.calls[2] != "ratelimit:"+expectedHash3 {
		t.Errorf("expected key ratelimit:%s, got %s", expectedHash3, backend.calls[2])
	}
}
