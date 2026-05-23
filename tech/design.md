# System Design: High-Performance Go Auth Gateway

This document provides the technical design, sequence flows, data structures, and architectural wiring for the Go API Gateway.

---

## 🏛️ 1. Hexagonal Layer Blueprints & Folder Layout

The gateway enforces the **Hexagonal (Ports and Adapters) Architecture** to ensure core domain logic is decoupled from frameworks, driver clients, and infrastructure APIs.

```text
go-api-management/
├── cmd/
│   └── gateway/
│       └── main.go         # DI Wiring & Entrypoint [Entry]
├── internal/
│   ├── core/
│   │   ├── domain/         # Core Domain Entities [Sacred Domain]
│   │   │   ├── session.go
│   │   │   ├── token.go
│   │   │   └── user.go
│   │   ├── ports/          # Interfaces (Inbound & Outbound) [Ports]
│   │   │   ├── auth_provider.go
│   │   │   └── session_repo.go
│   │   └── services/       # Core Business Orchestration [Services]
│   │       └── auth_service.go
│   └── adapters/
│       ├── gateway/        # Auth0 Egress Client Adapter [Outbound Adapter]
│       │   └── auth0_gateway.go
│       ├── handler/        # HTTP Chi Controller Handler [Inbound Adapter]
│       │   └── auth_handler.go
│       ├── repository/     # SQLX Database Session Repository [Outbound Adapter]
│       │   └── sql_session_repository.go
│       └── telemetry/      # Context-Aware Logger [Observability]
│           └── slog_logger.go
```

### The Layer Dependency Rule
* **adapters** layer may import from **ports** and **domain** layers.
* **services** and **ports** layers may import from the **domain** layer.
* **domain** and **ports** layers have **zero external imports** and must never depend on the **adapters** layer (zero-dependency core).

---

## 🔄 2. Key Sequences & Control Flows

The following sequences illustrate how requests traverse our adapters, execute domain calculations via services, and interact with the database and external IdP.

### A. Authorization Code + PKCE Initiation Flow
This sequence generates the high-entropy verifier, challenge, state, registers the transient session in PostgreSQL, and redirects the client browser.

```mermaid
sequenceDiagram
    autonumber
    actor User as Browser / Client
    participant H as AuthHandler (Chi Adapter)
    participant S as AuthService (Domain Core)
    participant R as SQLSessionRepository (DB Adapter)
    participant P as Auth0Gateway (IdP Adapter)
    database DB as PostgreSQL

    User->>H: GET /auth/login?redirect_uri=...
    H->>S: InitiateFlow(ctx, redirect_uri)
    Note over S: 1. Generate crypto-secure State<br/>2. Generate high-entropy verifier<br/>3. Calculate BASE64URL(SHA256(verifier))
    S->>R: SaveSession(ctx, Session)
    R->>DB: INSERT INTO auth_sessions
    DB-->>R: Row Created
    S->>P: GetAuthorizationURL(ctx, Session)
    P-->>S: Redirect URL (with challenge & S256)
    S-->>H: Redirect URL + Session UUID
    Note over H: Set Secure, HTTP-Only Cookie<br/>session_id = UUIDv7
    H-->>User: 302 Found (Redirect to Auth0)
```

### B. Callback Verification & Exchange Flow
Enforces state validation, instant single-use session destruction (to protect against replay attacks), and exchanges the authorization code.

```mermaid
sequenceDiagram
    autonumber
    actor User as Browser / Client
    participant H as AuthHandler (Chi Adapter)
    participant S as AuthService (Domain Core)
    participant R as SQLSessionRepository (DB Adapter)
    participant P as Auth0Gateway (IdP Adapter)
    database DB as PostgreSQL

    User->>H: GET /auth/callback?code=...&state=...
    Note over H: Read session_id Cookie (UUIDv7)
    H->>S: CompleteCallback(ctx, session_id, code, state)
    S->>R: GetSession(ctx, session_id)
    R->>DB: SELECT FROM auth_sessions WHERE id = UUIDv7
    DB-->>R: Return Session
    Note over S: Verify returned state matches DB state
    S->>R: DeleteSession(ctx, session_id)
    R->>DB: DELETE FROM auth_sessions WHERE id = UUIDv7
    DB-->>R: Row Deleted
    S->>P: ExchangeCode(ctx, code, verifier)
    Note over P: Egress POST /oauth/token (gobreaker wrapped)
    P-->>S: Return TokenPair (Access + Refresh)
    S-->>H: Return UserIdentity + TokenPair
    Note over H: Set HTTP-Only Secure SameSite cookie with refresh token
    H-->>User: Return UserIdentity (JSON)
```

---

## 💾 3. Database Schema

### PostgreSQL Session Database Table
```sql
CREATE TABLE auth_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    state VARCHAR(255) NOT NULL UNIQUE,
    code_verifier VARCHAR(128) NOT NULL,
    redirect_uri TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Index for automatic background purge tasks
CREATE INDEX idx_auth_sessions_expires_at ON auth_sessions(expires_at);
```

---

## 🔒 4. Cryptographic Specifications (RFC 7636)

To prevent Auth0 authorization failures, all generations must adhere strictly to these rules:

1. **High Entropy Source**: Always utilize `crypto/rand` to read bytes, avoiding predictable seed generators.
2. **Character Set Verification**: The verifier string must strictly consist of unreserved URL-safe characters.
3. **No base64 padding**: Ensure we use `base64.RawURLEncoding.EncodeToString` to prevent `=` signs.

```go
// CodeVerifier generation implementation
verifierBytes := make([]byte, 32)
if _, err := rand.Read(verifierBytes); err != nil {
    return nil, fmt.Errorf("read random: %w", err)
}
codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes) // Result: 43 characters

// S256 Challenge derivation implementation
hash := sha256.Sum256([]byte(codeVerifier))
codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
```

---

## ⚡ 5. Resilience Circuit Breakers

To isolate third-party outages, all egress HTTP requests made within `Auth0Gateway` are wrapped inside a circuit breaker:

```go
var cb = gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:        "auth0-api-gateway",
    MaxRequests: 5,
    Interval:    10 * time.Second,
    Timeout:     5 * time.Second,
    ReadyToTrip: func(counts gobreaker.Counts) bool {
        failureRatio := float64(counts.ConsecutiveFailures) / float64(counts.Requests)
        return counts.Requests >= 3 && failureRatio >= 0.5
    },
})
```
* If Auth0 returns failures on 50% of requests over a sliding window, downstream requests fail fast instantly for 5 seconds, protecting the gateway resources.

---
*Last Updated: 2026-05-23*
