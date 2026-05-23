package crypto

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
)

// GenerateState generates a high-entropy 32-byte CSRF mitigation state parameter.
// Time Complexity: O(1)
// Space Complexity: O(1) (fixed buffer size)
func GenerateState() (string, error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", fmt.Errorf("read random bytes for state: %w", err)
    }
    return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// GenerateVerifier generates a high-entropy 32-byte PKCE code_verifier (RFC 7636 §4.1).
// Time Complexity: O(1)
// Space Complexity: O(1) (fixed buffer size)
func GenerateVerifier() (string, error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", fmt.Errorf("read random bytes for PKCE verifier: %w", err)
    }
    return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// DeriveChallenge derives the PKCE S256 code_challenge from a given code_verifier (RFC 7636 §4.2).
// Time Complexity: O(N) where N is the length of the verifier string.
// Space Complexity: O(1) (fixed output hash buffer size)
func DeriveChallenge(codeVerifier string) string {
    hash := sha256.Sum256([]byte(codeVerifier))
    return base64.RawURLEncoding.EncodeToString(hash[:])
}
