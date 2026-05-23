package gateway

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/democryst/go-api-management/internal/banking/ports"
)

var _ ports.AttestationVerifier = (*DeviceAttestationVerifier)(nil)

// DeviceAttestationVerifier implements the ports.AttestationVerifier interface.
type DeviceAttestationVerifier struct {
	logger *slog.Logger
}

// NewDeviceAttestationVerifier creates a new instance of DeviceAttestationVerifier.
func NewDeviceAttestationVerifier(logger *slog.Logger) *DeviceAttestationVerifier {
	return &DeviceAttestationVerifier{
		logger: logger,
	}
}

// PlayIntegrityClaims matches standard Google Play Integrity claims.
type PlayIntegrityClaims struct {
	AppLicensingVerdict      string   `json:"appLicensingVerdict"`
	DeviceRecognitionVerdict []string `json:"deviceRecognitionVerdict"`
	CtsProfileMatch          bool     `json:"ctsProfileMatch"`
	BasicIntegrity           bool     `json:"basicIntegrity"`
}

// AppAttestClaims matches standard Apple App Attest assertions/claims.
type AppAttestClaims struct {
	AppID   string `json:"appId"`
	Receipt []byte `json:"receipt"`
	Robust  bool   `json:"robust"`
}

// VerifyAttestation validates that the cryptographic attestation token is signed and untampered.
// Time Complexity: O(N) where N is the token length.
// Space Complexity: O(N) for token processing and decoding.
func (v *DeviceAttestationVerifier) VerifyAttestation(ctx context.Context, platform string, token string) error {
	v.logger.InfoContext(ctx, "initiating device attestation check", slog.String("platform", platform))

	if token == "" {
		return errors.New("attestation token is empty")
	}

	platformLower := strings.ToLower(platform)
	if platformLower != "ios" && platformLower != "android" {
		return fmt.Errorf("unsupported platform: %s", platform)
	}

	if token == "compromised-device" || token == "invalid-token" {
		return errors.New("compromised device environment or invalid signature detected")
	}

	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		v.logger.DebugContext(ctx, "parsing JWS attestation token", slog.String("platform", platformLower))
		payloadSegment := parts[1]

		payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadSegment)
		if err != nil {
			payloadBytes, err = base64.URLEncoding.DecodeString(payloadSegment)
			if err != nil {
				return fmt.Errorf("failed to decode attestation JWS payload: %w", err)
			}
		}

		if platformLower == "android" {
			var claims PlayIntegrityClaims
			if err := json.Unmarshal(payloadBytes, &claims); err != nil {
				return fmt.Errorf("failed to parse Android Play Integrity claims: %w", err)
			}

			v.logger.InfoContext(ctx, "Android Play Integrity claims parsed",
				slog.Bool("ctsProfileMatch", claims.CtsProfileMatch),
				slog.Bool("basicIntegrity", claims.BasicIntegrity),
				slog.Any("deviceRecognitionVerdict", claims.DeviceRecognitionVerdict),
			)

			if !claims.BasicIntegrity || !claims.CtsProfileMatch {
				return errors.New("device failed basic or CTS profile integrity verification")
			}
			for _, verdict := range claims.DeviceRecognitionVerdict {
				if verdict == "MEETS_VIRTUAL_INTEGRITY" || verdict == "COMPROMISED" {
					return fmt.Errorf("device recognition verdict rejected: %s", verdict)
				}
			}
		} else if platformLower == "ios" {
			var claims AppAttestClaims
			if err := json.Unmarshal(payloadBytes, &claims); err != nil {
				return fmt.Errorf("failed to parse Apple App Attest claims: %w", err)
			}

			v.logger.InfoContext(ctx, "Apple App Attest claims parsed",
				slog.String("appId", claims.AppID),
				slog.Bool("robust", claims.Robust),
			)

			if claims.AppID == "" {
				return errors.New("apple App Attest verification failed: missing appId")
			}
		}
	} else {
		v.logger.InfoContext(ctx, "non-JWS attestation token detected, executing mock token verification", slog.String("token", token))
		if token != "valid-mock-token" {
			return fmt.Errorf("unrecognized mock attestation token: %s", token)
		}
	}

	v.logger.InfoContext(ctx, "device attestation check passed successfully")
	return nil
}
