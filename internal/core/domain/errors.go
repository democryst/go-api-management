package domain

import "errors"

// AuthError represents a typed domain-specific authentication error.
type AuthError string

func (e AuthError) Error() string {
    return string(e)
}

// Explicit domain-level errors preventing silent or implicit exception patterns
const (
    ErrSessionNotFound      AuthError = "auth_session_not_found"
    ErrSessionExpired       AuthError = "auth_session_expired"
    ErrInvalidState         AuthError = "invalid_oauth_state"
    ErrEgressFailed         AuthError = "external_identity_provider_unreachable"
    ErrTokenExchangeFailed  AuthError = "oauth_token_exchange_failed"
    ErrUnauthorized         AuthError = "user_identity_unauthorized"
)

// IsAuthError helper checks if a generic error matches a specific AuthError.
// Time Complexity: O(1)
// Space Complexity: O(1)
func IsAuthError(err error, target AuthError) bool {
    var authErr AuthError
    if errors.As(err, &authErr) {
        return authErr == target
    }
    return false
}
