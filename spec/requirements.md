# Requirements Specification: OAuth 2.0 + PKCE + Auth0 Gateway

This document represents the formal requirements specification for the high-performance Go authentication gateway. All system designs, interface definitions, and implementations must trace back to these rules.

## 🎯 1. Scope & System Boundary

The authentication gateway provides a production-grade secure bridge between client applications and Auth0 as the Identity Provider (IdP) using the **Authorization Code Flow with PKCE (RFC 7636)**.

### In Scope
* **RFC 7636 PKCE Flow**: Explicit authorization code exchange using `S256` hashing (plain text verifiers are strictly prohibited).
* **Token Validation**: Signature verification using RS256 via Auth0 JWKS with 5-minute local caching.
* **Refresh Token Rotation**: Automatic rotation with single-use reuse detection to mitigate token theft.
* **OpenAPI 3.1 Documentation**: Clean endpoint definitions tagged with `x-usePkce: true`.
* **PII Masking**: Mandatory logic-level masking on logging keys containing email addresses, names, or secret strings.

### Out of Scope
* Custom user registration UI or username/password store.
* Session management for resource endpoints outside the authentication scope.
* DPoP (Demonstrating Proof-of-Possession) or multi-tenant delegation support in this version.

---

## 🛠️ 2. Functional Requirements

### FR-01: PKCE Challenge Creation
* The system must generate a cryptographically secure, random 43-character `code_verifier` (from the unreserved set `[A-Z][a-z][0-9]-._~`) using a high-entropy source (`crypto/rand`).
* The corresponding `code_challenge` must be calculated using `BASE64URL-ENCODE(SHA256(ASCII(code_verifier)))` without padding.

### FR-02: Secure Session Storing
* The system must persist the ephemeral `state` and `code_verifier` temporarily in a PostgreSQL session table before redirecting the client to Auth0.
* A cryptographically secure session key (UUIDv7) must be returned to the client as an HTTP-only, secure, `SameSite=Strict` session cookie.

### FR-03: Code-to-Token Exchange
* Upon callback, the system must retrieve the session by its UUIDv7 cookie, validate the returned `state` against the stored state, and immediately delete the session from the DB.
* The system must make an explicit egress HTTP POST call to Auth0's `/oauth/token` containing the `code`, `code_verifier`, `client_id`, and `client_secret` (if applicable) to receive the `TokenPair`.

### FR-04: JWKS-Backed Token Validation
* All secure endpoints must intercept requests, validate the RS256 JWT in the `Authorization: Bearer <token>` header, verify claims (`iss`, `aud`, `exp`, `nbf`), and extract the user's `sub` identifier.
* The public keys must be retrieved from the Auth0 JWKS endpoint (`/.well-known/jwks.json`) and cached locally for **5 minutes** to prevent rate limits.

---

## 🔒 3. Data Security & Handling Classifications

All application data must be handled according to its strict classification profile:

| Data Type | Classification | Policy & Security Handling |
| :--- | :--- | :--- |
| **Access Token (JWT)** | Sensitive | Transmitted via `Authorization` header only. Never stored in logs or database tables. TLS 1.2+ mandatory on all transport paths. |
| **Refresh Token** | Secret | Returned to client strictly in a secure, HTTP-only, `SameSite=Strict` cookie. Ephemeral storage in database only if needed for rotation lookup. |
| **code_verifier** | Ephemeral Secret | In-memory only. Must never be written to persistent database, logs, or caches. Discarded immediately after token exchange. |
| **User Email / Name** | PII | Must be parsed and masked before logging. (e.g., `p***@example.com` or `J*** D***`). |
| **Auth0 Client Secret** | System Secret | Injected strictly via environment variables or secure credentials vaults. Never committed to source control or dumped in trace files. |

---

## 📊 4. Non-Functional Requirements & SLAs

### NFR-01: Latency Budgets
* **JWT Local Validation**: p95 < 5ms, p99 < 15ms.
* **Complete Login Redirect Initiation**: p95 < 30ms.
* **Token Exchange / Refresh Processing (Excluding Auth0 Egress latency)**: p95 < 20ms.

### NFR-02: High Availability & Fault Tolerance
* Egress calls to Auth0 must utilize a circuit breaker (via `gobreaker`) configured with a fail-ratio of 50% over a 10-second window, preventing cascade failures.
* Requests exceeding 3 seconds on egress calls must be aborted and gracefully returned as a system error (`AuthError`).

---

## 🧬 5. API Endpoints Specification

* `GET /auth/login` - Initiates the authorization code flow, creates session verifiers, and redirects to Auth0.
* `GET /auth/callback` - Callback handler that exchanges the authorization code + PKCE verifier for tokens.
* `POST /auth/logout` - Revokes active sessions and clears the secure HTTP-only cookies.
* `POST /auth/refresh` - Executes single-use refresh token rotation and issues a new `TokenPair`.
* `GET /auth/userinfo` - Returns profile details (`UserIdentity`) for the active authorized session.
* `GET /health` - Liveness/Readiness probe returning structural and DB connection status.

---
*Last Updated: 2026-05-23*
