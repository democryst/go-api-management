# Performance Profiler: Latency & Benchmarking Guidelines

This document outlines the workflows and standards for profiling and benchmarking Go API components.

## ⚡ Performance SLAs
All APIs and system logic must stay within these strict performance budgets unless explicitly overridden:
* **p95 Latency**: < 50ms
* **p99 Latency**: < 100ms
* **Max Memory Allocation per standard operation**: < 1MB

---

## 🔬 Go Benchmarking Workflow

All critical domain paths, parsing functions, and algorithm implementations must include micro-benchmarks.

### 1. Writing Benchmarks
In file `xyz_test.go`:
```go
func BenchmarkProcessPayload(b *testing.B) {
    payload := generateTestPayload()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := ProcessPayload(payload)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### 2. Execution Commands
To run benchmarks and output memory allocations:
```bash
go test -bench=. -benchmem ./...
```

---

## 📊 CPU and Memory Profiling with pprof

For deep latency or memory leak analysis, use Go's built-in `pprof` toolchain.

### 1. Collecting Profiles
Add CPU/Memory profiling hooks or trigger via `net/http/pprof`:
```bash
# CPU Profile
go test -cpuprofile cpu.prof -memprofile mem.prof -bench .

# Interactive Analysis
go tool pprof cpu.prof
```

### 2. Analysis Checklist
1. **Allocate-Free Loops**: Minimize allocations in hot loops (use `sync.Pool` if necessary).
2. **Lock Contention**: Identify mutex blocks and replace with lock-free atomic primitives or map isolation where possible.
3. **Connection Pooling**: Always tune Idle Connection limit and Max Open limit for DBs and HTTP Clients.

---
*Last Updated: 2026-05-23*
