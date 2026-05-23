# Project Manifest: go-api-management

Generated at: `2026-05-23 21:37:11`

## 🏛️ Directory Tree Structure

```text
go-api-management/
├── api/
│   └── openapi.yaml
├── documents/
│   ├── developer_guide.md
│   ├── end_user_documentation.md
│   └── handover_document.md
├── internal/
│   ├── adapters/
│   │   ├── crypto/
│   │   │   ├── pkce.go
│   │   │   └── pkce_test.go
│   │   ├── gateway/
│   │   │   └── auth0_gateway.go
│   │   ├── handler/
│   │   │   ├── auth_handler.go
│   │   │   ├── middleware.go
│   │   │   ├── policy_middleware.go
│   │   │   ├── policy_middleware_test.go
│   │   │   ├── rate_limiter.go
│   │   │   ├── rate_limiter_test.go
│   │   │   └── schema_middleware.go
│   │   ├── repository/
│   │   │   └── sql_session_repository.go
│   │   └── telemetry/
│   │       ├── otel_tracer.go
│   │       └── prometheus_metrics.go
│   ├── banking/
│   │   ├── adapters/
│   │   │   ├── crypto/
│   │   │   │   ├── crypto_benchmark_test.go
│   │   │   │   ├── jwt.go
│   │   │   │   ├── signature.go
│   │   │   │   └── signature_test.go
│   │   │   └── gateway/
│   │   │       └── attestation_verifier.go
│   │   ├── domain/
│   │   │   └── banking.go
│   │   ├── handler/
│   │   │   ├── banking_handler.go
│   │   │   └── banking_handler_test.go
│   │   ├── ports/
│   │   │   └── banking_ports.go
│   │   ├── repository/
│   │   │   └── sql_key_repository.go
│   │   └── services/
│   │       └── banking_service.go
│   └── core/
│       ├── domain/
│       │   ├── errors.go
│       │   ├── session.go
│       │   ├── token.go
│       │   └── user.go
│       ├── ports/
│       │   ├── auth_provider.go
│       │   └── session_repo.go
│       └── services/
│           ├── auth_service.go
│           └── auth_service_test.go
├── migrations/
│   └── schema.sql
├── plan/
│   └── roadmap.md
├── policies/
│   └── policy.rego
├── skills/
│   ├── backend_architect.md
│   ├── log_investigator.md
│   ├── obsidian_manager.md
│   └── perf_profiler.md
├── spec/
│   └── requirements.md
├── tech/
│   ├── adr-001-session-management.md
│   ├── adr-002-jwks-caching.md
│   └── design.md
├── tools/
│   └── indexer.py
├── AI_SYSTEM_MANIFEST.md
├── go.mod
└── go.sum
```

## 📂 Cataloged Repository Files

