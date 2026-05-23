package handler

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	jwtcore "github.com/auth0/go-jwt-middleware/v3/core"
	"github.com/auth0/go-jwt-middleware/v3/validator"
	crypto2 "github.com/democryst/go-api-management/internal/banking/adapters/crypto"
	"github.com/democryst/go-api-management/internal/banking/domain"
	"github.com/democryst/go-api-management/internal/banking/services"
)

// generateTestKeyPair generates a standard ECDSA P-256 keypair locally for handler testing.
func generateTestKeyPair(t *testing.T) (*ecdsa.PrivateKey, string) {
	t.Helper()
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	return privKey, string(pubPEM)
}

// MockAttestationVerifier implements ports.AttestationVerifier in memory for testing.
type MockAttestationVerifier struct{}

func (v *MockAttestationVerifier) VerifyAttestation(ctx context.Context, platform string, token string) error {
	if token == "compromised-token" {
		return errors.New("attestation failed: hardware compromised")
	}
	return nil
}

// MockKeyRepository implements ports.EnclaveKeyRepository in memory for testing.
type MockKeyRepository struct {
	mu   sync.Mutex
	keys map[string]domain.SecureEnclaveKey
}

func NewMockKeyRepository() *MockKeyRepository {
	return &MockKeyRepository{
		keys: make(map[string]domain.SecureEnclaveKey),
	}
}

func (r *MockKeyRepository) SaveKey(ctx context.Context, key *domain.SecureEnclaveKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.keys[key.ID] = *key
	return nil
}

func (r *MockKeyRepository) GetKey(ctx context.Context, id string) (*domain.SecureEnclaveKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key, ok := r.keys[id]
	if !ok {
		return nil, errors.New("key not found")
	}
	return &key, nil
}

func (r *MockKeyRepository) GetKeysByUserID(ctx context.Context, userID string) ([]domain.SecureEnclaveKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var userKeys []domain.SecureEnclaveKey
	for _, key := range r.keys {
		if key.UserID == userID {
			userKeys = append(userKeys, key)
		}
	}
	return userKeys, nil
}

func (r *MockKeyRepository) DeleteKey(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.keys, id)
	return nil
}

func TestBankingHandler_RegisterDevice_Success(t *testing.T) {
	repo := NewMockKeyRepository()
	verifier := &MockAttestationVerifier{}
	svc := services.NewBankingService(repo, verifier, "supersecret", "http://localhost:8081", nil, slogLogger())
	handler := NewBankingHandler(svc, slogLogger(), "domain", "audience", nil)

	// Mock OIDC JWT injector middleware using jwtcore.SetClaims helper
	mockJWTMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := &validator.ValidatedClaims{
				RegisteredClaims: validator.RegisteredClaims{
					Subject: "auth0|test-user-1",
				},
			}
			ctx := jwtcore.SetClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	router := handler.EdgeRoutes(mockJWTMiddleware)

	reqBody := RegisterDeviceRequest{
		ID:               "device-1",
		PublicKeyPEM:     "-----BEGIN PUBLIC KEY-----\nMOCK-PEM-KEY\n-----END PUBLIC KEY-----",
		Algorithm:        "ECDSA_P256",
		Platform:         "ios",
		AttestationToken: "valid-attestation",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/auth/register-device", bytes.NewBuffer(bodyBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected code 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "registered" || resp["key_id"] != "device-1" || resp["user_id"] != "auth0|test-user-1" {
		t.Errorf("unexpected response: %v", resp)
	}

	// Verify key is persisted in repo
	key, err := repo.GetKey(context.Background(), "device-1")
	if err != nil || key.UserID != "auth0|test-user-1" || key.PublicKeyPEM != "-----BEGIN PUBLIC KEY-----\nMOCK-PEM-KEY\n-----END PUBLIC KEY-----" {
		t.Errorf("key not saved properly: %v, err: %v", key, err)
	}
}

func TestBankingHandler_RegisterDevice_CompromisedAttestation(t *testing.T) {
	repo := NewMockKeyRepository()
	verifier := &MockAttestationVerifier{}
	svc := services.NewBankingService(repo, verifier, "supersecret", "http://localhost:8081", nil, slogLogger())
	handler := NewBankingHandler(svc, slogLogger(), "domain", "audience", nil)

	mockJWTMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := &validator.ValidatedClaims{
				RegisteredClaims: validator.RegisteredClaims{
					Subject: "auth0|test-user-1",
				},
			}
			ctx := jwtcore.SetClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	router := handler.EdgeRoutes(mockJWTMiddleware)

	reqBody := RegisterDeviceRequest{
		ID:               "device-1",
		PublicKeyPEM:     "-----BEGIN PUBLIC KEY-----\nMOCK-PEM-KEY\n-----END PUBLIC KEY-----",
		Algorithm:        "ECDSA_P256",
		Platform:         "ios",
		AttestationToken: "compromised-token",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/auth/register-device", bytes.NewBuffer(bodyBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected code 403 on compromised attestation, got %d", w.Code)
	}
}

