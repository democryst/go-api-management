package handler

import (
    "context"
    "encoding/json"
    "errors"
    "log/slog"
    "net/http"
    "time"

    "github.com/democryst/go-api-management/internal/adapters/telemetry"
    "github.com/democryst/go-api-management/internal/core/domain"
    "github.com/democryst/go-api-management/internal/core/services"
    "github.com/go-chi/chi/v5"
)

const (
    SessionCookieName      = "session_id"
    RefreshTokenCookieName = "refresh_token"
)

// AuthHandler wraps HTTP handlers and configures routing.
type AuthHandler struct {
    authService *services.AuthService
    logger      *slog.Logger
    auth0Domain string
    apiAudience string
    jwtProvider func(context.Context) (any, error)
}

// NewAuthHandler constructs a new AuthHandler.
func NewAuthHandler(
    authService *services.AuthService,
    logger *slog.Logger,
    auth0Domain string,
    apiAudience string,
    jwtProvider func(context.Context) (any, error),
) *AuthHandler {
    return &AuthHandler{
        authService: authService,
        logger:      logger,
        auth0Domain: auth0Domain,
        apiAudience: apiAudience,
        jwtProvider: jwtProvider,
    }
}

// Routes sets up go-chi subrouting and maps endpoints.
// Time Complexity: O(1)
// Space Complexity: O(1)
func (h *AuthHandler) Routes() chi.Router {
    r := chi.NewRouter()

    r.Use(CorrelationMiddleware)
    r.Use(telemetry.MetricsMiddleware)

    // Public REST endpoints
    r.Get("/login", h.Login)
    r.Get("/callback", h.Callback)
    r.Post("/refresh", h.Refresh)
    r.Get("/health", h.Health)
    r.Handle("/metrics", telemetry.MetricsHandler())

    // Authenticated REST endpoints (JWT verified)
    r.Group(func(r chi.Router) {
        r.Use(JWTMiddleware(h.auth0Domain, h.apiAudience, h.jwtProvider))
        r.Post("/logout", h.Logout)
        r.Get("/userinfo", h.UserInfo)
    })

    return r
}

// Login initiates the PKCE OIDC redirection.
// Time Complexity: O(1)
// Space Complexity: O(1)
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    redirectURI := r.URL.Query().Get("redirect_uri")
    if redirectURI == "" {
        h.writeError(r.Context(), w, http.StatusBadRequest, "Missing redirect_uri parameter")
        return
    }

    url, sessionID, err := h.authService.InitiateAuth(r.Context(), redirectURI)
    if err != nil {
        h.logger.ErrorContext(r.Context(), "failed to initiate authorization", slog.Any("error", err))
        h.writeError(r.Context(), w, http.StatusInternalServerError, "Internal Server Error")
        return
    }

    // Set ephemeral browser state cookie
    http.SetCookie(w, &http.Cookie{
        Name:     SessionCookieName,
        Value:    sessionID,
        Path:     "/",
        Expires:  time.Now().Add(10 * time.Minute),
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
    })

    w.Header().Set("Location", url)
    w.WriteHeader(http.StatusFound)
}

// Callback processes the OIDC redirect verification and code-token exchange.
// Time Complexity: O(1) (network bound)
// Space Complexity: O(1)
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
    cookie, err := r.Cookie(SessionCookieName)
    if err != nil {
        h.writeError(r.Context(), w, http.StatusBadRequest, "Missing state session cookie")
        return
    }

    code := r.URL.Query().Get("code")
    state := r.URL.Query().Get("state")
    if code == "" || state == "" {
        h.writeError(r.Context(), w, http.StatusBadRequest, "Missing authorization parameters")
        return
    }

    user, tokens, err := h.authService.HandleCallback(r.Context(), cookie.Value, code, state)
    if err != nil {
        h.logger.ErrorContext(r.Context(), "callback exchange failed", slog.Any("error", err))
        if errors.Is(err, domain.ErrSessionExpired) || errors.Is(err, domain.ErrInvalidState) {
            h.writeError(r.Context(), w, http.StatusUnauthorized, err.Error())
            return
        }
        h.writeError(r.Context(), w, http.StatusInternalServerError, "Token exchange failed")
        return
    }

    // Single-use: Clear ephemeral session cookie
    http.SetCookie(w, &http.Cookie{
        Name:     SessionCookieName,
        Value:    "",
        Path:     "/",
        MaxAge:   -1,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
    })

    // Set Refresh Token in Secure HTTP-Only Cookie
    if tokens.RefreshToken != "" {
        http.SetCookie(w, &http.Cookie{
            Name:     RefreshTokenCookieName,
            Value:    tokens.RefreshToken,
            Path:     "/",
            Expires:  time.Now().Add(30 * 24 * time.Hour), // 30 Days TTL
            HttpOnly: true,
            Secure:   true,
            SameSite: http.SameSiteStrictMode,
        })
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]interface{}{
        "user":         user,
        "access_token": tokens.AccessToken,
        "expires_in":   tokens.ExpiresIn,
    })
}

// Refresh handles single-use refresh token rotation.
// Time Complexity: O(1) (network bound)
// Space Complexity: O(1)
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
    cookie, err := r.Cookie(RefreshTokenCookieName)
    if err != nil {
        h.writeError(r.Context(), w, http.StatusBadRequest, "Missing refresh token cookie")
        return
    }

    tokens, err := h.authService.RefreshToken(r.Context(), cookie.Value)
    if err != nil {
        h.logger.ErrorContext(r.Context(), "refresh rotation failed", slog.Any("error", err))
        h.writeError(r.Context(), w, http.StatusUnauthorized, "Invalid refresh token")
        return
    }

    // Set new rotated Refresh Token in secure cookie
    if tokens.RefreshToken != "" {
        http.SetCookie(w, &http.Cookie{
            Name:     RefreshTokenCookieName,
            Value:    tokens.RefreshToken,
            Path:     "/",
            Expires:  time.Now().Add(30 * 24 * time.Hour),
            HttpOnly: true,
            Secure:   true,
            SameSite: http.SameSiteStrictMode,
        })
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]interface{}{
        "access_token": tokens.AccessToken,
        "expires_in":   tokens.ExpiresIn,
    })
}

// Logout revokes active tokens and clears cookies.
// Time Complexity: O(1) (network bound)
// Space Complexity: O(1)
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
    cookie, err := r.Cookie(RefreshTokenCookieName)
    if err == nil {
        _ = h.authService.RevokeToken(r.Context(), cookie.Value)
    }

    // Clear refresh cookie
    http.SetCookie(w, &http.Cookie{
        Name:     RefreshTokenCookieName,
        Value:    "",
        Path:     "/",
        MaxAge:   -1,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
    })

    w.WriteHeader(http.StatusNoContent)
}

// UserInfo returns authenticated OIDC details.
// Time Complexity: O(1)
// Space Complexity: O(1)
func (h *AuthHandler) UserInfo(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]string{
        "status": "authenticated",
    })
}

// Health readiness liveness check.
// Time Complexity: O(1)
// Space Complexity: O(1)
func (h *AuthHandler) Health(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]string{
        "status": "UP",
    })
}

func (h *AuthHandler) writeError(ctx context.Context, w http.ResponseWriter, code int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    _ = json.NewEncoder(w).Encode(map[string]string{
        "error": msg,
    })
}
