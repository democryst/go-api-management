package handler

import (
    "context"
    "crypto/rand"
    "fmt"
    "log/slog"
    "net/http"
    "net/url"

    "github.com/auth0/go-jwt-middleware/v3"
    "github.com/auth0/go-jwt-middleware/v3/validator"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

// GetRequestID extracts the tracing request ID from context.
// Time Complexity: O(1)
// Space Complexity: O(1)
func GetRequestID(ctx context.Context) string {
    if id, ok := ctx.Value(RequestIDKey).(string); ok {
        return id
    }
    return ""
}

// ContextHandler wraps slog.Handler to inject request_id context attributes automatically.
type ContextHandler struct {
    slog.Handler
}

// NewContextHandler creates a ContextHandler wrapper.
func NewContextHandler(h slog.Handler) *ContextHandler {
    return &ContextHandler{Handler: h}
}

// Handle extracts context variables and appends them as structured attributes.
// Time Complexity: O(1)
// Space Complexity: O(1)
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
    if id := GetRequestID(ctx); id != "" {
        r.AddAttrs(slog.String("request_id", id))
    }
    return h.Handler.Handle(ctx, r)
}

// CorrelationMiddleware injects request correlation identifiers into all HTTP contexts.
// Time Complexity: O(1)
// Space Complexity: O(1)
func CorrelationMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = generateUUIDv4()
        }

        ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
        w.Header().Set("X-Request-ID", requestID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// JWTMiddleware returns the middleware verifying Auth0 JWT signatures locally via cached JWKS.
// Time Complexity: O(1) (local cryptographic signature checks)
// Space Complexity: O(1)
func JWTMiddleware(auth0Domain, apiAudience string, provider func(context.Context) (any, error)) func(http.Handler) http.Handler {
    issuerURL, _ := url.Parse("https://" + auth0Domain + "/")

    jwtValidator, err := validator.New(
        validator.WithKeyFunc(provider),
        validator.WithAlgorithm(validator.RS256), // Mitigates alg:none attacks
        validator.WithIssuer(issuerURL.String()),
        validator.WithAudience(apiAudience),
    )
    if err != nil {
        panic(fmt.Sprintf("failed to construct JWT validator: %v", err))
    }

    middleware, err := jwtmiddleware.New(jwtmiddleware.WithValidator(jwtValidator))
    if err != nil {
        panic(fmt.Sprintf("failed to construct JWT middleware: %v", err))
    }

    return middleware.CheckJWT
}

func generateUUIDv4() string {
    uuid := make([]byte, 16)
    _, _ = rand.Read(uuid)
    uuid[6] = (uuid[6] & 0x0f) | 0x40
    uuid[8] = (uuid[8] & 0x3f) | 0x80
    return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}
