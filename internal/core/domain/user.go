package domain

import (
    "log/slog"
    "strings"
)

// MaskedEmail represents a PII-redacted user email type.
type MaskedEmail string

// String formats the email by masking characters preceding the domain separator.
// Time Complexity: O(N) where N is the length of the email address.
// Space Complexity: O(N) to construct the return slice.
func (e MaskedEmail) String() string {
    parts := strings.SplitN(string(e), "@", 2)
    if len(parts) != 2 {
        return "***"
    }
    prefix := parts[0]
    domain := parts[1]
    if len(prefix) <= 1 {
        return prefix + "***@" + domain
    }
    return prefix[:1] + "***@" + domain
}

// LogValue satisfies the slog.LogValuer interface, returning a safe, masked representation.
// Time Complexity: O(N)
// Space Complexity: O(N)
func (e MaskedEmail) LogValue() slog.Value {
    return slog.StringValue(e.String())
}

// MaskedName represents a PII-redacted user name type.
type MaskedName string

// String formats the user's full name by masking individual name segments.
// Time Complexity: O(N) where N is the number of characters in the name.
// Space Complexity: O(N)
func (n MaskedName) String() string {
    parts := strings.Fields(string(n))
    if len(parts) == 0 {
        return "***"
    }
    var masked []string
    for _, part := range parts {
        if len(part) <= 1 {
            masked = append(masked, part+"***")
        } else {
            masked = append(masked, part[:1]+"***")
        }
    }
    return strings.Join(masked, " ")
}

// LogValue satisfies the slog.LogValuer interface.
// Time Complexity: O(N)
// Space Complexity: O(N)
func (n MaskedName) LogValue() slog.Value {
    return slog.StringValue(n.String())
}

// UserIdentity represents an authenticated user's profile with PII-safe loggers.
type UserIdentity struct {
    ID           string      `json:"sub"`
    Email        MaskedEmail `json:"email"`
    Name         MaskedName  `json:"name"`
    Picture      string      `json:"picture,omitempty"`
    EmailVerified bool       `json:"email_verified"`
}
