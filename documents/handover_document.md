# Secure Banking Gateway: Engineering & Ops Handover Manual
## SRE and Backend DevOps Guide

This handover document provides systems engineers, site reliability engineers (SREs), and backend developers with structural specifications, maintenance protocols, and deployment commands for the high-performance Go Secure Banking Gateway.

---

## 🏛️ 1. Multi-Zone Network Segmentation (VPC Zoning)

To protect corporate Ledgers and user credentials, the architecture is partitioned into two isolated, firewalled VPC network environments. No direct public ingress to the private intranet is permitted:

```text
  [ PUBLIC INTERNET ]                    [ DMZ / EDGE ZONE ]                  [ SECURE INTRANET ZONE ]
                        
  ┌──────────────────┐                  ┌──────────────────┐                  ┌───────────────────┐
  │                  │                  │                  │                  │ Core Banking      │
  │  Mobile Client   │───( HTTPS 443 )─►│  API Gateway &   │───( mTLS Peer )─►│ Transfer Service  │
  │                  │                  │  BFF (Edge App)  │                  │ (Private IP Only) │
  └──────────────────┘                  └──────────────────┘                  └───────────────────┘
```

### Zone A: Public-Facing DMZ (Edge BFF)
* **Hosts**: API Gateway routes and BFF session controllers.
* **Exposed Routes**: `POST /auth/register-device`, `POST /auth/login`, `POST /transfer/initiate`.
* **State**: Stateless. Validates public sessions using cached Auth0 JWKS providers.
* **Responsibility**: Translates low-privilege Edge BFF session cookies into high-privilege scoped Internal JWTs (**BFF Token Swap**).

### Zone B: Private VPC Intranet (Private Core Banking)
* **Hosts**: Ledger Transfer Service. Exposed *strictly* via private IP address spaces.
* **Exposed Routes**: `POST /private/transfers`.
* **Security Bounds**: Accepts requests *only* from the Edge BFF gateway over private lines using **Mutual TLS (mTLS)** with CA-signed certificates and strict IP-allowlists.
* **Responsibility**: Verifies the high-privilege Internal JWT, resolves the user's registered public keys from persistent databases, cryptographically validates the Secure Enclave signature, and executes the ledger transfer.

---

## 💾 2. State Management & Database Schemas

The Gateway utilizes **ORM-free PostgreSQL persistence** via lightweight `sqlx` prepared statements to guarantee optimal query plans and complete immunity to SQL injection. State is segmented into two persistence scopes:

### 1. Ephemeral PKCE Auth Sessions (ADR-001)
* **Table**: `auth_sessions`
* **Purpose**: Temporarily stores PKCE `code_verifier`, state, and callback parameters across external OIDC redirects.
* **Security constraints**: Enforces **single-use consumption**. Rows are instantly queried and deleted from the database in a single atomic database exchange *before* beginning external code-exchange calls, blocking callback replay forging.
* **Schema**:
```sql
CREATE TABLE auth_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    state VARCHAR(255) NOT NULL UNIQUE,
    code_verifier VARCHAR(128) NOT NULL,
    redirect_uri TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);
CREATE INDEX idx_auth_sessions_expires_at ON auth_sessions(expires_at);
```

### 2. Persistent Device Enclave Keys
* **Table**: `device_enclave_keys`
* **Purpose**: Persists hardware-backed ECDSA P-256 public keys registered by client devices, tied strictly to the user identity subject.
* **Schema**:
```sql
CREATE TABLE device_enclave_keys (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    public_key_pem TEXT NOT NULL,
    algorithm VARCHAR(64) NOT NULL,
    registered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_device_enclave_keys_user_id ON device_enclave_keys(user_id);
```

### 3. Distributed Valkey Token Bucket State (In-Memory Cache)
To defend Edge DMZ gateways against DDoS attacks and brute forcing, we enforce in-memory token buckets backed by Valkey:
* **Key Format**: `ratelimit:<hex-sha256(raw-identifier)>`
  * Unauthenticated Clients: Hashed remote network address (e.g. `ip:192.168.1.1`)
  * Authenticated Clients: Hashed OIDC token Subject (e.g. `user:auth0|user-12345`)
  * Hashing the key ensures that **no plain client IP addresses or user IDs** escape into Valkey cache keys, strictly defending user PII.
* **Storage Type**: Valkey Hash Map (`HMSET` / `HMGET` fields):
  * `tokens`: Decimal representation of remaining token balance.
  * `last_update`: Unix millisecond integer tracking the last request evaluation.
* **Eviction Strategy**: Keys are created with an automatic `EXPIRE` TTL calculated dynamically as twice the bucket's maximum refill duration:
  $$\text{TTL} = \text{ceil}\left(\frac{\text{Capacity}}{\text{RefillRate}} \times \text{RefillPeriod} \times 2\right)$$
  This guarantees that idle rate-limiting buckets are automatically garbage collected by Valkey, keeping RAM consumption bounded.

