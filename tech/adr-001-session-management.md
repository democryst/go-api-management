# ADR-001: Session-Based PKCE State Management

## Status
🟢 **Decided**

---

## Context

In an Authorization Code Flow with PKCE (RFC 7636), the client must initiate the login process by generating a cryptographically secure random value (`code_verifier`) and a derived cryptographic challenge (`code_challenge`). The client redirects the user to the Identity Provider (Auth0) with the challenge.

When the Identity Provider redirects the user back to the Callback URL, it passes an authorization `code` and the `state` parameter. The server must intercept this callback, match the returned `state`, retrieve the corresponding `code_verifier`, and exchange both the code and verifier for the final token pair.

Because the user is redirected away from the application to the Auth0 domain and then redirected back, the server must persist the ephemeral `state`, `code_verifier`, and `redirect_uri` across separate stateless HTTP operations. We must choose a secure storage mechanism that protects these parameters from exposure and tampering.

---

## Alternatives Considered

### Option A: Stateless Encryption in Client Cookies
* **Mechanism**: Store the `state` and `code_verifier` in an encrypted, authentication-signed cookie in the user's browser.
* **Pros**: Simple, does not require database write/read overhead.
* **Cons**: Cookies containing access tokens or verifiers increase request headers payload size; if key leakage occurs, the verifier is compromised in-flight.

### Option B: Memory Cache (e.g. Redis or In-Memory Map)
* **Mechanism**: Persist transient sessions inside a key-value store or local Go sync map.
* **Pros**: Incredibly fast writes and reads (< 1ms).
* **Cons**: Local sync map is not horizontally scalable across multiple gateway nodes; Redis introduces a separate deployment dependency and cluster overhead.

### Option C: PostgreSQL-Backed Transient Sessions (Selected)
* **Mechanism**: Store sessions in a dedicated Postgres database table (`auth_sessions`) and link them via a cryptographically random UUIDv7 token passed to the browser as a secure, HTTP-only, `SameSite=Strict` cookie.
* **Pros**: Scalable across horizontal nodes, strong consistent ACID guarantees, zero additional dependencies since Postgres is already in use for core data, robust CSRF protection.
* **Cons**: Small DB read/write overhead per authentication flow.

---

## Decision

We will store ephemeral authorization session states inside a PostgreSQL database table named `auth_sessions`.

### Session Table Design
```sql
CREATE TABLE auth_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- Or UUIDv7
    state VARCHAR(255) NOT NULL UNIQUE,
    code_verifier VARCHAR(128) NOT NULL,
    redirect_uri TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Index for automatic TTL sweep logic
CREATE INDEX idx_auth_sessions_expires_at ON auth_sessions(expires_at);
```

### Flow Sequence
1. **Initiation**: When hitting `/auth/login`, generate state & verifier. Persist in database. Set `session_id` UUID in secure client cookie.
2. **Exchange**: Upon hitting `/auth/callback`, fetch session by cookie `session_id`, validate returned `state` match, perform token exchange, and immediately **delete** the row to enforce strict single-use limits.

---

## Consequences

* **CSRF Defended**: Strong state matching prevents request forgery.
* **Zero Leakage**: Ephemeral secrets (`code_verifier`) are strictly localized to the backend DB, never leaking to the frontend.
* **ACID Confirmed**: Database guarantees guarantee immediate deletion upon exchange callback, preventing verifier reuse replay attacks.

---
*Last Updated: 2026-05-23*