| File Name | Rel Path | Size (Bytes) | Last Modified | Architectural Purpose |
| :--- | :--- | :--- | :--- | :--- |
| `AI_SYSTEM_MANIFEST.md` | [`AI_SYSTEM_MANIFEST.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/AI_SYSTEM_MANIFEST.md) | 2449 | 2026-05-23 19:53:51 | Repository Core Metadata |
| `go.mod` | [`go.mod`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/go.mod) | 5885 | 2026-05-23 21:33:53 | Repository Core Metadata |
| `go.sum` | [`go.sum`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/go.sum) | 28883 | 2026-05-23 21:33:53 | Repository Core Metadata |
| `schema.sql` | [`migrations/schema.sql`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/migrations/schema.sql) | 1205 | 2026-05-23 21:00:30 | Repository Core Metadata |
| `indexer.py` | [`tools/indexer.py`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/tools/indexer.py) | 4275 | 2026-05-23 20:02:23 | Developer Tooling / Script |
| `roadmap.md` | [`plan/roadmap.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/plan/roadmap.md) | 5770 | 2026-05-23 20:56:39 | Repository Core Metadata |
| `requirements.md` | [`spec/requirements.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/spec/requirements.md) | 6119 | 2026-05-23 20:56:31 | Repository Core Metadata |
| `auth_provider.go` | [`internal/core/ports/auth_provider.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/core/ports/auth_provider.go) | 1072 | 2026-05-23 20:19:07 | Go Source Code / Business Logic |
| `session_repo.go` | [`internal/core/ports/session_repo.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/core/ports/session_repo.go) | 842 | 2026-05-23 20:19:12 | Go Source Code / Business Logic |
| `errors.go` | [`internal/core/domain/errors.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/core/domain/errors.go) | 988 | 2026-05-23 20:18:45 | Go Source Code / Business Logic |
| `session.go` | [`internal/core/domain/session.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/core/domain/session.go) | 741 | 2026-05-23 20:18:50 | Go Source Code / Business Logic |
| `token.go` | [`internal/core/domain/token.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/core/domain/token.go) | 308 | 2026-05-23 20:18:55 | Go Source Code / Business Logic |
| `user.go` | [`internal/core/domain/user.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/core/domain/user.go) | 2061 | 2026-05-23 20:19:02 | Go Source Code / Business Logic |
| `auth_service.go` | [`internal/core/services/auth_service.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/core/services/auth_service.go) | 5133 | 2026-05-23 20:46:38 | Go Source Code / Business Logic |
| `auth_service_test.go` | [`internal/core/services/auth_service_test.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/core/services/auth_service_test.go) | 6236 | 2026-05-23 20:49:56 | Go Source Code / Business Logic |
| `auth_handler.go` | [`internal/adapters/handler/auth_handler.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/adapters/handler/auth_handler.go) | 7921 | 2026-05-23 21:30:13 | Go Source Code / Business Logic |
| `middleware.go` | [`internal/adapters/handler/middleware.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/adapters/handler/middleware.go) | 2990 | 2026-05-23 20:47:36 | Go Source Code / Business Logic |
| `policy_middleware.go` | [`internal/adapters/handler/policy_middleware.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/adapters/handler/policy_middleware.go) | 6525 | 2026-05-23 21:36:14 | Go Source Code / Business Logic |
| `policy_middleware_test.go` | [`internal/adapters/handler/policy_middleware_test.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/adapters/handler/policy_middleware_test.go) | 5274 | 2026-05-23 21:36:55 | Go Source Code / Business Logic |
| `rate_limiter.go` | [`internal/adapters/handler/rate_limiter.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/adapters/handler/rate_limiter.go) | 6018 | 2026-05-23 21:35:24 | Go Source Code / Business Logic |
| `rate_limiter_test.go` | [`internal/adapters/handler/rate_limiter_test.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/adapters/handler/rate_limiter_test.go) | 6557 | 2026-05-23 21:35:07 | Go Source Code / Business Logic |
| `schema_middleware.go` | [`internal/adapters/handler/schema_middleware.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/adapters/handler/schema_middleware.go) | 5439 | 2026-05-23 21:28:45 | Go Source Code / Business Logic |
| `pkce.go` | [`internal/adapters/crypto/pkce.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/adapters/crypto/pkce.go) | 1324 | 2026-05-23 20:29:17 | Go Source Code / Business Logic |
| `pkce_test.go` | [`internal/adapters/crypto/pkce_test.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/adapters/crypto/pkce_test.go) | 1541 | 2026-05-23 20:50:12 | Go Source Code / Business Logic |
| `sql_session_repository.go` | [`internal/adapters/repository/sql_session_repository.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/adapters/repository/sql_session_repository.go) | 4086 | 2026-05-23 20:46:31 | Go Source Code / Business Logic |
| `otel_tracer.go` | [`internal/adapters/telemetry/otel_tracer.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/adapters/telemetry/otel_tracer.go) | 2180 | 2026-05-23 21:29:29 | Go Source Code / Business Logic |
| `prometheus_metrics.go` | [`internal/adapters/telemetry/prometheus_metrics.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/adapters/telemetry/prometheus_metrics.go) | 2686 | 2026-05-23 21:29:37 | Go Source Code / Business Logic |
| `auth0_gateway.go` | [`internal/adapters/gateway/auth0_gateway.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/adapters/gateway/auth0_gateway.go) | 8902 | 2026-05-23 20:29:27 | Go Source Code / Business Logic |
| `banking_handler.go` | [`internal/banking/handler/banking_handler.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/handler/banking_handler.go) | 7032 | 2026-05-23 21:36:25 | Go Source Code / Business Logic |
| `banking_handler_test.go` | [`internal/banking/handler/banking_handler_test.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/handler/banking_handler_test.go) | 11162 | 2026-05-23 21:31:33 | Go Source Code / Business Logic |
| `sql_key_repository.go` | [`internal/banking/repository/sql_key_repository.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/repository/sql_key_repository.go) | 3751 | 2026-05-23 21:05:49 | Go Source Code / Business Logic |
| `crypto_benchmark_test.go` | [`internal/banking/adapters/crypto/crypto_benchmark_test.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/adapters/crypto/crypto_benchmark_test.go) | 1785 | 2026-05-23 21:11:54 | Go Source Code / Business Logic |
| `jwt.go` | [`internal/banking/adapters/crypto/jwt.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/adapters/crypto/jwt.go) | 2712 | 2026-05-23 21:05:56 | Go Source Code / Business Logic |
| `signature.go` | [`internal/banking/adapters/crypto/signature.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/adapters/crypto/signature.go) | 3470 | 2026-05-23 21:04:34 | Go Source Code / Business Logic |
| `signature_test.go` | [`internal/banking/adapters/crypto/signature_test.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/adapters/crypto/signature_test.go) | 5991 | 2026-05-23 21:04:23 | Go Source Code / Business Logic |
| `attestation_verifier.go` | [`internal/banking/adapters/gateway/attestation_verifier.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/adapters/gateway/attestation_verifier.go) | 4127 | 2026-05-23 21:04:09 | Go Source Code / Business Logic |
| `banking_ports.go` | [`internal/banking/ports/banking_ports.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/ports/banking_ports.go) | 1254 | 2026-05-23 21:00:23 | Go Source Code / Business Logic |
| `banking.go` | [`internal/banking/domain/banking.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/domain/banking.go) | 1933 | 2026-05-23 21:00:18 | Go Source Code / Business Logic |
| `banking_service.go` | [`internal/banking/services/banking_service.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/services/banking_service.go) | 7978 | 2026-05-23 21:06:04 | Go Source Code / Business Logic |
| `policy.rego` | [`policies/policy.rego`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/policies/policy.rego) | 1108 | 2026-05-23 21:35:53 | Repository Core Metadata |
| `openapi.yaml` | [`api/openapi.yaml`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/api/openapi.yaml) | 5422 | 2026-05-23 21:28:39 | Repository Core Metadata |
| `backend_architect.md` | [`skills/backend_architect.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/skills/backend_architect.md) | 2353 | 2026-05-23 20:02:03 | Operational Skill Guideline / Blueprint |
| `log_investigator.md` | [`skills/log_investigator.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/skills/log_investigator.md) | 1982 | 2026-05-23 20:02:12 | Operational Skill Guideline / Blueprint |
| `obsidian_manager.md` | [`skills/obsidian_manager.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/skills/obsidian_manager.md) | 1814 | 2026-05-23 20:02:17 | Operational Skill Guideline / Blueprint |
| `perf_profiler.md` | [`skills/perf_profiler.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/skills/perf_profiler.md) | 1708 | 2026-05-23 20:02:07 | Operational Skill Guideline / Blueprint |
| `developer_guide.md` | [`documents/developer_guide.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/documents/developer_guide.md) | 11773 | 2026-05-23 21:18:36 | Repository Core Metadata |
| `end_user_documentation.md` | [`documents/end_user_documentation.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/documents/end_user_documentation.md) | 12593 | 2026-05-23 21:16:45 | Repository Core Metadata |
| `handover_document.md` | [`documents/handover_document.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/documents/handover_document.md) | 8271 | 2026-05-23 21:16:52 | Repository Core Metadata |
| `adr-001-session-management.md` | [`tech/adr-001-session-management.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/tech/adr-001-session-management.md) | 3745 | 2026-05-23 20:14:47 | Repository Core Metadata |
| `adr-002-jwks-caching.md` | [`tech/adr-002-jwks-caching.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/tech/adr-002-jwks-caching.md) | 2529 | 2026-05-23 20:14:54 | Repository Core Metadata |
| `design.md` | [`tech/design.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/tech/design.md) | 6686 | 2026-05-23 20:15:01 | Repository Core Metadata |