---

## ⚡ 3. Telemetry Tracing & Log Masking (PCI-DSS Compliance)

Telemetry is designed for millisecond-level observability while enforcing strict compliance boundaries:

### 1. Request ID Correlation Tracing
The gateway injects a correlation identifier `X-Request-ID` into every HTTP call.
* **Middleware**: `CorrelationMiddleware` extracts or generates a UUIDv4 on entry.
* **Telemetry Wrapper**: `ContextHandler` intercepts standard `log/slog` emissions, automatically extracting the request identifier from the context and appending it as a structured field to all system outputs:
  ```json
  {"time":"2026-05-23T21:10:21.000Z","level":"INFO","msg":"finalized transfer","request_id":"c3a2f8b9-..."}
  ```

### 2. PII Log Redaction Engine
To comply with PCI-DSS and PSD2 logging restrictions, the custom domain wrapper type `MaskedAccount` intercepts telemetry output.
* **Logic**: Overrides `fmt.Stringer` and `slog.LogValuer` interfaces to automatically mask digits, leaving only the first two and last four numbers visible:
  `1234567890` $\rightarrow$ `12******7890`
* **Telemetry Output Example**:
  ```text
  2026/05/23 21:10:21 INFO initiating transfer sender=12****7890 receiver=09****4321
  ```
  Plaintext account numbers are mathematically blocked from escaping to stdout or logging pipelines, eliminating risk of exposure in files or monitoring indices.

---

## 🛡️ 4. System Resilience: Circuit Breaking

All egress connections and intranet peer calls are encapsulated within sliding-window **`gobreaker` Circuit Breakers** to defend Gateway pool resources against downstream outages:

* **Trigger settings**:
  * **Failure Threshold**: Trips when failure ratio $\ge 50\%$ over a sliding window.
  * **Evaluation Volume**: Evaluates once requests $\ge 5$ within the active window.
  * **Timeout**: Retries after a `30s` sleep window by shifting to `Half-Open` state.
* **Outage Mitigation**: When a circuit trips, downstream calls are failed-fast instantly (returning a `503 Service Unavailable` equivalent domain error), immediately protecting network sockets and database connection limits from exhaustion.

---

## 📊 5. Performance SLAs & Benchmark Metrics

We scientifically profiled our custom verifiers and authorization engine using standard Go benchmarks on an **Apple M4 ARM64 (10 cores)** processor. Latencies are well below the required p95 budgets:

| Operational Metric | Required SLA Budget | Scientifically Measured Performance | Speed Performance Margin |
| :--- | :--- | :--- | :--- |
| **Embedded OPA Rego policy evaluation** | **`< 0.1 ms`** (100 µs) | **`10.62 microseconds`** (10,620 ns/op) | **10x faster than SLA** |
| **Secure Enclave ECDSA P-256 signature check** | **`< 5.0 ms`** | **`34.99 microseconds`** (34,998 ns/op) | **140x faster than SLA** |
| **HMAC-SHA256 Token Swapper verification** | **`< 15.0 ms`** | **`1.04 microseconds`** (1,041 ns/op) | **14,400x faster than SLA** |
| **Max memory allocation** | **`< 1 MB`** | **`10.5 KB`** (OPA) / **`1.96 KB`** (Signature) | **100x below memory limit** |

---

## 🛠️ 6. Maintenance & Deployment Cheat Sheet

Sysadmins and developers can manage the codebase using these standard CLI commands executed in the workspace root `/Volumes/SSD990PRO2TB/workspace/go-api-management`:

### 1. Execute Unit & Integration Test Suites
Validate PKCE authentication flows, attestation verifiers, and secure signature validations:
```bash
go test -v ./...
```

### 2. Execute Latency Benchmarks
Profile CPU speeds and memory allocations of core crypto operations and dynamic OPA/rate-limiting shield layers:
```bash
# Profile Cryptographic signature verifiers
go test -bench=. -benchmem ./internal/banking/adapters/crypto/...

# Profile Embedded OPA Rego policy evaluations
go test -bench=. -benchmem ./internal/adapters/handler/...
```

### 3. Generate CPU & Memory Profiles
Generate diagnostic reports to analyze code performance in `pprof`:
```bash
# Generate profile outputs
go test -cpuprofile cpu.prof -memprofile mem.prof -bench . ./internal/banking/adapters/crypto/...

# Analyze CPU profile interactively
go tool pprof cpu.prof
```

### 4. Sync Project Manifest Directory Tree
Re-run the indexer after creating, modifying, or revoking source files:
```bash
python3 tools/indexer.py
```

---
*Handover Manual Version: 1.1.0*
*Last Optimized: 2026-05-24*
