package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// SchemaValidationError represents a contract validation failure.
type SchemaValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// SchemaValidationMiddleware intercepts requests to validate payload boundaries against openapi.yaml specs.
// Time Complexity: O(N) where N is the payload byte size (JSON decode bound).
// Space Complexity: O(N) for reading and buffering the request body.
func SchemaValidationMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// 1. Intercept Device Key Registration
			if path == "/auth/register-device" && r.Method == http.MethodPost {
				bodyBytes, err := readAndBufferBody(r)
				if err != nil {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "body", "failed to read request body")
					return
				}

				var req struct {
					ID               string `json:"id"`
					PublicKeyPEM     string `json:"public_key_pem"`
					Algorithm        string `json:"algorithm"`
					Platform         string `json:"platform"`
					AttestationToken string `json:"attestation_token"`
				}

				if err := json.Unmarshal(bodyBytes, &req); err != nil {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "body", "malformed JSON payload")
					return
				}

				// Enforce declarative OpenAPI parameter constraints
				if req.ID == "" {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "id", "parameter is required and cannot be empty")
					return
				}
				if !strings.Contains(req.PublicKeyPEM, "-----BEGIN PUBLIC KEY-----") {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "public_key_pem", "must be a valid PEM public key block")
					return
				}
				if req.Algorithm != "ECDSA_P256" {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "algorithm", "must be exactly ECDSA_P256")
					return
				}
				platformLower := strings.ToLower(req.Platform)
				if platformLower != "ios" && platformLower != "android" {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "platform", "must be 'ios' or 'android'")
					return
				}
				if req.AttestationToken == "" {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "attestation_token", "parameter is required and cannot be empty")
					return
				}
			}

			// 2. Intercept Money Transfer initiation and Intranet Ledger execution
			if (path == "/transfer/initiate" || path == "/private/transfers") && r.Method == http.MethodPost {
				bodyBytes, err := readAndBufferBody(r)
				if err != nil {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "body", "failed to read request body")
					return
				}

				var req struct {
					ID              string    `json:"id"`
					SenderAccount   string    `json:"sender_account"`
					ReceiverAccount string    `json:"receiver_account"`
					AmountCents     int64     `json:"amount_cents"`
					Currency        string    `json:"currency"`
					Timestamp       time.Time `json:"timestamp"`
					Signature       string    `json:"signature"`
				}

				if err := json.Unmarshal(bodyBytes, &req); err != nil {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "body", "malformed JSON payload")
					return
				}

				// Enforce OpenAPI transfer payload parameters validation
				if req.ID == "" {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "id", "parameter is required and cannot be empty")
					return
				}
				if len(req.SenderAccount) < 6 {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "sender_account", "must be at least 6 characters in length")
					return
				}
				if len(req.ReceiverAccount) < 6 {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "receiver_account", "must be at least 6 characters in length")
					return
				}
				if req.AmountCents <= 0 {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "amount_cents", "must be a positive integer greater than 0")
					return
				}
				if len(req.Currency) != 3 {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "currency", "must be exactly 3 characters (ISO 4217 code)")
					return
				}
				if req.Timestamp.IsZero() {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "timestamp", "must be a valid RFC3339 formatted date-time")
					return
				}
				if req.Signature == "" {
					writeSchemaError(r.Context(), w, http.StatusBadRequest, "signature", "parameter is required and cannot be empty")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func readAndBufferBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, errors.New("empty request body")
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	// Restore request body for downstream handlers
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return bodyBytes, nil
}

func writeSchemaError(ctx context.Context, w http.ResponseWriter, code int, field string, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": "openapi schema contract validation failed",
		"details": SchemaValidationError{
			Field:   field,
			Message: msg,
		},
	})
}
