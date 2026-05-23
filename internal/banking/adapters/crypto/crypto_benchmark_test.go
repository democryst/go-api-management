package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/democryst/go-api-management/internal/banking/domain"
)

// BenchmarkVerifyEnclaveSignature profiles standard ECDSA signature verification latencies.
func BenchmarkVerifyEnclaveSignature(b *testing.B) {
	// Initialize test keys and signed payloads once outside the hot loop
	privKey, pubKeyPEM := GenerateTestKeyPair(&testing.T{})

	payload := &domain.TransferPayload{
		ID:              "tx-bench",
		SenderAccount:   "1234567890",
		ReceiverAccount: "0987654321",
		AmountCents:     50000,
		Currency:        "USD",
		Timestamp:       time.Now().Truncate(time.Second),
	}

	canonicalBytes := CanonicalTransferBytes(payload)
	hash := sha256.Sum256(canonicalBytes)

	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash[:])
	if err != nil {
		b.Fatalf("failed to sign: %v", err)
	}

	derSig, err := asn1.Marshal(struct{ R, S *big.Int }{R: r, S: s})
	if err != nil {
		b.Fatalf("failed to marshal: %v", err)
	}

	payload.Signature = base64.StdEncoding.EncodeToString(derSig)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = VerifyEnclaveSignature(pubKeyPEM, payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVerifyInternalJWT profiles internal HMAC-SHA256 token verification latencies.
func BenchmarkVerifyInternalJWT(b *testing.B) {
	secret := "secret-jwt-token-swap-key-value-verification"
	scopes := []string{"transfer:execute"}
	token, err := GenerateInternalJWT("auth0|bench-user", scopes, secret, 5*time.Minute)
	if err != nil {
		b.Fatalf("failed to generate JWT: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = VerifyInternalJWT(token, secret)
		if err != nil {
			b.Fatal(err)
		}
	}
}
