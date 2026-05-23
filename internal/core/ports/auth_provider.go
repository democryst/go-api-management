package ports

import (
    "context"
    "github.com/democryst/go-api-management/internal/core/domain"
)

// AuthProvider defines the outbound port boundary for communicating with Auth0 IdP.
type AuthProvider interface {
    // GetAuthorizationURL builds the Auth0 redirect link with S256 PKCE challenges.
    GetAuthorizationURL(ctx context.Context, session domain.AuthSession) (string, error)

    // ExchangeCode exchanges the raw authorization code + unpadded verifier for token sets.
    ExchangeCode(ctx context.Context, code, codeVerifier string) (domain.TokenPair, error)

    // RefreshToken executes single-use refresh token rotation to obtain fresh credentials.
    RefreshToken(ctx context.Context, refreshToken string) (domain.TokenPair, error)

    // RevokeToken signs out the session and revokes the corresponding tokens.
    RevokeToken(ctx context.Context, token string) error

    // GetUserInfo retrieves OIDC profile attributes associated with the access token.
    GetUserInfo(ctx context.Context, accessToken string) (domain.UserIdentity, error)
}
