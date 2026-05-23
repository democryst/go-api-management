package ports

import (
    "context"
    "github.com/democryst/go-api-management/internal/core/domain"
)

// SessionRepository defines the outbound port boundary for PostgreSQL session transactions.
type SessionRepository interface {
    // Save registers a transient authentication state session inside the database.
    Save(ctx context.Context, session *domain.AuthSession) error

    // Get retrieves an active transient authentication session by its UUID identifier.
    Get(ctx context.Context, id string) (*domain.AuthSession, error)

    // Delete clears the transient authentication session instantly upon consumption (single-use constraint).
    Delete(ctx context.Context, id string) error

    // DeleteExpired sweeps and deletes expired session garbage in a batch background task.
    DeleteExpired(ctx context.Context) error
}
