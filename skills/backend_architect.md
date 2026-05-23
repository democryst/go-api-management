# Backend Service Architect: "No-Magic" Go Standards

This document establishes the architectural standards for BackendServiceArchitect tasks.

## 🏛️ Core Architectural Principles

### 1. Explicit Dependency Injection
* **Rule**: No global state, packages-level singletons, or implicit container magic.
* **Practice**: All dependencies must be passed explicitly via constructor functions (e.g., `NewService(...)`) or method parameters.
* **Go Idiom**:
  ```go
  type Service struct {
      repo UserRepository // explicit port interface
      logger *slog.Logger
  }

  func NewService(repo UserRepository, logger *slog.Logger) *Service {
      return &Service{
          repo: repo,
          logger: logger,
      }
  }
  ```

### 2. Observability-First
* **Rule**: All execution paths must be traceable. Never swallow a `context.Context` or request ID.
* **Practice**: Always accept `context.Context` as the first argument in all method signatures across domain, ports, and adapters.
* **Go Idiom**:
  ```go
  func (s *Service) CreateUser(ctx context.Context, req *CreateUserDTO) (*User, error) {
      // Extract tracing context, log structured events
      s.logger.InfoContext(ctx, "creating user", slog.String("email", req.Email))
  }
  ```

### 3. Data Access Transparency
* **Rule**: No heavy ORM abstractions that hide queries or utilize lazy-loading.
* **Practice**: Use raw SQL or explicit builders (e.g., standard SQL with `database/sql` or `sqlx`).
* **Go Idiom**:
  ```go
  const createUserQuery = `INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)`
  ```

### 4. Complexity Determinism
* **Rule**: Every public function and business process must document its Big-O time and space complexity in the docstring.
* **Practice**:
  ```go
  // ValidateToken parses and validates a JWT token.
  // Time Complexity: O(1)
  // Space Complexity: O(1)
  func (s *Service) ValidateToken(ctx context.Context, token string) (*Claims, error) { ... }
  ```

### 5. Result Pattern
* **Rule**: No silent panics or implicit exception handling.
* **Practice**: Use explicit multi-value returns `(Result, error)` and wrapping for errors to preserve stack traces.
* **Go Idiom**:
  ```go
  user, err := s.repo.FindByID(ctx, id)
  if err != nil {
      return nil, fmt.Errorf("find user: %w", err)
  }
  ```

---
*Last Updated: 2026-05-23*
