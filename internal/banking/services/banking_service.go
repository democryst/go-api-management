package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/democryst/go-api-management/internal/banking/adapters/crypto"
	"github.com/democryst/go-api-management/internal/banking/domain"
	"github.com/democryst/go-api-management/internal/banking/ports"
	"github.com/sony/gobreaker"
)

// BankingService coordinates device registrations, edge-to-intranet token swaps, and enclave signature validation.
type BankingService struct {
	keyRepo      ports.EnclaveKeyRepository
	verifier     ports.AttestationVerifier
	sharedSecret string
	privateURL   string
	httpClient   *http.Client
	cb           *gobreaker.CircuitBreaker
	logger       *slog.Logger
}

// NewBankingService constructs a new BankingService.
func NewBankingService(
	keyRepo ports.EnclaveKeyRepository,
	verifier ports.AttestationVerifier,
	sharedSecret string,
	privateURL string,
	httpClient *http.Client,
	logger *slog.Logger,
) *BankingService {
	// Configure standard gobreaker sliding-window settings
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "PrivateCoreBankingClient",
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.5
		},
	})

	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	return &BankingService{
		keyRepo:      keyRepo,
		verifier:     verifier,
		sharedSecret: sharedSecret,
		privateURL:   privateURL,
		httpClient:   httpClient,
		cb:           cb,
		logger:       logger,
	}
}

// RegisterDeviceKey registers a client hardware public key after verifying SafetyNet / Play Integrity / App Attest tokens.
// Time Complexity: O(1) (network and DB bound)
// Space Complexity: O(1)
func (s *BankingService) RegisterDeviceKey(ctx context.Context, userID string, key *domain.SecureEnclaveKey, platform string, attestationToken string) error {
	s.logger.InfoContext(ctx, "registering device key with attestation", slog.String("user_id", userID), slog.String("key_id", key.ID))

	// 1. Verify Device Attestation
	if err := s.verifier.VerifyAttestation(ctx, platform, attestationToken); err != nil {
		s.logger.WarnContext(ctx, "device attestation failed", slog.String("user_id", userID), slog.Any("error", err))
		return fmt.Errorf("device integrity check failed: %w", err)
	}

	// 2. Persist hardware-backed public key
	key.UserID = userID
	key.RegisteredAt = time.Now()
	if err := s.keyRepo.SaveKey(ctx, key); err != nil {
		s.logger.ErrorContext(ctx, "failed to save enclave public key", slog.String("user_id", userID), slog.Any("error", err))
		return fmt.Errorf("failed to save enclave key: %w", err)
	}

	s.logger.InfoContext(ctx, "device key registered successfully", slog.String("user_id", userID), slog.String("key_id", key.ID))
	return nil
}

// InitiateTransfer acts as the Edge DMZ BFF Token Swapper. It translates low-privilege sessions to scoped Internal JWTs,
// forwards them over private network boundaries via mTLS, and manages network resilience via sliding-window circuit breakers.
// Time Complexity: O(1) (network bound)
// Space Complexity: O(1)
func (s *BankingService) InitiateTransfer(ctx context.Context, userID string, payload *domain.TransferPayload) error {
	s.logger.InfoContext(ctx, "initiating money transfer edge request",
		slog.String("user_id", userID),
		slog.String("tx_id", payload.ID),
		slog.Any("sender", payload.SenderAccount),
		slog.Any("receiver", payload.ReceiverAccount),
		slog.Int64("amount_cents", payload.AmountCents),
	)

	// 1. Core validations
	if payload.AmountCents <= 0 {
		return errors.New("invalid transfer amount: must be positive")
	}
	if payload.SenderAccount == "" || payload.ReceiverAccount == "" {
		return errors.New("sender and receiver accounts must not be empty")
	}

	// 2. Perform Token Swap: Generate short-lived internal JWT with execute scope
	internalJWT, err := crypto.GenerateInternalJWT(userID, []string{"transfer:execute"}, s.sharedSecret, 1*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to perform token swap: %w", err)
	}

	// 3. Serialize transaction payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize payload: %w", err)
	}

	// 4. Forward to Core Intranet HTTP Service through gobreaker circuit breaker
	_, err = s.cb.Execute(func() (interface{}, error) {
		req, err := http.NewRequestWithContext(ctx, "POST", s.privateURL, bytes.NewBuffer(payloadBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+internalJWT)
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			s.logger.ErrorContext(ctx, "intranet peer HTTP request failed", slog.Any("error", err))
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errResp map[string]string
			_ = json.NewDecoder(resp.Body).Decode(&errResp)
			errMsg := errResp["error"]
			if errMsg == "" {
				errMsg = fmt.Sprintf("HTTP status %d", resp.StatusCode)
			}
			return nil, fmt.Errorf("intranet execution rejected: %s", errMsg)
		}

		return "SUCCESS", nil
	})

	if err != nil {
		s.logger.ErrorContext(ctx, "failed to initiate transfer via intranet connection", slog.Any("error", err))
		return fmt.Errorf("transfer execution failed: %w", err)
	}

	s.logger.InfoContext(ctx, "transfer execution completed successfully", slog.String("tx_id", payload.ID))
	return nil
}

// ExecuteIntranetTransfer processes private intranet-bound transaction requests.
// It verifies the high-privilege internal JWT, resolves active device keys, and performs hardware non-repudiation signature verification.
// Time Complexity: O(K) where K is the number of keys registered for the user.
// Space Complexity: O(1)
func (s *BankingService) ExecuteIntranetTransfer(ctx context.Context, internalToken string, payload *domain.TransferPayload) error {
	s.logger.InfoContext(ctx, "executing private VPC intranet ledger transfer", slog.String("tx_id", payload.ID))

	// 1. Verify Internal JWT
	claims, err := crypto.VerifyInternalJWT(internalToken, s.sharedSecret)
	if err != nil {
		s.logger.WarnContext(ctx, "internal JWT validation failed", slog.Any("error", err))
		return fmt.Errorf("unauthorized internal token: %w", err)
	}

	// 2. Validate transaction execute scope
	hasScope := false
	for _, sc := range claims.Scopes {
		if sc == "transfer:execute" {
			hasScope = true
			break
		}
	}
	if !hasScope {
		return errors.New("unauthorized: missing required 'transfer:execute' scope")
	}

	// 3. Retrieve user registered enclave keys from persistent storage
	keys, err := s.keyRepo.GetKeysByUserID(ctx, claims.Subject)
	if err != nil {
		return fmt.Errorf("failed to retrieve enclave keys for user: %w", err)
	}
	if len(keys) == 0 {
		return errors.New("unauthorized: user has zero registered device enclave keys")
	}

	// 4. Verify Secure Enclave biometric hardware signature (non-repudiation)
	var signatureValid bool
	for _, key := range keys {
		if err := crypto.VerifyEnclaveSignature(key.PublicKeyPEM, payload); err == nil {
			signatureValid = true
			s.logger.InfoContext(ctx, "Secure Enclave signature validated successfully", slog.String("key_id", key.ID))
			break
		}
	}

	if !signatureValid {
		s.logger.WarnContext(ctx, "Secure Enclave signature check failed for all registered keys")
		return errors.New("unauthorized: cryptographic secure enclave signature is invalid or tampered")
	}

	// 5. Execute ledger entry (ledger simulation)
	s.logger.InfoContext(ctx, "transaction finalized on core secure banking ledger",
		slog.String("tx_id", payload.ID),
		slog.Any("sender", payload.SenderAccount),
		slog.Any("receiver", payload.ReceiverAccount),
		slog.Int64("amount_cents", payload.AmountCents),
	)

	return nil
}