func TestBankingHandler_FullTransferRoundtrip_Success(t *testing.T) {
	repo := NewMockKeyRepository()
	verifier := &MockAttestationVerifier{}

	// Generate test key locally
	privKey, pubKeyPEM := generateTestKeyPair(t)

	// Persist key to Mock Repo for "auth0|test-user-1"
	err := repo.SaveKey(context.Background(), &domain.SecureEnclaveKey{
		ID:           "device-key-1",
		UserID:       "auth0|test-user-1",
		PublicKeyPEM: pubKeyPEM,
		Algorithm:    "ECDSA_P256",
		RegisteredAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to seed key: %v", err)
	}

	// Spin up Private Intranet Server (VPC Secure Zone)
	sharedSecret := "intranethighprivilegesecret"
	privateHandler := NewBankingHandler(services.NewBankingService(repo, verifier, sharedSecret, "", nil, slogLogger()), slogLogger(), "", "", nil)
	privateRouter := privateHandler.PrivateRoutes()
	privateServer := httptest.NewServer(privateRouter)
	defer privateServer.Close()

	// Initialize DMZ Edge Gateway with private URL referencing Intranet Server
	edgeSvc := services.NewBankingService(repo, verifier, sharedSecret, privateServer.URL+"/private/transfers", nil, slogLogger())
	edgeHandler := NewBankingHandler(edgeSvc, slogLogger(), "domain", "audience", nil)

	mockJWTMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := &validator.ValidatedClaims{
				RegisteredClaims: validator.RegisteredClaims{
					Subject: "auth0|test-user-1",
				},
			}
			ctx := jwtcore.SetClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	edgeRouter := edgeHandler.EdgeRoutes(mockJWTMiddleware)

	// Construct signed payload
	payload := &domain.TransferPayload{
		ID:              "tx-999",
		SenderAccount:   "1234567890",
		ReceiverAccount: "0987654321",
		AmountCents:     50000,
		Currency:        "USD",
		Timestamp:       time.Now().Truncate(time.Second),
	}

	canonicalBytes := crypto2.CanonicalTransferBytes(payload)
	hash := sha256.Sum256(canonicalBytes)
	rVal, sVal, err := ecdsa.Sign(rand.Reader, privKey, hash[:])
	if err != nil {
		t.Fatalf("failed to sign payload: %v", err)
	}
	derSig, _ := asn1.Marshal(struct{ R, S *big.Int }{R: rVal, S: sVal})
	payload.Signature = base64.StdEncoding.EncodeToString(derSig)

	// POST to Edge Gateway
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/transfer/initiate", bytes.NewBuffer(bodyBytes))
	w := httptest.NewRecorder()

	edgeRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected edge response code 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "initiated" || resp["tx_id"] != "tx-999" {
		t.Errorf("unexpected edge response: %v", resp)
	}
}

func TestBankingHandler_TransferRoundtrip_TamperedRejection(t *testing.T) {
	repo := NewMockKeyRepository()
	verifier := &MockAttestationVerifier{}

	privKey, pubKeyPEM := generateTestKeyPair(t)

	_ = repo.SaveKey(context.Background(), &domain.SecureEnclaveKey{
		ID:           "device-key-1",
		UserID:       "auth0|test-user-1",
		PublicKeyPEM: pubKeyPEM,
		Algorithm:    "ECDSA_P256",
		RegisteredAt: time.Now(),
	})

	sharedSecret := "intranethighprivilegesecret"
	privateHandler := NewBankingHandler(services.NewBankingService(repo, verifier, sharedSecret, "", nil, slogLogger()), slogLogger(), "", "", nil)
	privateRouter := privateHandler.PrivateRoutes()
	privateServer := httptest.NewServer(privateRouter)
	defer privateServer.Close()

	edgeSvc := services.NewBankingService(repo, verifier, sharedSecret, privateServer.URL+"/private/transfers", nil, slogLogger())
	edgeHandler := NewBankingHandler(edgeSvc, slogLogger(), "domain", "audience", nil)

	mockJWTMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := &validator.ValidatedClaims{
				RegisteredClaims: validator.RegisteredClaims{
					Subject: "auth0|test-user-1",
				},
			}
			ctx := jwtcore.SetClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	edgeRouter := edgeHandler.EdgeRoutes(mockJWTMiddleware)

	// Construct signed payload
	payload := &domain.TransferPayload{
		ID:              "tx-999",
		SenderAccount:   "1234567890",
		ReceiverAccount: "0987654321",
		AmountCents:     50000,
		Currency:        "USD",
		Timestamp:       time.Now().Truncate(time.Second),
	}

	canonicalBytes := crypto2.CanonicalTransferBytes(payload)
	hash := sha256.Sum256(canonicalBytes)
	rVal, sVal, _ := ecdsa.Sign(rand.Reader, privKey, hash[:])
	derSig, _ := asn1.Marshal(struct{ R, S *big.Int }{R: rVal, S: sVal})
	payload.Signature = base64.StdEncoding.EncodeToString(derSig)

	// Tamper the payload
	payload.AmountCents = 1000000

	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/transfer/initiate", bytes.NewBuffer(bodyBytes))
	w := httptest.NewRecorder()

	edgeRouter.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Error("expected edge request to fail due to tampered amount signature verification, but got success code 200")
	}
}

func slogLogger() *slog.Logger {
	return slog.Default()
}
