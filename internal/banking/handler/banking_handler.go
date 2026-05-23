package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/auth0/go-jwt-middleware/v3"
	"github.com/auth0/go-jwt-middleware/v3/validator"
	"github.com/democryst/go-api-management/internal/banking/domain"
	"github.com/democryst/go-api-management/internal/banking/services"
	"github.com/go-chi/chi/v5"
)

// BankingHandler wraps HTTP controllers for secure banking routes.
type BankingHandler struct {
	service     *services.BankingService
	logger      *slog.Logger
	auth0Domain string
	apiAudience string
	jwtProvider func(context.Context) (any, error)
}

// NewBankingHandler constructs a new BankingHandler.
func NewBankingHandler(
	service *services.BankingService,
	logger *slog.Logger,
	auth0Domain string,
	apiAudience string,
	jwtProvider func(context.Context) (any, error),
) *BankingHandler {
	return &BankingHandler{
		service:     service,
		logger:      logger,
		auth0Domain: auth0Domain,
		apiAudience: apiAudience,
		jwtProvider: jwtProvider,
	}
}

// RegisterDeviceRequest defines the request body for binding a Secure Enclave public key.
type RegisterDeviceRequest struct {
	ID               string `json:"id"`
	PublicKeyPEM     string `json:"public_key_pem"`
	Algorithm        string `json:"algorithm"`
	Platform         string `json:"platform"`
	AttestationToken string `json:"attestation_token"`
}

// EdgeRoutes configures and returns Chi routing for the public DMZ edge gateway.
// Time Complexity: O(1)
// Space Complexity: O(1)
func (h *BankingHandler) EdgeRoutes(jwtMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Authenticated Device & Transfer Operations
	r.Group(func(r chi.Router) {
		r.Use(jwtMiddleware)
		r.Post("/auth/register-device", h.RegisterDevice)
		r.Post("/transfer/initiate", h.InitiateTransfer)
	})

	return r
}

// PrivateRoutes configures and returns Chi routing for the Private Intranet Zone.
// Time Complexity: O(1)
// Space Complexity: O(1)
func (h *BankingHandler) PrivateRoutes() chi.Router {
	r := chi.NewRouter()

	// Secure Intranet Private Ledgers
	r.Post("/private/transfers", h.ExecutePrivateTransfer)

	return r
}

// RegisterDevice handles POST /auth/register-device to bind device keys.
// Time Complexity: O(1) (network and DB bound)
// Space Complexity: O(1)
func (h *BankingHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractSubject(r)
	if err != nil {
		h.writeError(r.Context(), w, http.StatusUnauthorized, err.Error())
		return
	}

	var req RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ID == "" || req.PublicKeyPEM == "" || req.Algorithm == "" || req.Platform == "" || req.AttestationToken == "" {
		h.writeError(r.Context(), w, http.StatusBadRequest, "Missing required parameters")
		return
	}

	key := &domain.SecureEnclaveKey{
		ID:           req.ID,
		PublicKeyPEM: req.PublicKeyPEM,
		Algorithm:    req.Algorithm,
	}

	err = h.service.RegisterDeviceKey(r.Context(), userID, key, req.Platform, req.AttestationToken)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "device registration failed", slog.Any("error", err))
		h.writeError(r.Context(), w, http.StatusForbidden, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "registered",
		"key_id":  req.ID,
		"user_id": userID,
	})
}

// InitiateTransfer handles POST /transfer/initiate from the public Edge BFF.
// Time Complexity: O(1) (network bound)
// Space Complexity: O(1)
func (h *BankingHandler) InitiateTransfer(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractSubject(r)
	if err != nil {
		h.writeError(r.Context(), w, http.StatusUnauthorized, err.Error())
		return
	}

	var payload domain.TransferPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeError(r.Context(), w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err = h.service.InitiateTransfer(r.Context(), userID, &payload)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "transfer initiation rejected at Edge", slog.Any("error", err))
		h.writeError(r.Context(), w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "initiated",
		"tx_id":  payload.ID,
	})
}

// ExecutePrivateTransfer handles POST /private/transfers in the Intranet Zone.
// Time Complexity: O(K) where K is the number of keys registered for the user.
// Space Complexity: O(1)
func (h *BankingHandler) ExecutePrivateTransfer(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		h.writeError(r.Context(), w, http.StatusUnauthorized, "Missing or invalid Authorization header")
		return
	}
	internalToken := strings.TrimPrefix(authHeader, "Bearer ")

	var payload domain.TransferPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeError(r.Context(), w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.service.ExecuteIntranetTransfer(r.Context(), internalToken, &payload)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "intranet ledger execution rejected", slog.Any("error", err))
		h.writeError(r.Context(), w, http.StatusUnauthorized, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"tx_id":  payload.ID,
	})
}

func (h *BankingHandler) extractSubject(r *http.Request) (string, error) {
	claims, err := jwtmiddleware.GetClaims[*validator.ValidatedClaims](r.Context())
	if err != nil || claims == nil {
		return "", errors.New("unauthorized: missing token claims in request context")
	}
	return claims.RegisteredClaims.Subject, nil
}

func (h *BankingHandler) writeError(ctx context.Context, w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}
