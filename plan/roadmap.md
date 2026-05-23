# Project Roadmap & Risk Registry

This document establishes the official architectural roadmap, milestone timeline, package dependency locks, and risk registry for the `go-api-management` Auth Gateway.

---

## 📅 1. Architectural Milestones & Timeline

The project is structured into four distinct engineering milestones to ensure safe and decoupled layer implementation:

### Milestone 1: Foundational Layers & Core Domain (Day 1 - 2)
* **Goal**: Implement primary domain entities, ports (interfaces), error classifications, and DB persistence schemas.
* **Deliverables**:
  - Domain structs (`AuthSession`, `TokenPair`, `UserIdentity`)
  - Ports interfaces (`AuthProvider`, `SessionRepository`)
  - SQL Schema + Migration scripts for Postgres session persistence (`auth_sessions` table)
  - Memory-safe PII masking wrapper types.

### Milestone 2: Cryptographic Logic & Auth0 Gateway Adapter (Day 3 - 5)
* **Goal**: Write PKCE code verifier and S256 challenge logic; construct the Auth0 external adapter implementing `AuthProvider`.
* **Deliverables**:
  - `crypto/rand` based PKCE helper utility.
  - Auth0 HTTP Adapter with internal request-response models and domain type mapping.
  - Circuit Breaker (`gobreaker`) integration around Auth0 egress HTTP connections.

### Milestone 3: HTTP Handlers, Middleware & Observability (Day 6 - 8)
* **Goal**: Configure routing, build controllers, integrate JWT verification, and wire request tracing metrics.
* **Deliverables**:
  - HTTP handlers (`/auth/login`, `/auth/callback`, `/auth/logout`, `/auth/refresh`, `/auth/userinfo`, `/health`)
  - JWKS Caching Provider middleware (`auth0/go-jwt-middleware/v3` with a 5-minute TTL)
  - Logger correlation middleware extracting trace correlation identifiers (`X-Request-ID`) into `context.Context` slog handlers.

### Milestone 4: QA Verification, Integration Tests & Security Audit (Day 9 - 10)
* **Goal**: Run test suites, verify latencies SLAs, ensure hexagonal boundary integrity, and compile build outputs.
* **Deliverables**:
  - Mock-backed unit tests for Services and Gateways.
  - Real database integration tests for session repository.
  - Walkthrough validation of SLAs and architectural compliance.

---

## 🔗 2. Dependency Lock & Modules Policy

To comply with the **Library Lockdown** directive, the service is restricted to using these verified and approved Go modules:

| Module / Package Name | Category | Scope / Rationale |
| :--- | :--- | :--- |
| **`std/crypto/rand`** | Cryptography | Secure high-entropy generation of PKCE verifiers & state parameter values. |
| **`std/log/slog`** | Observability | Native high-performance structured logging. |
| **`github.com/go-chi/chi/v5`** | Router | Clean, lightweight, standard-library compliant HTTP routing. |
| **`github.com/auth0/go-jwt-middleware/v3`** | Security | Official Auth0 RS256 token verification and JWKS validation library. |
| **`github.com/jmoiron/sqlx`** | Persistence | Lightweight ORM-free SQL wrapper with structured row-mapping capabilities. |
| **`github.com/jackc/pgx/v5`** | Driver | Performance-optimized, pure Go PostgreSQL database driver. |
| **`github.com/sony/gobreaker`** | Resilience | Circuit breaker implementation to isolate Auth0 egress API calls. |

---

## ⚡ 3. Technical Risk Registry & Mitigations

We identify the following technical risks and lock down their concrete code-level mitigations:

### Risk 01: Auth0 JWKS Rate Limiting
* **Impact**: External requests to `/.well-known/jwks.json` on every single incoming API call will exceed Auth0 rate limits, resulting in `429 Too Many Requests` and complete gateway outage.
* **Mitigation**: Implement `jwks.NewCachingProvider` within the JWT middleware. The public keys will be cached in memory for **5 minutes**, ensuring only 1 egress request is made per key interval.

### Risk 02: PII Data Leaking in Application Logs
* **Impact**: Under high troubleshooting load, engineers may dump user entities containing emails or names into trace logging, violating compliance regulations.
* **Mitigation**: Implement custom masking types for `Email` and `Username` implementing `slog.LogValuer` and `fmt.Stringer`. Every log path will output filtered data (e.g., `p***@example.com`), preventing accidental exposure in plaintext.

### Risk 03: Insecure or Padded PKCE Challenge Values
* **Impact**: Base64 encoding standard libraries add `=` padding by default. Auth0 silences verification of padded values, causing authentication failures.
* **Mitigation**: Explicitly mandate `base64.RawURLEncoding` (which omits padding character sequences) for the generation of all `code_verifier` and `code_challenge` values.

### Risk 04: Auth0 Egress Downtime or Outage Cascades
* **Impact**: Network errors or Auth0 system outages freeze incoming callback operations, locking goroutines, exhausting DB connection pools, and knocking down the entire gateway.
* **Mitigation**: Wrap the `Auth0Gateway` client within a `gobreaker.CircuitBreaker`. The breaker will immediately fail-fast when egress failure rates exceed 50% over a 10s sliding window, protecting system pool resources.

---
*Last Updated: 2026-05-23*
