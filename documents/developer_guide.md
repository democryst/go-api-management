# Secure Banking Gateway: Developer Extension Guide
## How to Implement New APIs & Endpoints (Hexagonal Architecture)

This guide provides step-by-step instructions and coding blueprints for backend developers to implement new APIs inside our isolated `internal/banking/` tree. 

As a concrete example, we will walk through the implementation of an **Account Inquiry API** consisting of two endpoints:
1. `POST /transfer/accounts` (Edge public request) $\rightarrow$ Swaps token to `scopes: ["accounts:read"]` $\rightarrow$ Forwards to intranet.
2. `POST /private/accounts` (Private intranet request) $\rightarrow$ Validates JWT claims, queries database/core registry, and returns details with PII masking.

---

## 🏛️ Hexagonal Layer Mapping

When adding a new API, you must implement it sequentially across our clean architectural layers:

```text
 [ Inbound Adapters ]              [ Ports Boundary ]              [ Core Domain ]              [ Outbound Adapters ]
  banking_handler.go  ──► calls ──► banking_ports.go  ──► calls ──► banking_service.go ──► calls ──► sql_key_repository.go
```

1. **`domain/`**: Define request/response models and log redactions.
2. **`ports/`**: Define interfaces for outbound persistence or third-party client boundaries.
3. **`services/`**: Extend `BankingService` to orchestrate business validation, BFF token swaps, circuit breakers, and verification.
4. **`repository/` or `adapters/`**: Implement persistence adapters or mock intranet clients.
5. **`handler/`**: Configure routes on the public edge and private intranet, parse JSON, check scopes, and return payloads.
6. **`tests`**: Write unit/integration tests asserting security, tampering rejections, and log redactions.

---

## 📂 Step 1: Define the Domain Model (`domain/`)

Define request and response payloads in [`internal/banking/domain/banking.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/domain/banking.go). Ensure that any account number properties utilize the custom `MaskedAccount` type to guarantee PCI-DSS compliance in debug log files:

```go
// AccountDetails represents a user's bank account info with redacted logging.
type AccountDetails struct {
	AccountNumber MaskedAccount `db:"account_number" json:"account_number"`
	AccountName   string        `db:"account_name" json:"account_name"`
	BalanceCents  int64         `db:"balance_cents" json:"balance_cents"`
	Currency      string        `db:"currency" json:"currency"`
	AccountType   string        `db:"account_type" json:"account_type"` // e.g. SAVINGS, CHECKING
}

// AccountInquiryResponse contains the user's active bank accounts list.
type AccountInquiryResponse struct {
	UserID   string           `json:"user_id"`
	Accounts []AccountDetails `json:"accounts"`
}
```

---

## 🔌 Step 2: Declare Outbound Ports (`ports/`)

If your new API needs to query the database or connect to third-party bank ledger APIs, declare the outbound port interface in [`internal/banking/ports/banking_ports.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/ports/banking_ports.go). 

All methods across ports and services **MUST** accept `context.Context` as their first argument and explicitly document their Big-O time and space complexity:

```go
// AccountRepository defines the outbound port boundary for querying user accounts.
type AccountRepository interface {
	// GetAccountsByUserID retrieves all active accounts assigned to a user profile.
	// Time Complexity: O(A) where A is the number of active accounts.
	// Space Complexity: O(A)
	GetAccountsByUserID(ctx context.Context, userID string) ([]domain.AccountDetails, error)
}
```

---

## ⚙️ Step 3: Implement Outbound Adapters (`repository/`)

Implement the `ports.AccountRepository` interface in `internal/banking/repository/sql_account_repository.go` using ORM-free SQLX prepared queries:

```go
package repository

import (
	"context"
	"fmt"
	"log/slog"
	"github.com/democryst/go-api-management/internal/banking/domain"
	"github.com/jmoiron/sqlx"
)

type SQLAccountRepository struct {
	db     *sqlx.DB
	logger *slog.Logger
}

func NewSQLAccountRepository(db *sqlx.DB, logger *slog.Logger) *SQLAccountRepository {
	return &SQLAccountRepository{db: db, logger: logger}
}

// Time Complexity: O(A) (DB query bound)
// Space Complexity: O(A)
func (r *SQLAccountRepository) GetAccountsByUserID(ctx context.Context, userID string) ([]domain.AccountDetails, error) {
	const query = `
		SELECT account_number, account_name, balance_cents, currency, account_type
		FROM bank_accounts
		WHERE user_id = $1 AND is_active = true
	`
	r.logger.InfoContext(ctx, "fetching bank accounts from PostgreSQL", slog.String("user_id", userID))

	var accounts []domain.AccountDetails
	err := r.db.SelectContext(ctx, &accounts, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query bank accounts: %w", err)
	}
	return accounts, nil
}
```

