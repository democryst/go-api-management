package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPolicyMiddleware_Scenarios(t *testing.T) {
	// Initialize context
	ctx := context.Background()

	// Locate policy file path
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	// Walk up to find policies directory in workspace root
	policyPath := filepath.Join(wd, "../../../policies/policy.rego")
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		// Fallback for tests run in workspace root
		policyPath = filepath.Join(wd, "policies/policy.rego")
	}

	evaluator, err := NewOPAPolicyEvaluator(ctx, policyPath)
	if err != nil {
		t.Fatalf("failed to initialize OPA evaluator: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(&strings.Builder{}, nil))
	middleware := NewPolicyMiddleware(evaluator, logger)

	// Mock downstream success handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Handler(nextHandler)

	// Helper to generate mock Bearer token header with given claims
	mockAuthHeader := func(sub string, scopes []string) string {
		claims := map[string]interface{}{
			"sub":   sub,
			"scope": strings.Join(scopes, " "),
		}
		claimsBytes, _ := json.Marshal(claims)
		encoded := base64.RawURLEncoding.EncodeToString(claimsBytes)
		return "Bearer header." + encoded + ".signature"
	}

	tests := []struct {
		name           string
		method         string
		path           string
		authHeader     string
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name:           "Public path allowed (no token)",
			method:         "POST",
			path:           "/auth/register-device",
			authHeader:     "",
			body:           nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Edge transfer initiation allowed (amount < $10k)",
			method:         "POST",
			path:           "/transfer/initiate",
			authHeader:     mockAuthHeader("auth0|user1", []string{"transfer:write"}),
			body:           map[string]interface{}{"amount_cents": 500000}, // $5,000
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Edge transfer initiation rejected (amount >= $10k, missing scope)",
			method:         "POST",
			path:           "/transfer/initiate",
			authHeader:     mockAuthHeader("auth0|user1", []string{"transfer:write"}),
			body:           map[string]interface{}{"amount_cents": 1500000}, // $15,000
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Edge transfer initiation allowed (amount >= $10k, with high_value scope)",
			method:         "POST",
			path:           "/transfer/initiate",
			authHeader:     mockAuthHeader("auth0|user1", []string{"transfer:write", "transfer:high_value"}),
			body:           map[string]interface{}{"amount_cents": 1500000}, // $15,000
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Intranet execute transfer allowed (with transfer:execute scope)",
			method:         "POST",
			path:           "/private/transfers",
			authHeader:     mockAuthHeader("auth0|user1", []string{"transfer:execute"}),
			body:           map[string]interface{}{"amount_cents": 250000},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Intranet execute transfer rejected (missing transfer:execute scope)",
			method:         "POST",
			path:           "/private/transfers",
			authHeader:     mockAuthHeader("auth0|user1", []string{"transfer:read"}),
			body:           map[string]interface{}{"amount_cents": 250000},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var reqBody io.Reader
			if tc.body != nil {
				b, _ := json.Marshal(tc.body)
				reqBody = bytes.NewBuffer(b)
			}
			req := httptest.NewRequest(tc.method, tc.path, reqBody)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tc.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// BenchmarkRegoPolicyEvaluation profiles in-memory policy evaluation speeds.
// Expectation: Time Complexity O(1), execution duration < 100 microseconds (< 0.1ms SLA).
func BenchmarkRegoPolicyEvaluation(b *testing.B) {
	ctx := context.Background()
	wd, _ := os.Getwd()
	policyPath := filepath.Join(wd, "../../../policies/policy.rego")
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		policyPath = filepath.Join(wd, "policies/policy.rego")
	}

	evaluator, err := NewOPAPolicyEvaluator(ctx, policyPath)
	if err != nil {
		b.Fatalf("failed to initialize OPA evaluator: %v", err)
	}

	input := map[string]interface{}{
		"path":   "/transfer/initiate",
		"method": "POST",
		"claims": map[string]interface{}{
			"sub":    "auth0|user-999",
			"scopes": []interface{}{"transfer:write"},
		},
		"body": map[string]interface{}{
			"amount_cents": 500000,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := evaluator.Evaluate(ctx, input)
		if err != nil {
			b.Fatalf("evaluation failed: %v", err)
		}
	}
}
