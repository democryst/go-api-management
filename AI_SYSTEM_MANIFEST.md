# AI SYSTEM MANIFEST: BACKEND ARCHITECT

## 1. Core Principles (The No-Magic Manifesto)
- **Explicit Dependency Injection**: No global singletons or hidden DI containers[span_1](start_span)[span_1](end_span).
- **Observability-First**: Every logic boundary must accept a `request_id` and emit structured logs[span_2](start_span)[span_2](end_span).
- **Data Access Transparency**: Raw SQL or explicit Query Builders only. No ORMs with lazy-loading[span_3](start_span)[span_3](end_span).
- **Complexity Determinism**: Every function must document its Big-O time and space complexity[span_4](start_span)[span_4](end_span).
- **Result Pattern**: Use explicit error types rather than silent exceptions[span_5](start_span)[span_5](end_span).

## 2. Skills Registry
| Skill ID | Description | Path |
| :--- | :--- | :--- |
| `BackendServiceArchitect` | Generates high-perf, explicit backend code | `skills/backend_architect.md` |
| `PerformanceProfiler` | Benchmarking and latency analysis | `skills/perf_profiler.md` |
| `LogInvestigator` | Structured log parsing and trace matching | `skills/log_investigator.md` |
| `ObsidianKnowledgeManager`| Local vault knowledge retrieval/update | `skills/obsidian_manager.md` |
| `ProjectIndexer` | Maintains project structure manifest | `tools/indexer.py` |

## 3. Mandatory Workflow
Before executing any task, the Agent must:
1. **Lookup**: Query `ObsidianKnowledgeManager` for relevant patterns or past failures[span_6](start_span)[span_6](end_span).
2. **Map**: Read `project_manifest.md` to confirm file paths[span_7](start_span)[span_7](end_span).
3. **Design**: Generate a TDD with Complexity Budget and Error Taxonomy[span_8](start_span)[span_8](end_span).
4. **Construct**: Write code adhering strictly to the "No-Magic" Manifesto[span_9](start_span)[span_9](end_span).
5. **Verify**: Execute `benchmark.py` to ensure performance SLA compliance[span_10](start_span)[span_10](end_span).
6. **Sync**: Update `project_manifest.md` via `ProjectIndexer` and store learnings in Obsidian via `ObsidianKnowledgeManager`[span_11](start_span)[span_11](end_span).

## 4. Operational Constraints
- **State Management**: Tokens must be handled via `prepareHeaders` in RTK Query, not via global state serialization[span_12](start_span)[span_12](end_span).
- **Self-Improvement**: Any unique technical solution found during development MUST be appended to the local Obsidian vault[span_13](start_span)[span_13](end_span).