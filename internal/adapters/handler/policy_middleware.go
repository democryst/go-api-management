package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/open-policy-agent/opa/rego"
)

// OPAPolicyEvaluator compiles and evaluates Rego policies using the embedded OPA runtime.
type OPAPolicyEvaluator struct {
	mu         sync.RWMutex
	regoQuery  rego.PreparedEvalQuery
	policyPath string
}

// NewOPAPolicyEvaluator compiles the Rego policy at startup.
func NewOPAPolicyEvaluator(ctx context.Context, policyPath string) (*OPAPolicyEvaluator, error) {
	evaluator := &OPAPolicyEvaluator{policyPath: policyPath}
	if err := evaluator.reload(ctx); err != nil {
		return nil, err
	}
	return evaluator, nil
}

func (e *OPAPolicyEvaluator) reload(ctx context.Context) error {
	content, err := os.ReadFile(e.policyPath)
	if err != nil {
		return err
	}

	r := rego.New(
		rego.Query("data.gateway.authz.allow"),
		rego.Module(e.policyPath, string(content)),
	)

	query, err := r.PrepareForEval(ctx)
	if err != nil {
		return err
	}

	e.mu.Lock()
	e.regoQuery = query
	e.mu.Unlock()
	return nil
}

// Evaluate performs in-memory evaluation against the compiled Rego policy.
// Time Complexity: O(1) (< 0.1ms SLA)
// Space Complexity: O(1)
func (e *OPAPolicyEvaluator) Evaluate(ctx context.Context, input map[string]interface{}) (bool, error) {
	e.mu.RLock()
	query := e.regoQuery
	e.mu.RUnlock()

	results, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, err
	}

	if len(results) == 0 || len(results[0].Expressions) == 0 {
		return false, nil
	}

	allowed, ok := results[0].Expressions[0].Value.(bool)
	if !ok {
		return false, nil
	}

	return allowed, nil
}

// StartWatcher starts a background fsnotify routine to hot-reload policy changes.
func (e *OPAPolicyEvaluator) StartWatcher(ctx context.Context, logger *slog.Logger) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Watch for create or write operations on the policy file
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					logger.InfoContext(ctx, "embedded OPA policy file changed, hot-reloading...", slog.String("path", event.Name))
					if err := e.reload(ctx); err != nil {
						logger.ErrorContext(ctx, "failed to reload OPA policy rules", slog.Any("error", err))
					} else {
						logger.InfoContext(ctx, "embedded OPA policy rules successfully hot-reloaded and compiled")
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.ErrorContext(ctx, "fsnotify OPA policy watcher error", slog.Any("error", err))
			}
		}
	}()

	return watcher.Add(e.policyPath)
}

// PolicyMiddleware intercepts requests to evaluate dynamic authorization policies.
type PolicyMiddleware struct {
	evaluator *OPAPolicyEvaluator
	logger    *slog.Logger
}

// NewPolicyMiddleware constructs a PolicyMiddleware.
func NewPolicyMiddleware(evaluator *OPAPolicyEvaluator, logger *slog.Logger) *PolicyMiddleware {
	return &PolicyMiddleware{
		evaluator: evaluator,
		logger:    logger,
	}
}

// Handler returns the authorization checking middleware handler.
func (m *PolicyMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		path := r.URL.Path

		// 1. Decode JWT claims from incoming HTTP Authorization header safely
		rawClaims := m.extractJWTClaims(r)

		var sub string
		var scopes []string
		if rawClaims != nil {
			if s, ok := rawClaims["sub"].(string); ok {
				sub = s
			}
			if scopeVal, ok := rawClaims["scope"].(string); ok {
				scopes = strings.Split(scopeVal, " ")
			} else if scopeSlice, ok := rawClaims["scopes"].([]interface{}); ok {
				for _, s := range scopeSlice {
					if str, ok := s.(string); ok {
						scopes = append(scopes, str)
					}
				}
			} else if permSlice, ok := rawClaims["permissions"].([]interface{}); ok {
				for _, p := range permSlice {
					if str, ok := p.(string); ok {
						scopes = append(scopes, str)
					}
				}
			}
		}

		// 2. Safely read and buffer request body parameter mappings for POST payloads
		var body map[string]interface{}
		if r.Method == http.MethodPost {
			bodyBytes, err := m.readAndBufferBody(r)
			if err == nil && len(bodyBytes) > 0 {
				_ = json.Unmarshal(bodyBytes, &body)
			}
		}

		// 3. Assemble complete context input structure for Rego engine evaluation
		input := map[string]interface{}{
			"path":   path,
			"method": r.Method,
			"claims": map[string]interface{}{
				"sub":    sub,
				"scopes": scopes,
			},
			"body": body,
		}

		// 4. Perform ultra-low-latency local query evaluation
		allowed, err := m.evaluator.Evaluate(ctx, input)
		if err != nil {
			m.logger.ErrorContext(ctx, "OPA policy evaluation failed", slog.Any("error", err))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "internal policy evaluation error",
			})
			return
		}

		if !allowed {
			m.logger.WarnContext(ctx, "OPA policy check rejected request",
				slog.String("path", path),
				slog.String("sub", sub),
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":   "forbidden",
				"message": "access denied by system security policy",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *PolicyMiddleware) extractJWTClaims(r *http.Request) map[string]interface{} {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil
	}

	return claims
}

func (m *PolicyMiddleware) readAndBufferBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, errors.New("empty request body")
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return bodyBytes, nil
}