---

## 🧠 Step 4: Extend Core Orchestration (`services/`)

Extend [`internal/banking/services/banking_service.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/services/banking_service.go). You will implement two separate service functions representing the two network zones:

### 1. The Edge Zone: BFF Token Swapper
Generates a short-lived Internal JWT containing the custom scope `scopes: ["accounts:read"]` and forwards the request over secure intranet peer lines wrapped in `gobreaker` circuit breakers.

```go
// InquiryAccounts acts as the Edge BFF Swapper. Swaps token to accounts scope, and forwards to private VPC.
// Time Complexity: O(1) (network bound)
// Space Complexity: O(1)
func (s *BankingService) InquiryAccounts(ctx context.Context, userID string) (*domain.AccountInquiryResponse, error) {
	s.logger.InfoContext(ctx, "initiating accounts inquiry edge request", slog.String("user_id", userID))

	// 1. Perform Token Swap: Generate short-lived scoped Internal JWT
	internalJWT, err := crypto.GenerateInternalJWT(userID, []string{"accounts:read"}, s.sharedSecret, 1*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("token swap failed: %w", err)
	}

	var response domain.AccountInquiryResponse

	// 2. Forward to Intranet HTTP Endpoint through gobreaker sliding-window circuit breaker
	_, err = s.cb.Execute(func() (interface{}, error) {
		req, err := http.NewRequestWithContext(ctx, "POST", s.privateURL+"/private/accounts", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+internalJWT)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("intranet accounts inquiry rejected: status %d", resp.StatusCode)
		}

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, err
		}
		return &response, nil
	})

	if err != nil {
		return nil, fmt.Errorf("accounts lookup failed: %w", err)
	}

	return &response, nil
}
```

### 2. The Intranet Zone: Ledger Reader
Executes inside the Secure Intranet VPC. Verifies the high-privilege JWT, asserts `accounts:read` scope, and queries the database repository.

```go
// ExecuteIntranetInquiry processes private intranet-bound account lookup requests.
// Time Complexity: O(A) where A is the number of active accounts.
// Space Complexity: O(A)
func (s *BankingService) ExecuteIntranetInquiry(ctx context.Context, internalToken string) (*domain.AccountInquiryResponse, error) {
	s.logger.InfoContext(ctx, "executing private VPC intranet account lookup")

	// 1. Validate Internal JWT
	claims, err := crypto.VerifyInternalJWT(internalToken, s.sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("unauthorized internal token: %w", err)
	}

	// 2. Validate custom accounts scope presence
	hasScope := false
	for _, sc := range claims.Scopes {
		if sc == "accounts:read" {
			hasScope = true
			break
		}
	}
	if !hasScope {
		return nil, errors.New("unauthorized: missing required 'accounts:read' scope")
	}

	// 3. Query PostgreSQL persistent repository
	accounts, err := s.accountRepo.GetAccountsByUserID(ctx, claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("get accounts from repository: %w", err)
	}

	return &domain.AccountInquiryResponse{
		UserID:   claims.Subject,
		Accounts: accounts,
	}, nil
}
```

---

## 🔌 Step 5: Wire Routing and Controllers (`handler/`)

Configure routes on both the public Edge router and the private Intranet router inside [`internal/banking/handler/banking_handler.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/handler/banking_handler.go):

```go
// 1. In EdgeRoutes() wire the public endpoint protected by OIDC JWT Middleware
r.Post("/transfer/accounts", h.InquiryAccounts)

// 2. In PrivateRoutes() wire the private intranet endpoint
r.Post("/private/accounts", h.ExecutePrivateInquiry)
```

Now implement the HTTP controller functions:

