# Log Investigator: Structured Logging & Tracing Guidelines

This document outlines the standard for request tracking, context logging, and trace analysis.

## 🪵 Structured Logging Rules
* **Format**: All logs in production must be structured (JSON format).
* **Library**: Use the native `log/slog` library (Go 1.21+).
* **No Unstructured Logs**: Avoid `log.Printf` or fmt.Println.
* **Fields**: Always include domain-specific context (e.g., `user_id`, `request_id`, `resource_type`).

---

## 🔗 Trace Correlation & Context Propagation

Every incoming request must be assigned a unique, immutable request identifier (`request_id`). This ID must propagate down to all subsystems, databases, and outgoing HTTP calls.

### 1. HTTP Middleware Context Injection
```go
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = generateUUID()
        }

        // Store requestID in Context
        ctx := context.WithValue(r.Context(), "request_id", requestID)
        r = r.WithContext(ctx)

        // Inject requestID into Response Headers
        w.Header().Set("X-Request-ID", requestID)

        next.ServeHTTP(w, r)
    })
}
```

### 2. Context-Aware Logger Logging
Always pass context to the logger to ensure trace correlation is emitted in JSON outputs:
```go
slog.InfoContext(ctx, "database transaction committed", slog.String("db_operation", "insert_user"))
```

---

## 🕵️ Troubleshooting & Log Analysis Checklist
When debugging failures:
1. **Find by request_id**: Extract the full sequence of actions across systems using `request_id`.
2. **Identify Boundary Failures**: Check transition points (e.g., Handlers to Services, Repository responses).
3. **Analyze Stack Traces**: Ensure that unexpected errors print detailed nested error messages via `%w`.

---
*Last Updated: 2026-05-23*
