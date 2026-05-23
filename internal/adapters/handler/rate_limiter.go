package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v3"
	"github.com/auth0/go-jwt-middleware/v3/validator"
	"github.com/valkey-io/valkey-go"
)

// RateLimiterBackend abstracts the underlying rate limiter client operations.
type RateLimiterBackend interface {
	EvalTokenBucket(ctx context.Context, key string, capacity, refillRate, refillPeriodMs, nowMs int64) (allowed bool, remaining int64, err error)
}

// ValkeyRateLimiterBackend implements RateLimiterBackend using the official valkey-go client.
type ValkeyRateLimiterBackend struct {
	client valkey.Client
}

// NewValkeyRateLimiterBackend constructs a ValkeyRateLimiterBackend.
func NewValkeyRateLimiterBackend(client valkey.Client) *ValkeyRateLimiterBackend {
	return &ValkeyRateLimiterBackend{client: client}
}

const tokenBucketLuaScript = `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local refill_period = tonumber(ARGV[3])
local now = tonumber(ARGV[4])

-- Using Valkey's native server.call namespace instead of redis.call
local data = server.call('HMGET', key, 'tokens', 'last_update')
local tokens = tonumber(data[1])
local last_update = tonumber(data[2])

if tokens == nil then
    tokens = capacity
    last_update = now
else
    local elapsed = now - last_update
    if elapsed > 0 then
        local refilled = elapsed * (refill_rate / refill_period)
        if refilled > 0 then
            tokens = math.min(capacity, tokens + refilled)
            last_update = now
        end
    end
end

if tokens >= 1 then
    tokens = tokens - 1
    server.call('HMSET', key, 'tokens', tokens, 'last_update', last_update)
    local ttl = math.ceil((capacity / refill_rate) * (refill_period / 1000) * 2)
    server.call('EXPIRE', key, ttl)
    return {1, math.floor(tokens * 1000)}
else
    return {0, math.floor(tokens * 1000)}
end
`

// EvalTokenBucket executes the atomic Lua script on Valkey to evaluate the request.
func (b *ValkeyRateLimiterBackend) EvalTokenBucket(ctx context.Context, key string, capacity, refillRate, refillPeriodMs, nowMs int64) (bool, int64, error) {
	cmd := b.client.B().Eval().
		Script(tokenBucketLuaScript).
		Numkeys(1).
		Key(key).
		Arg(
			strconv.FormatInt(capacity, 10),
			strconv.FormatInt(refillRate, 10),
			strconv.FormatInt(refillPeriodMs, 10),
			strconv.FormatInt(nowMs, 10),
		).
		Build()

	res := b.client.Do(ctx, cmd)
	if err := res.Error(); err != nil {
		return false, 0, err
	}

	vals, err := res.ToArray()
	if err != nil {
		return false, 0, err
	}

	if len(vals) < 2 {
		return false, 0, valkey.Nil
	}

	allowed, err1 := vals[0].ToInt64()
	remaining, err2 := vals[1].ToInt64()
	if err1 != nil {
		return false, 0, err1
	}
	if err2 != nil {
		return false, 0, err2
	}

	return allowed == 1, remaining, nil
}

// RateLimiterConfig defines token bucket boundaries.
type RateLimiterConfig struct {
	Capacity     int64         // Maximum tokens the bucket can hold
	RefillRate   int64         // Tokens added per refill period
	RefillPeriod time.Duration // Time interval for refilling tokens
}

// RateLimiterMiddleware enforces distributed token bucket rate limiting.
// Time Complexity: O(1)
// Space Complexity: O(1)
type RateLimiterMiddleware struct {
	backend RateLimiterBackend
	logger  *slog.Logger
	config  RateLimiterConfig
}

// NewRateLimiterMiddleware constructs a RateLimiterMiddleware.
func NewRateLimiterMiddleware(backend RateLimiterBackend, logger *slog.Logger, config RateLimiterConfig) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		backend: backend,
		logger:  logger,
		config:  config,
	}
}

// Handler returns the rate limiting middleware handler.
func (m *RateLimiterMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		rawId := m.getClientIdentifier(r)
		hashedId := m.hashIdentifier(rawId)
		key := "ratelimit:" + hashedId

		allowed, remaining, err := m.backend.EvalTokenBucket(
			ctx,
			key,
			m.config.Capacity,
			m.config.RefillRate,
			m.config.RefillPeriod.Milliseconds(),
			time.Now().UnixMilli(),
		)

		if err != nil {
			// Fail-open strategy: log warning and proceed to avoid total outage
			m.logger.WarnContext(ctx, "rate limiter backend error, failing open", slog.Any("error", err))
			next.ServeHTTP(w, r)
			return
		}

		// Set rate limit telemetry headers
		w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(m.config.Capacity, 10))
		w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

		if !allowed {
			// 429 Too Many Requests
			w.Header().Set("Retry-After", strconv.FormatInt(int64(m.config.RefillPeriod.Seconds()), 10))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":   "too many requests",
				"message": "rate limit exceeded, please retry later",
			})
			m.logger.WarnContext(ctx, "rate limit exceeded for client", slog.String("client_hash", hashedId))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *RateLimiterMiddleware) getClientIdentifier(r *http.Request) string {
	// 1. Try OIDC Subject from JWT claims (authenticated user context)
	claims, err := jwtmiddleware.GetClaims[*validator.ValidatedClaims](r.Context())
	if err == nil && claims != nil && claims.RegisteredClaims.Subject != "" {
		return "user:" + claims.RegisteredClaims.Subject
	}

	// 2. Fall back to client IP (unauthenticated network context)
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip = strings.TrimSpace(ips[0])
		}
	}

	return "ip:" + ip
}

func (m *RateLimiterMiddleware) hashIdentifier(id string) string {
	hash := sha256.Sum256([]byte(id))
	return hex.EncodeToString(hash[:])
}
