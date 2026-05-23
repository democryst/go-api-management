package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/democryst/go-api-management/internal/banking/domain"
)

// GenerateTestKeyPair generates a standard ECDSA P-256 keypair and returns the private key and the public key in PEM format.
func GenerateTestKeyPair(t *testing.T) (*ecdsa.PrivateKey, string) {
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

func TestVerifyEnclaveSignature_Success(t *testing.T) {
	privKey, pubKeyPEM := GenerateTestKeyPair(t)

	payload := &domain.TransferPayload{
		ID:              "tx-12345",
		SenderAccount:   "1234567890",
		ReceiverAccount: "0987654321",
		AmountCents:     100000, // $1000.00
		Currency:        "USD",
		Timestamp:       time.Now().Truncate(time.Second), // Truncate to match canonicalized seconds
	}

	// 1. Sign canonical bytes
	canonicalBytes := CanonicalTransferBytes(payload)
	hash := sha256.Sum256(canonicalBytes)

	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash[:])
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	// Marshal to DER ASN.1
	type asnSig struct {
		R, S *big.Int
	}
	derSig, err := asn1.Marshal(asnSig{R: r, S: s})
	if err != nil {
		t.Fatalf("failed to marshal DER: %v", err)
	}

	// Base64 encode
	payload.Signature = base64.StdEncoding.EncodeToString(derSig)

	// Verify
	err = VerifyEnclaveSignature(pubKeyPEM, payload)
	if err != nil {
		t.Errorf("expected signature to be valid, got error: %v", err)
	}
}

func TestVerifyEnclaveSignature_RawRS_Success(t *testing.T) {
	privKey, pubKeyPEM := GenerateTestKeyPair(t)

	payload := &domain.TransferPayload{
		ID:              "tx-12345",
		SenderAccount:   "1234567890",
		ReceiverAccount: "0987654321",
		AmountCents:     100000,
		Currency:        "USD",
		Timestamp:       time.Now().Truncate(time.Second),
	}

	canonicalBytes := CanonicalTransferBytes(payload)
	hash := sha256.Sum256(canonicalBytes)

	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash[:])
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	// Construct raw 64-byte signature
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// Pad r and s bytes to 32 bytes each
	rawSig := make([]byte, 64)
	copy(rawSig[32-len(rBytes):32], rBytes)
	copy(rawSig[64-len(sBytes):64], sBytes)

	payload.Signature = hex.EncodeToString(rawSig)

	// Verify
	err = VerifyEnclaveSignature(pubKeyPEM, payload)
	if err != nil {
		t.Errorf("expected raw R||S signature to be valid, got error: %v", err)
	}
}

func TestVerifyEnclaveSignature_TamperingRejection(t *testing.T) {
	privKey, pubKeyPEM := GenerateTestKeyPair(t)

	timestamp := time.Now().Truncate(time.Second)
	payload := &domain.TransferPayload{
		ID:              "tx-12345",
		SenderAccount:   "1234567890",
		ReceiverAccount: "0987654321",
		AmountCents:     100000,
		Currency:        "USD",
		Timestamp:       timestamp,
	}

	canonicalBytes := CanonicalTransferBytes(payload)
	hash := sha256.Sum256(canonicalBytes)

	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash[:])
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	derSig, _ := asn1.Marshal(struct{ R, S *big.Int }{R: r, S: s})
	payload.Signature = base64.StdEncoding.EncodeToString(derSig)

	// 1. Modify AmountCents (tampering)
	tamperedPayload := *payload
	tamperedPayload.AmountCents = 999999
	err = VerifyEnclaveSignature(pubKeyPEM, &tamperedPayload)
	if err == nil {
		t.Error("expected error when amount is tampered, but signature passed")
	}

	// 2. Modify SenderAccount
	tamperedPayload = *payload
	tamperedPayload.SenderAccount = "9999999999"
	err = VerifyEnclaveSignature(pubKeyPEM, &tamperedPayload)
	if err == nil {
		t.Error("expected error when sender account is tampered, but signature passed")
	}

	// 3. Modify ReceiverAccount
	tamperedPayload = *payload
	tamperedPayload.ReceiverAccount = "9999999999"
	err = VerifyEnclaveSignature(pubKeyPEM, &tamperedPayload)
	if err == nil {
		t.Error("expected error when receiver account is tampered, but signature passed")
	}

	// 4. Modify Currency
	tamperedPayload = *payload
	tamperedPayload.Currency = "EUR"
	err = VerifyEnclaveSignature(pubKeyPEM, &tamperedPayload)
	if err == nil {
		t.Error("expected error when currency is tampered, but signature passed")
	}

	// 5. Modify Timestamp
	tamperedPayload = *payload
	tamperedPayload.Timestamp = timestamp.Add(time.Hour)
	err = VerifyEnclaveSignature(pubKeyPEM, &tamperedPayload)
	if err == nil {
		t.Error("expected error when timestamp is tampered, but signature passed")
	}
}

func TestVerifyEnclaveSignature_InvalidInputs(t *testing.T) {
	_, pubKeyPEM := GenerateTestKeyPair(t)

	payload := &domain.TransferPayload{
		ID:              "tx-12345",
		SenderAccount:   "1234567890",
		ReceiverAccount: "0987654321",
		AmountCents:     100000,
		Currency:        "USD",
		Timestamp:       time.Now(),
		Signature:       "not-a-valid-sig",
	}

	// 1. Test empty key
	err := VerifyEnclaveSignature("", payload)
	if err == nil {
		t.Error("expected error on empty public key PEM")
	}

	// 2. Test nil payload
	err = VerifyEnclaveSignature(pubKeyPEM, nil)
	if err == nil {
		t.Error("expected error on nil payload")
	}

	// 3. Test empty signature
	emptySigPayload := *payload
	emptySigPayload.Signature = ""
	err = VerifyEnclaveSignature(pubKeyPEM, &emptySigPayload)
	if err == nil {
		t.Error("expected error on empty signature")
	}

	// 4. Test invalid PEM
	err = VerifyEnclaveSignature("-----BEGIN PUBLIC KEY-----\ninvalid\n-----END PUBLIC KEY-----", payload)
	if err == nil {
		t.Error("expected error on invalid PEM key")
	}
}
