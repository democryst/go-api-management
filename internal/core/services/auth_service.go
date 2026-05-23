package services

import (
    "context"
    "crypto/rand"
    "fmt"
    "log/slog"
    "time"

    "github.com/democryst/go-api-management/internal/adapters/crypto"
    "github.com/democryst/go-api-management/internal/core/domain"
    "github.com/democryst/go-api-management/internal/core/ports"
)

// AuthService orchestrates the entire high-performance OIDC PKCE flow.
type AuthService struct {
    repo     ports.SessionRepository
    provider ports.AuthProvider
    logger   *slog.Logger
}

// NewAuthService constructs a new AuthService orchestrator.
func NewAuthService(
    repo ports.SessionRepository,
    provider ports.AuthProvider,
    logger *slog.Logger,
) *AuthService {
    return &AuthService{
        repo:     repo,
        provider: provider,
        logger:   logger,
    }
}

// InitiateAuth generates cryptographically random states and verifiers, stores them in DB, and returns the redirect link.
// Time Complexity: O(1) (cryptographic and index DB inserts)
// Space Complexity: O(1)
func (s *AuthService) InitiateAuth(ctx context.Context, redirectURI string) (string, string, error) {
    state, err := crypto.GenerateState()
    if err != nil {
        return "", "", fmt.Errorf("generate state: %w", err)
    }

    verifier, err := crypto.GenerateVerifier()
    if err != nil {
        return "", "", fmt.Errorf("generate verifier: %w", err)
    }

    challenge := crypto.DeriveChallenge(verifier)
    sessionID := s.generateUUIDv4()

    session := &domain.AuthSession{
        ID:           sessionID,
        State:        state,
        CodeVerifier: verifier, // DB session stores the verifier
        RedirectURI:  redirectURI,
        CreatedAt:    time.Now(),
        ExpiresAt:    time.Now().Add(5 * time.Minute), // Strict 5-minute transient window
    }

    if err := s.repo.Save(ctx, session); err != nil {
        return "", "", fmt.Errorf("persist session: %w", err)
    }

    // Prepare redirect context by passing the challenge derived from the verifier
    redirectSession := *session
    redirectSession.CodeVerifier = challenge // Overwrite verifier with S256 challenge string

    authURL, err := s.provider.GetAuthorizationURL(ctx, redirectSession)
    if err != nil {
        return "", "", fmt.Errorf("fetch auth provider URL: %w", err)
    }

    s.logger.InfoContext(ctx, "initiated PKCE authorization sequence", slog.String("session_id", sessionID))
    return authURL, sessionID, nil
}

// HandleCallback verifies state parameters, instantly deletes the transient session, and performs token code exchange.
// Time Complexity: O(1) (network and indexed DB lookup bound)
// Space Complexity: O(1)
func (s *AuthService) HandleCallback(ctx context.Context, sessionID, code, state string) (domain.UserIdentity, domain.TokenPair, error) {
    session, err := s.repo.Get(ctx, sessionID)
    if err != nil {
        return domain.UserIdentity{}, domain.TokenPair{}, fmt.Errorf("retrieve auth session: %w", err)
    }

    // Instantly delete database record (Single-Use enforcement defending against callback replays)
    _ = s.repo.Delete(ctx, sessionID)

    if session.IsExpired() {
        return domain.UserIdentity{}, domain.TokenPair{}, fmt.Errorf("callback verification failed: %w", domain.ErrSessionExpired)
    }

    if session.State != state {
        return domain.UserIdentity{}, domain.TokenPair{}, fmt.Errorf("callback verification failed: %w", domain.ErrInvalidState)
    }

    tokens, err := s.provider.ExchangeCode(ctx, code, session.CodeVerifier)
    if err != nil {
        return domain.UserIdentity{}, domain.TokenPair{}, fmt.Errorf("exchange authorization code: %w", err)
    }

    user, err := s.provider.GetUserInfo(ctx, tokens.AccessToken)
    if err != nil {
        return domain.UserIdentity{}, domain.TokenPair{}, fmt.Errorf("retrieve user profile: %w", err)
    }

    s.logger.InfoContext(ctx, "completed authentication callback", slog.String("user_id", user.ID), slog.Any("user_email", user.Email))
    return user, tokens, nil
}

// RefreshToken executes refresh token rotation to obtain fresh credentials.
// Time Complexity: O(1) (network bound)
// Space Complexity: O(1)
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (domain.TokenPair, error) {
    tokens, err := s.provider.RefreshToken(ctx, refreshToken)
    if err != nil {
        return domain.TokenPair{}, fmt.Errorf("rotate refresh token: %w", err)
    }
    return tokens, nil
}

// RevokeToken signs out the session and revokes active tokens on the IdP.
// Time Complexity: O(1) (network bound)
// Space Complexity: O(1)
func (s *AuthService) RevokeToken(ctx context.Context, token string) error {
    if err := s.provider.RevokeToken(ctx, token); err != nil {
        return fmt.Errorf("revoke session token: %w", err)
    }
    return nil
}

func (s *AuthService) generateUUIDv4() string {
    uuid := make([]byte, 16)
    _, _ = rand.Read(uuid)
    uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
    uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant 10
    return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}
