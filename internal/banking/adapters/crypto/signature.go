package crypto

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"

	"github.com/democryst/go-api-management/internal/banking/domain"
)

// CanonicalTransferBytes serializes a TransferPayload into a deterministic, format-stable byte block.
// Time Complexity: O(N) where N is the total length of the accounts, amount, and currency strings.
// Space Complexity: O(N)
func CanonicalTransferBytes(payload *domain.TransferPayload) []byte {
	return []byte(fmt.Sprintf("%s:%s:%d:%s:%d",
		string(payload.SenderAccount),
		string(payload.ReceiverAccount),
		payload.AmountCents,
		payload.Currency,
		payload.Timestamp.Unix(),
	))
}

// VerifyEnclaveSignature parses a PEM-encoded ECDSA P-256 public key and cryptographically verifies the signature on a transfer payload.
// It supports both ASN.1 DER-encoded signatures and raw 64-byte (R || S) signature formats.
// Time Complexity: O(1) (asymmetric cryptographic verification is computationally bounded by P-256 parameters).
// Space Complexity: O(1)
func VerifyEnclaveSignature(publicKeyPEM string, payload *domain.TransferPayload) error {
	if publicKeyPEM == "" {
		return errors.New("empty public key PEM")
	}
	if payload == nil {
		return errors.New("nil transfer payload")
	}
	if payload.Signature == "" {
		return errors.New("empty transfer payload signature")
	}

	// 1. Parse PEM Block
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return errors.New("failed to parse PEM block containing the public key")
	}

	// 2. Parse PKIX Public Key
	pubKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key bytes: %w", err)
	}

	pubKey, ok := pubKeyInterface.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("public key is not an ECDSA public key")
	}

	// 3. Assert ECDSA P-256 standard
	if pubKey.Curve.Params().Name != "P-256" {
		return fmt.Errorf("unsupported curve: %s (only ECDSA P-256 is permitted for Secure Enclave compliance)", pubKey.Curve.Params().Name)
	}

	// 4. Compute SHA-256 hash of canonical transfer bytes
	canonicalBytes := CanonicalTransferBytes(payload)
	hash := sha256.Sum256(canonicalBytes)

	// 5. Decode Signature String (Try Hex first since it is a strict subset of Base64 characters)
	sigBytes, err := hex.DecodeString(payload.Signature)
	if err != nil {
		sigBytes, err = base64.StdEncoding.DecodeString(payload.Signature)
		if err != nil {
			return fmt.Errorf("signature is neither valid Base64 nor valid Hex: %w", err)
		}
	}

	// 6. Cryptographically Verify Signature (Supports DER or Raw R||S formats)
	var verified bool
	if len(sigBytes) == 64 {
		// Raw R || S signature
		rBytes := sigBytes[:32]
		sBytes := sigBytes[32:]
		r := new(big.Int).SetBytes(rBytes)
		s := new(big.Int).SetBytes(sBytes)
		verified = ecdsa.Verify(pubKey, hash[:], r, s)
	} else {
		// Try parsing as ASN.1 DER
		verified = ecdsa.VerifyASN1(pubKey, hash[:], sigBytes)
		if !verified {
			// Fallback: Check if it's formatted as standard asn1 SEQUENCE{R, S} in case VerifyASN1 is strict
			var dSig struct {
				R, S *big.Int
			}
			if _, err := asn1.Unmarshal(sigBytes, &dSig); err == nil {
				verified = ecdsa.Verify(pubKey, hash[:], dSig.R, dSig.S)
			}
		}
	}

	if !verified {
		return errors.New("cryptographic signature verification failed (data may have been tampered or signature is invalid)")
	}

	return nil
}
