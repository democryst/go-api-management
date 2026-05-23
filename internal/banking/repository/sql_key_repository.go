package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/democryst/go-api-management/internal/banking/domain"
	"github.com/democryst/go-api-management/internal/banking/ports"
	"github.com/jmoiron/sqlx"
)

var _ ports.EnclaveKeyRepository = (*SQLEnclaveKeyRepository)(nil)

// SQLEnclaveKeyRepository implements the ports.EnclaveKeyRepository outbound adapter backed by SQLX.
type SQLEnclaveKeyRepository struct {
	db     *sqlx.DB
	logger *slog.Logger
}

// NewSQLEnclaveKeyRepository constructs a new SQLEnclaveKeyRepository.
func NewSQLEnclaveKeyRepository(db *sqlx.DB, logger *slog.Logger) *SQLEnclaveKeyRepository {
	return &SQLEnclaveKeyRepository{
		db:     db,
		logger: logger,
	}
}

// SaveKey registers a hardware-backed device public key tied to a user.
// Time Complexity: O(1) (DB INSERT bound)
// Space Complexity: O(1)
func (r *SQLEnclaveKeyRepository) SaveKey(ctx context.Context, key *domain.SecureEnclaveKey) error {
	const query = `
		INSERT INTO device_enclave_keys (id, user_id, public_key_pem, algorithm, registered_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	r.logger.InfoContext(ctx, "persisting Secure Enclave public key",
		slog.String("key_id", key.ID),
		slog.String("user_id", key.UserID),
		slog.String("algorithm", key.Algorithm),
	)

	_, err := r.db.ExecContext(ctx, query, key.ID, key.UserID, key.PublicKeyPEM, key.Algorithm, key.RegisteredAt)
	if err != nil {
		return fmt.Errorf("failed to insert enclave key: %w", err)
	}
	return nil
}

// GetKey retrieves a registered Secure Enclave key by its unique device identifier.
// Time Complexity: O(1) (Indexed PK lookup)
// Space Complexity: O(1)
func (r *SQLEnclaveKeyRepository) GetKey(ctx context.Context, id string) (*domain.SecureEnclaveKey, error) {
	const query = `
		SELECT id, user_id, public_key_pem, algorithm, registered_at
		FROM device_enclave_keys
		WHERE id = $1
	`
	r.logger.InfoContext(ctx, "retrieving Secure Enclave public key by ID", slog.String("key_id", id))

	var key domain.SecureEnclaveKey
	err := r.db.GetContext(ctx, &key, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("enclave key not found: %s", id)
		}
		return nil, fmt.Errorf("failed to query enclave key: %w", err)
	}
	return &key, nil
}

// GetKeysByUserID lists all registered enclave keys belonging to a user.
// Time Complexity: O(K) where K is the number of keys registered for the user (indexed scan)
// Space Complexity: O(K)
func (r *SQLEnclaveKeyRepository) GetKeysByUserID(ctx context.Context, userID string) ([]domain.SecureEnclaveKey, error) {
	const query = `
		SELECT id, user_id, public_key_pem, algorithm, registered_at
		FROM device_enclave_keys
		WHERE user_id = $1
		ORDER BY registered_at DESC
	`
	r.logger.InfoContext(ctx, "retrieving all Secure Enclave keys for user", slog.String("user_id", userID))

	var keys []domain.SecureEnclaveKey
	err := r.db.SelectContext(ctx, &keys, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to select enclave keys by user ID: %w", err)
	}
	return keys, nil
}

// DeleteKey unbinds and removes a registered device public key.
// Time Complexity: O(1) (Indexed delete)
// Space Complexity: O(1)
func (r *SQLEnclaveKeyRepository) DeleteKey(ctx context.Context, id string) error {
	const query = `
		DELETE FROM device_enclave_keys
		WHERE id = $1
	`
	r.logger.InfoContext(ctx, "revoking/deleting Secure Enclave public key", slog.String("key_id", id))

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete enclave key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return fmt.Errorf("enclave key not found: %s", id)
	}
	return nil
}
