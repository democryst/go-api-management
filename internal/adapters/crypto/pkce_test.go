package crypto

import (
    "strings"
    "testing"
)

// TestGenerateState asserts state entropy and padding-free properties.
func TestGenerateState(t *testing.T) {
    state, err := GenerateState()
    if err != nil {
        t.Fatalf("unexpected error generating state: %v", err)
    }

    if len(state) != 43 {
        t.Errorf("expected state to be 43 characters, got %d", len(state))
    }

    if strings.Contains(state, "=") {
        t.Errorf("state contains base64 padding '=': %s", state)
    }
}

// TestGenerateVerifier asserts verifier entropy and padding-free base64url compliance.
func TestGenerateVerifier(t *testing.T) {
    verifier, err := GenerateVerifier()
    if err != nil {
        t.Fatalf("unexpected error generating verifier: %v", err)
    }

    if len(verifier) != 43 {
        t.Errorf("expected verifier to be 43 characters, got %d", len(verifier))
    }

    if strings.Contains(verifier, "=") {
        t.Errorf("verifier contains base64 padding '=': %s", verifier)
    }
}

// TestDeriveChallenge asserts derived S256 challenge against known RFC 7636 Appendix B test vectors.
func TestDeriveChallenge(t *testing.T) {
    // Known RFC 7636 test values
    knownVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
    expectedChallenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

    derivedChallenge := DeriveChallenge(knownVerifier)
    if derivedChallenge != expectedChallenge {
        t.Errorf("derived challenge failed: expected %s, got %s", expectedChallenge, derivedChallenge)
    }
}
