package repository

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "log/slog"
    "time"

    "github.com/democryst/go-api-management/internal/core/domain"
    "github.com/democryst/go-api-management/internal/core/ports"
    "github.com/jmoiron/sqlx"
)

// Ensure SQLSessionRepository satisfies ports.SessionRepository at compile time.
var _ ports.SessionRepository = (*SQLSessionRepository)(nil)

// SQLSessionRepository implements the session repository outbound adapter backed by SQLX/PostgreSQL.
type SQLSessionRepository struct {
    db     *sqlx.DB
    logger *slog.Logger
}

// NewSQLSessionRepository constructs a new SQLSessionRepository adapter.
func NewSQLSessionRepository(db *sqlx.DB, logger *slog.Logger) *SQLSessionRepository {
    return &SQLSessionRepository{
        db:     db,
        logger: logger,
    }
}

// Save registers a transient authentication state session inside the database.
// Time Complexity: O(1) (Postgres INSERT bound)
// Space Complexity: O(1)
func (r *SQLSessionRepository) Save(ctx context.Context, session *domain.AuthSession) error {
    const query = `
        INSERT INTO auth_sessions (id, state, code_verifier, redirect_uri, expires_at)
        VALUES ($1, $2, $3, $4, $5)
    `
    r.logger.InfoContext(ctx, "inserting transient auth session", slog.String("session_id", session.ID), slog.String("state", session.State))

    _, err := r.db.ExecContext(ctx, query, session.ID, session.State, session.CodeVerifier, session.RedirectURI, session.ExpiresAt)
    if err != nil {
        return fmt.Errorf("failed to save auth session: %w", err)
    }
    return nil
}

// Get retrieves an active transient authentication session by its UUID identifier.
// Time Complexity: O(1) (Indexed UUID lookup)
// Space Complexity: O(1)
func (r *SQLSessionRepository) Get(ctx context.Context, id string) (*domain.AuthSession, error) {
    const query = `
        SELECT id, state, code_verifier, redirect_uri, created_at, expires_at
        FROM auth_sessions
        WHERE id = $1
    `
    r.logger.InfoContext(ctx, "retrieving transient auth session", slog.String("session_id", id))

    var session domain.AuthSession
    err := r.db.GetContext(ctx, &session, query, id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, fmt.Errorf("%w: session %s", domain.ErrSessionNotFound, id)
        }
        return nil, fmt.Errorf("failed to query auth session: %w", err)
    }
    return &session, nil
}

// Delete clears the transient authentication session instantly upon consumption (single-use constraint).
// Time Complexity: O(1) (Indexed delete)
// Space Complexity: O(1)
func (r *SQLSessionRepository) Delete(ctx context.Context, id string) error {
    const query = `
        DELETE FROM auth_sessions
        WHERE id = $1
    `
    r.logger.InfoContext(ctx, "deleting transient auth session", slog.String("session_id", id))

    result, err := r.db.ExecContext(ctx, query, id)
    if err != nil {
        return fmt.Errorf("failed to delete auth session: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err == nil && rowsAffected == 0 {
        return fmt.Errorf("%w: session %s", domain.ErrSessionNotFound, id)
    }
    return nil
}

// DeleteExpired sweeps and deletes expired session garbage in a batch background task.
// Time Complexity: O(M) where M is the number of expired rows deleted.
// Space Complexity: O(1)
func (r *SQLSessionRepository) DeleteExpired(ctx context.Context) error {
    const query = `
        DELETE FROM auth_sessions
        WHERE expires_at < $1
    `
    now := time.Now()
    r.logger.InfoContext(ctx, "sweeping expired transient auth sessions", slog.Time("now", now))

    result, err := r.db.ExecContext(ctx, query, now)
    if err != nil {
        return fmt.Errorf("failed to delete expired auth sessions: %w", err)
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected > 0 {
        r.logger.InfoContext(ctx, "purged expired auth sessions", slog.Int64("purged_rows", rowsAffected))
    }
    return nil
}
