package domain

import "time"

// AuthSession represents the ephemeral authorization state holding PKCE verifiers.
type AuthSession struct {
    ID           string    `db:"id" json:"id"`
    State        string    `db:"state" json:"state"`
    CodeVerifier string    `db:"code_verifier" json:"code_verifier"`
    RedirectURI  string    `db:"redirect_uri" json:"redirect_uri"`
    CreatedAt    time.Time `db:"created_at" json:"created_at"`
    ExpiresAt    time.Time `db:"expires_at" json:"expires_at"`
}

// IsExpired checks if the temporary authorization session has surpassed its TTL expiration window.
// Time Complexity: O(1)
// Space Complexity: O(1)
func (s *AuthSession) IsExpired() bool {
    return time.Now().After(s.ExpiresAt)
}
