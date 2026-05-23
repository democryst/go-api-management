package services

import (
    "context"
    "errors"
    "log/slog"
    "os"
    "testing"
    "time"

    "github.com/democryst/go-api-management/internal/core/domain"
    "github.com/democryst/go-api-management/internal/core/ports"
)

// Zero-Magic Mock Implementation: SessionRepository
type mockSessionRepository struct {
    ports.SessionRepository
    sessions map[string]*domain.AuthSession
    saveCalled bool
    getCalled bool
    deleteCalled bool
    deletedID string
}

func (m *mockSessionRepository) Save(ctx context.Context, s *domain.AuthSession) error {
    m.saveCalled = true
    m.sessions[s.ID] = s
    return nil
}

func (m *mockSessionRepository) Get(ctx context.Context, id string) (*domain.AuthSession, error) {
    m.getCalled = true
    s, ok := m.sessions[id]
    if !ok {
        return nil, domain.ErrSessionNotFound
    }
    return s, nil
}

func (m *mockSessionRepository) Delete(ctx context.Context, id string) error {
    m.deleteCalled = true
    m.deletedID = id
    delete(m.sessions, id)
    return nil
}

// Zero-Magic Mock Implementation: AuthProvider
type mockAuthProvider struct {
    ports.AuthProvider
    authURL string
    tokens  domain.TokenPair
    user    domain.UserIdentity
}

func (m *mockAuthProvider) GetAuthorizationURL(ctx context.Context, s domain.AuthSession) (string, error) {
    return m.authURL, nil
}

func (m *mockAuthProvider) ExchangeCode(ctx context.Context, code, verifier string) (domain.TokenPair, error) {
    return m.tokens, nil
}

func (m *mockAuthProvider) GetUserInfo(ctx context.Context, accessToken string) (domain.UserIdentity, error) {
    return m.user, nil
}

// TestInitiateAuth verifies state & verifier persistence and correct redirect link formatting.
func TestInitiateAuth(t *testing.T) {
    repo := &mockSessionRepository{sessions: make(map[string]*domain.AuthSession)}
    provider := &mockAuthProvider{authURL: "https://auth0.com/authorize"}
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    service := NewAuthService(repo, provider, logger)

    url, sessionID, err := service.InitiateAuth(context.Background(), "https://client.com/callback")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if url != "https://auth0.com/authorize" {
        t.Errorf("expected redirect url, got %s", url)
    }

    if !repo.saveCalled {
        t.Error("expected repository Save to have been called")
    }

    storedSession := repo.sessions[sessionID]
    if storedSession == nil {
        t.Fatal("session was not saved in mock repository")
    }

    if storedSession.RedirectURI != "https://client.com/callback" {
        t.Errorf("expected callback URI matching parameter, got %s", storedSession.RedirectURI)
    }
}

// TestHandleCallback_Success asserts successful callback exchanges and instant verifier deletion.
func TestHandleCallback_Success(t *testing.T) {
    repo := &mockSessionRepository{sessions: make(map[string]*domain.AuthSession)}
    provider := &mockAuthProvider{
        tokens: domain.TokenPair{AccessToken: "token-123"},
        user:   domain.UserIdentity{ID: "auth0|user1"},
    }
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    service := NewAuthService(repo, provider, logger)

    // Pre-insert session
    sessionID := "session-abc"
    session := &domain.AuthSession{
        ID:           sessionID,
        State:        "state-123",
        CodeVerifier: "verifier-123",
        RedirectURI:  "https://client.com/callback",
        ExpiresAt:    time.Now().Add(5 * time.Minute),
    }
    _ = repo.Save(context.Background(), session)

    user, tokens, err := service.HandleCallback(context.Background(), sessionID, "code-123", "state-123")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if user.ID != "auth0|user1" {
        t.Errorf("expected user sub matching payload, got %s", user.ID)
    }

    if tokens.AccessToken != "token-123" {
        t.Errorf("expected access token matching exchange, got %s", tokens.AccessToken)
    }

    // Critical: Verify instant single-use deletion!
    if !repo.deleteCalled || repo.deletedID != sessionID {
        t.Error("expected DB session to be deleted instantly upon extraction")
    }
}

// TestHandleCallback_Expired asserts session TTL expiry triggers immediate failures.
func TestHandleCallback_Expired(t *testing.T) {
    repo := &mockSessionRepository{sessions: make(map[string]*domain.AuthSession)}
    provider := &mockAuthProvider{}
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    service := NewAuthService(repo, provider, logger)

    sessionID := "session-expired"
    session := &domain.AuthSession{
        ID:           sessionID,
        State:        "state-123",
        CodeVerifier: "verifier-123",
        ExpiresAt:    time.Now().Add(-1 * time.Minute), // Expired!
    }
    _ = repo.Save(context.Background(), session)

    _, _, err := service.HandleCallback(context.Background(), sessionID, "code-123", "state-123")
    if err == nil {
        t.Fatal("expected error for expired session callback, got nil")
    }

    if !errors.Is(err, domain.ErrSessionExpired) {
        t.Errorf("expected domain.ErrSessionExpired, got %v", err)
    }
}

// TestHandleCallback_StateMismatch asserts CSRF state forged callbacks get rejected.
func TestHandleCallback_StateMismatch(t *testing.T) {
    repo := &mockSessionRepository{sessions: make(map[string]*domain.AuthSession)}
    provider := &mockAuthProvider{}
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    service := NewAuthService(repo, provider, logger)

    sessionID := "session-forged"
    session := &domain.AuthSession{
        ID:           sessionID,
        State:        "legitimate-state",
        CodeVerifier: "verifier-123",
        ExpiresAt:    time.Now().Add(5 * time.Minute),
    }
    _ = repo.Save(context.Background(), session)

    _, _, err := service.HandleCallback(context.Background(), sessionID, "code-123", "forged-state-parameter")
    if err == nil {
        t.Fatal("expected error for mismatched state callback, got nil")
    }

    if !errors.Is(err, domain.ErrInvalidState) {
        t.Errorf("expected domain.ErrInvalidState, got %v", err)
    }
}
