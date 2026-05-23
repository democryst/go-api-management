package ports

import (
    "context"
    "github.com/democryst/go-api-management/internal/banking/domain"
)

// AttestationVerifier defines the outbound port boundary for verifying mobile device hardware attestation (SafetyNet / Play Integrity / App Attest).
type AttestationVerifier interface {
    // VerifyAttestation validates that the cryptographic attestation token is signed and untampered.
    VerifyAttestation(ctx context.Context, platform string, token string) error
}

// EnclaveKeyRepository defines the outbound port boundary for persisting client public keys generated inside Secure Enclaves.
type EnclaveKeyRepository interface {
    // SaveKey registers a hardware-backed device public key tied to a user.
    SaveKey(ctx context.Context, key *domain.SecureEnclaveKey) error

    // GetKey retrieves a registered Secure Enclave key by its unique device identifier.
    GetKey(ctx context.Context, id string) (*domain.SecureEnclaveKey, error)

    // GetKeysByUserID lists all registered enclave keys belonging to a user.
    GetKeysByUserID(ctx context.Context, userID string) ([]domain.SecureEnclaveKey, error)

    // DeleteKey unbinds and removes a registered device public key.
    DeleteKey(ctx context.Context, id string) error
}