```go
// InquiryAccounts handles POST /transfer/accounts on the Edge Gateway.
// Time Complexity: O(1) (network bound)
// Space Complexity: O(1)
func (h *BankingHandler) InquiryAccounts(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractSubject(r)
	if err != nil {
		h.writeError(r.Context(), w, http.StatusUnauthorized, err.Error())
		return
	}

	resp, err := h.service.InquiryAccounts(r.Context(), userID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "accounts inquiry failed at Edge", slog.Any("error", err))
		h.writeError(r.Context(), w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// ExecutePrivateInquiry handles POST /private/accounts in the Secure Intranet VPC.
// Time Complexity: O(A) where A is the number of user accounts.
// Space Complexity: O(A)
func (h *BankingHandler) ExecutePrivateInquiry(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		h.writeError(r.Context(), w, http.StatusUnauthorized, "Missing or invalid Authorization header")
		return
	}
	internalToken := strings.TrimPrefix(authHeader, "Bearer ")

	resp, err := h.service.ExecuteIntranetInquiry(r.Context(), internalToken)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "intranet accounts inquiry rejected", slog.Any("error", err))
		h.writeError(r.Context(), w, http.StatusUnauthorized, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
```

---

## 🧪 Step 6: Write Unit and Integration Tests

To verify correctness, add unit tests in [`internal/banking/handler/banking_handler_test.go`](file:///Volumes/SSD990PRO2TB/workspace/go-api-management/internal/banking/handler/banking_handler_test.go) asserting Edge-to-Intranet token swaps, JWT signature checks, and dynamic log redaction:

```go
func TestBankingHandler_InquiryAccounts_Success(t *testing.T) {
	repo := NewMockKeyRepository()
	// Mock account repository inside tests
	// Verify that the Edge Handler correctly fetches bank account lists, 
	// performs the OIDC claims context extraction, swaps tokens, and logs PII redacted entries.
}
```

### Run Tests and Verify Compliance
Run your test suites to ensure 100% compilation and validation success:
```bash
go test -v ./...
```

Verify that system logs automatically mask the bank accounts (e.g. `sender=12****7890`) when printing the account inquiry outputs.

---

## 🛡️ Step 7: Configure the Enterprise Shield Layers (OpenAPI, Valkey, OPA)

To ensure that your newly created endpoints are securely validated, rate-limited, and authorized, you must extend the configurations in our gateway shield middlewares:

### 1. Declare OpenAPI Schema Rules (`api/openapi.yaml`)
Register the endpoint specifications and payload constraints inside our OpenAPI spec file. This enables the high-performance `SchemaValidationMiddleware` to automatically validate parameters:
```yaml
paths:
  /transfer/accounts:
    post:
      summary: Retrieve authenticated user's active accounts list
      security:
        - BearerAuth: []
      responses:
        '200':
          description: Successful accounts retrieval
  /private/accounts:
    post:
      summary: Executed inside the intranet zone to fetch masked database details
      security:
        - InternalTokenAuth: []
      responses:
        '200':
          description: Database accounts list returned
```

### 2. Configure Distributed Valkey Rate Limiting
The rate limiter automatically hashes user OIDC Subjects (`user:<sub-id>`) or unauthenticated network IPs (`ip:<remote-addr>`) using SHA-256 before compiling Valkey keys. This completely mitigates PII leak risk in Redis/Valkey cache stores:
* If you need custom bucket settings, update `RateLimiterConfig` inside `main.go`. The default configuration uses a robust sliding-window bucket.

### 3. Configure Dynamic OPA Rego Policies (`policies/policy.rego`)
Write custom authorization rules inside our Rego policy file. The background file watcher daemon utilizes `fsnotify` to instantly hot-reload and compile these changes into the embedded evaluator with **zero server downtime**:
```rego
# Permitted edge accounts lookup for authenticated subjects
allow {
    input.path == "/transfer/accounts"
    input.method == "POST"
    input.claims.sub != ""
}

# Permitted private intranet lookup requiring custom read scope
allow {
    input.path == "/private/accounts"
    input.method == "POST"
    has_scope(input.claims.scopes, "accounts:read")
}
```

---

## 🔄 Step 8: Synchronize Workspace Project Manifest
Always run the repository indexer tool to update catalogs in `project_manifest.md` before compiling:
```bash
python3 tools/indexer.py
```

---
*Extension Guide Version: 1.1.0*
*Last Optimized: 2026-05-24*

