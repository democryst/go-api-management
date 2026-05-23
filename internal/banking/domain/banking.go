package domain

import (
    "log/slog"
    "time"
)

// MaskedAccount represents a PII-redacted bank account or card number.
type MaskedAccount string

// String formats the account by masking middle digits (PCI-DSS compliant).
// Time Complexity: O(N) where N is the length of the account string.
// Space Complexity: O(N)
func (a MaskedAccount) String() string {
    str := string(a)
    if len(str) < 6 {
        return "***"
    }
    firstTwo := str[:2]
    lastFour := str[len(str)-4:]
    maskLen := len(str) - 6
    mask := ""
    for i := 0; i < maskLen; i++ {
        mask += "*"
    }
    return firstTwo + mask + lastFour
}

// LogValue satisfies the slog.LogValuer interface, returning a redacted representation.
// Time Complexity: O(N)
// Space Complexity: O(N)
func (a MaskedAccount) LogValue() slog.Value {
    return slog.StringValue(a.String())
}

// SecureEnclaveKey represents a hardware-backed device public key registered to a user.
type SecureEnclaveKey struct {
    ID           string    `db:"id" json:"id"`
    UserID       string    `db:"user_id" json:"user_id"`
    PublicKeyPEM string    `db:"public_key_pem" json:"public_key_pem"`
    Algorithm    string    `db:"algorithm" json:"algorithm"` // e.g. ECDSA_P256
    RegisteredAt time.Time `db:"registered_at" json:"registered_at"`
}

// TransferPayload represents a signed money-transfer transaction request.
type TransferPayload struct {
    ID              string        `json:"id"`
    SenderAccount   MaskedAccount `json:"sender_account"`
    ReceiverAccount MaskedAccount `json:"receiver_account"`
    AmountCents     int64         `json:"amount_cents"` // Represented in cents to prevent float precision loss
    Currency        string        `json:"currency"`     // e.g. USD
    Timestamp       time.Time     `json:"timestamp"`
    Signature       string        `json:"signature"`    // Hex-encoded or Base64-encoded Secure Enclave signature
}
