package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// InternalJWTClaims represents the claims encoded in the short-lived intranet token.
type InternalJWTClaims struct {
	Subject   string   `json:"sub"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	Scopes    []string `json:"scopes"`
}

// GenerateInternalJWT generates a highly privileged, short-lived Internal JWT signed via HMAC-SHA256.
// Time Complexity: O(1)
// Space Complexity: O(1)
func GenerateInternalJWT(subject string, scopes []string, secret string, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("empty shared secret")
	}

	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := InternalJWTClaims{
		Subject:   subject,
		ExpiresAt: now.Add(ttl).Unix(),
		IssuedAt:  now.Unix(),
		Scopes:    scopes,
	}
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerBytes)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsBytes)

	signingInput := headerB64 + "." + claimsB64

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	sigBytes := mac.Sum(nil)

	sigB64 := base64.RawURLEncoding.EncodeToString(sigBytes)

	return signingInput + "." + sigB64, nil
}

// VerifyInternalJWT cryptographically validates the signature and expiration of an Internal JWT.
// Time Complexity: O(1)
// Space Complexity: O(1)
func VerifyInternalJWT(token string, secret string) (*InternalJWTClaims, error) {
	if secret == "" {
		return nil, errors.New("empty shared secret")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	headerB64, claimsB64, sigB64 := parts[0], parts[1], parts[2]

	signingInput := headerB64 + "." + claimsB64
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expectedSig := mac.Sum(nil)
	expectedSigB64 := base64.RawURLEncoding.EncodeToString(expectedSig)

	if !hmac.Equal([]byte(sigB64), []byte(expectedSigB64)) {
		return nil, errors.New("invalid signature")
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(claimsB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	var claims InternalJWTClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	if time.Now().Unix() > claims.ExpiresAt {
		return nil, errors.New("token is expired")
	}

	return &claims, nil
}
