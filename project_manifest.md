# Project Manifest: go-api-management

Generated at: `2026-05-23 20:42:19`

## 🏛️ Directory Tree Structure

```text
go-api-management/
├── internal/
│   ├── adapters/
│   │   ├── crypto/
│   │   │   └── pkce.go
│   │   └── gateway/
│   │       └── auth0_gateway.go
│   └── core/
│       ├── domain/
│       │   ├── errors.go
│       │   ├── session.go
│       │   ├── token.go
│       │   └── user.go
│       └── ports/
│           ├── auth_provider.go
│           └── session_repo.go
├── migrations/
│   └── schema.sql
├── plan/
│   └── roadmap.md
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
| `go.mod` | [`go.mod`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/go.mod) | 99 | 2026-05-23 20:41:44 | Repository Core Metadata |
| `go.sum` | [`go.sum`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/go.sum) | 777 | 2026-05-23 20:41:44 | Repository Core Metadata |
| `schema.sql` | [`migrations/schema.sql`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/migrations/schema.sql) | 623 | 2026-05-23 20:19:17 | Repository Core Metadata |
| `indexer.py` | [`tools/indexer.py`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/tools/indexer.py) | 4275 | 2026-05-23 20:02:23 | Developer Tooling / Script |
| `roadmap.md` | [`plan/roadmap.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/plan/roadmap.md) | 5157 | 2026-05-23 20:11:21 | Repository Core Metadata |
| `requirements.md` | [`spec/requirements.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/spec/requirements.md) | 5174 | 2026-05-23 20:07:33 | Repository Core Metadata |
| `auth_provider.go` | [`internal/core/ports/auth_provider.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/core/ports/auth_provider.go) | 1072 | 2026-05-23 20:19:07 | Go Source Code / Business Logic |
| `session_repo.go` | [`internal/core/ports/session_repo.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/core/ports/session_repo.go) | 842 | 2026-05-23 20:19:12 | Go Source Code / Business Logic |
| `errors.go` | [`internal/core/domain/errors.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/core/domain/errors.go) | 988 | 2026-05-23 20:18:45 | Go Source Code / Business Logic |
| `session.go` | [`internal/core/domain/session.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/core/domain/session.go) | 741 | 2026-05-23 20:18:50 | Go Source Code / Business Logic |
| `token.go` | [`internal/core/domain/token.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/core/domain/token.go) | 308 | 2026-05-23 20:18:55 | Go Source Code / Business Logic |
| `user.go` | [`internal/core/domain/user.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/core/domain/user.go) | 2061 | 2026-05-23 20:19:02 | Go Source Code / Business Logic |
| `pkce.go` | [`internal/adapters/crypto/pkce.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/adapters/crypto/pkce.go) | 1324 | 2026-05-23 20:29:17 | Go Source Code / Business Logic |
| `auth0_gateway.go` | [`internal/adapters/gateway/auth0_gateway.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/adapters/gateway/auth0_gateway.go) | 8902 | 2026-05-23 20:29:27 | Go Source Code / Business Logic |
| `backend_architect.md` | [`skills/backend_architect.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/skills/backend_architect.md) | 2353 | 2026-05-23 20:02:03 | Operational Skill Guideline / Blueprint |
| `log_investigator.md` | [`skills/log_investigator.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/skills/log_investigator.md) | 1982 | 2026-05-23 20:02:12 | Operational Skill Guideline / Blueprint |
| `obsidian_manager.md` | [`skills/obsidian_manager.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/skills/obsidian_manager.md) | 1814 | 2026-05-23 20:02:17 | Operational Skill Guideline / Blueprint |
| `perf_profiler.md` | [`skills/perf_profiler.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/skills/perf_profiler.md) | 1708 | 2026-05-23 20:02:07 | Operational Skill Guideline / Blueprint |
| `adr-001-session-management.md` | [`tech/adr-001-session-management.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/tech/adr-001-session-management.md) | 3745 | 2026-05-23 20:14:47 | Repository Core Metadata |
| `adr-002-jwks-caching.md` | [`tech/adr-002-jwks-caching.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/tech/adr-002-jwks-caching.md) | 2529 | 2026-05-23 20:14:54 | Repository Core Metadata |
| `design.md` | [`tech/design.md`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/tech/design.md) | 6686 | 2026-05-23 20:15:01 | Repository Core Metadata |
