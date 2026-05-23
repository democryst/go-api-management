# ADR-002: JWT Validation & JWKS Caching Strategy

## Status
🟢 **Decided**

---

## Context

The gateway acts as an API Resource Server that must secure access to downstream resource paths. Every incoming request must provide an `Authorization: Bearer <token>` access token containing signed JSON Web Tokens (JWT).

Because the Identity Provider (Auth0) uses asymmetric encryption (RS256), the gateway must verify the token's signature using the IdP's public JSON Web Key Set (JWKS). Fetching the public keys via HTTP on *every* incoming request adds significant latency overhead (> 100ms) and quickly triggers rate limiting blocks on the Auth0 endpoint (`429 Too Many Requests`), taking down the service.

---

## Alternatives Considered

### Option A: Manual HTTP Fetch on Signature Failures
* **Mechanism**: Fetch the keys and cache them indefinitely. If validation fails, re-fetch once to support key rotation.
* **Pros**: Simple cache logic.
* **Cons**: Unnecessary edge failures during key rotations; prone to caching stampedes if key changes occur.

### Option B: Layered In-Memory Caching Provider (Selected)
* **Mechanism**: Use the official `auth0/go-jwt-middleware/v3` integrated with `jwks.NewCachingProvider`. Set a strict cache Time-To-Live (TTL) of **5 minutes**.
* **Pros**: Robust key caching, low local latency (< 5ms), native key-rotation handling, official sdk alignment.
* **Cons**: None.

---

## Decision

We will integrate `auth0/go-jwt-middleware/v3` utilizing a local `jwks.NewCachingProvider` configuration.

### Implementation Blueprint
```go
import (
    "net/url"
    "time"
    "github.com/auth0/go-jwt-middleware/v3"
    "github.com/auth0/go-jwt-middleware/v3/validator"
)

// Initialize caching provider
issuerURL, _ := url.Parse("https://" + auth0Domain + "/")
provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

// Configure token validator
jwtValidator, err := validator.New(
    provider.KeyFunc,
    validator.RS256, // Enforce RS256 algorithm signature validation
    issuerURL.String(),
    []string{apiAudience},
)
```

---

## Consequences

* **Sub-Millisecond Verification**: Local cache lookups keep token validation overhead below 5ms (typically < 1ms), satisfying our strict latency SLAs.
* **Algorithm Isolation**: Enforcing `validator.RS256` prevents `alg:none` validation bypass vulnerabilities.
* **Fault Tolerant Key Rotation**: The 5-minute TTL guarantees automated caching refresh without manual intervention or restart cascades.

---
*Last Updated: 2026-05-23*
