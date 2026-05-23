# Project Roadmap & Risk Registry: Secure Banking Transfer Gateway

This document establishes the official architectural roadmap, milestone timeline, module dependency locks, and risk registry for the `go-api-management` Secure Banking Gateway.

---

## 📅 1. Architectural Milestones & Timeline

To safely build out this multi-zone secure banking architecture, development is structured into four sequential, gate-controlled milestones:

### Milestone 1: Foundational Layers & Secure Domain (Day 1 - 3)
* **Goal**: Implement banking domain entities, ports (interfaces), error classifications, and DB persistence schemas.
* **Deliverables**:
  - Domain structs (`SecureEnclaveKey`, `TransferPayload`, `BFFSession`)
  - Ports interfaces (`AttestationVerifier`, `SessionRepository`, `EnclaveKeyRepository`)
  - SQL Migrations for Postgres session & device key tables (`auth_sessions` and `device_enclave_keys` tables)
  - Masking wrapper types for financial bank accounts.

### Milestone 2: Attestation & Cryptographic Validation Adapters (Day 4 - 6)
* **Goal**: Build Google Play Integrity/Apple App Attest verifier clients and hardware enclave signature verifiers.
* **Deliverables**:
  - Play Integrity JSON Web Signature (JWS) parsing utilities.
  - Secure Enclave ECDSA public key parsing and cryptographic verification engine (verifying transaction payloads signed by native mobile chips).
  - Unit test mock suites with pre-generated secure key pairs.

### Milestone 3: Token Swapper & Private mTLS peer Routing (Day 7 - 9)
* **Goal**: Build the DMZ-to-Secure Zone Token Swapper engine and establish firewalled mTLS HTTP clients.
* **Deliverables**:
  - HTTP handlers (`/auth/register-device`, `/auth/login`, `/transfer/initiate`, `/health`).
  - BFF-to-Internal Token Swap mapping controllers (issuing scoped internal JWTs).
  - Private HTTP peer client configurations with mutual TLS (mTLS) certificate configurations and `gobreaker` circuit breakers.
  - Logging middleware extracting trace correlation identifiers (`X-Request-ID`) into custom context handlers.

### Milestone 4: QA Integration, Attack Mocking & Audit (Day 10 - 12)
* **Goal**: Execute end-to-end integration tests, mock edge gateway compromise attack scenarios, and verify latency SLAs.
* **Deliverables**:
  - Automated integration tests verifying that modified transaction payloads fail signature checks.
  - Speed benchmarks validating local token validation latencies (p95 < 5ms).
  - Walkthrough validation of architectural and PCI-DSS compliance.

---

## 🔗 2. Dependency Lock & Modules Policy

The gateway is restricted to using these verified and approved Go modules to prevent supply-chain vulnerabilities:

| Module / Package Name | Category | Scope / Rationale |
| :--- | :--- | :--- |
| **`std/crypto/ecdsa`** | Cryptography | Hardware enclave signature checking (ECDSA P-256). |
| **`std/crypto/rand`** | Cryptography | Secure high-entropy generation of OIDC verifiers & state parameter values. |
| **`github.com/go-chi/chi/v5`** | Router | Clean, lightweight, standard-library compliant HTTP routing. |
| **`github.com/auth0/go-jwt-middleware/v3`** | Security | Official OIDC token verification and JWKS validation library. |
| **`github.com/jmoiron/sqlx`** | Persistence | Lightweight ORM-free SQL wrapper with structured row-mapping capabilities. |
| **`github.com/jackc/pgx/v5`** | Driver | Performance-optimized, pure Go PostgreSQL database driver. |
| **`github.com/sony/gobreaker`** | Resilience | Circuit breaker implementation to isolate third-party OIDC/attestation calls. |

---

## ⚡ 3. Technical Risk Registry & Mitigations

We identify the following technical risks and lock down their concrete code-level mitigations:

### Risk 01: DMZ API Gateway / BFF Layer Compromise
* **Impact**: A hacker gains full root access to the public-facing DMZ API Gateway, gaining the ability to intercept calls, inject fake transaction payloads, and manipulate database states.
* **Mitigation**: Implement **Secure Enclave Non-Repudiation**. The Money Transfer service in the isolated Secure Intranet Zone validates the transaction payload signature *directly* against the registered user public key stored inside the intranet DB. Because the private key is physically locked inside the user's phone hardware, a compromised gateway cannot forge a money-transfer transaction.

### Risk 02: PII Financial Account Data Leaks in Application Logs
* **Impact**: Financial account numbers, routing numbers, or card data are written to debug logging, violating PCI-DSS compliance and exposing users.
* **Mitigation**: Implement custom banking masking types (e.g. `MaskedAccount`) implementing `slog.LogValuer` and `fmt.Stringer`. Every log path will redact account digits (e.g., `12******3456`), preventing accidental exposure in plaintext.

### Risk 03: Jailbroken Device Verification Bypass
* **Impact**: A user jailbreaks their device to bypass application checks, spoof local parameters, and compromise transaction safety.
* **Mitigation**: Enforce **App Attest / Play Integrity** verification at the Edge. Cryptographically signed tokens from Apple/Google are decoded and checked against registered App IDs on the backend on every session request, immediately rejecting modified runtimes.

### Risk 04: External API Gateway Outages (e.g. Auth0 / Attestation servers)
* **Impact**: Egress attestation or verification API requests freeze, locking connection pools, exhausting DB connections, and knocking down the banking gateway.
* **Mitigation**: Wrap all external client calls within `gobreaker` circuit breakers. The breakers will immediately fail-fast when failure ratios exceed 50% over a 10s sliding window, protecting system pool resources.

---
*Last Updated: 2026-05-23*
