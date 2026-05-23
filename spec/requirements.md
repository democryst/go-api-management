# Requirements Specification: Secure Banking Money-Transfer Gateway

This document represents the formal requirements specification for the high-performance Go Secure Banking API Gateway. All designs, network mappings, and code implementations must trace back to these rules.

---

## 🎯 1. Scope, Segmentation & Boundaries

To comply with global financial regulations (PCI-DSS and PSD2), our gateway is segmented into two distinct, firewalled VPC network environments:

```text
  [ PUBLIC INTERNET ]                   [ DMZ / EDGE ZONE ]                 [ SECURE INTRANET ZONE ]
                        
 ┌──────────────────┐                  ┌──────────────────┐                  ┌───────────────────┐
 │                  │                  │                  │                  │ Core Banking      │
 │  Mobile Client   │───( HTTPS 443 )─►│  API Gateway &   │───( mTLS Peer )─►│ Transfer Service  │
 │                  │                  │  BFF (Edge App)  │                  │ (Private IP Only) │
 └──────────────────┘                  └──────────────────┘                  └───────────────────┘
```

### In Scope
* **SR-01 (Multi-Zone Topology)**: Network isolation separating the public-facing API Gateway (DMZ Zone) from the Core Money-Transfer Service (Private VPC Intranet).
* **SR-02 (SafetyNet/App Attest)**: Verification of cryptographic device attestation tokens (Apple App Attest / Google Play Integrity) at the Edge DMZ before granting sessions.
* **SR-03 (Token Swap Pattern)**: Translation of public Edge BFF tokens into highly privileged, short-lived, mTLS-bound Internal JWTs containing transactional scopes.
* **SR-04 (Hardware Non-Repudiation)**: Directly verifying SHA-256 cryptographic signatures generated inside the client mobile device's Secure Enclave (iOS) or Keystore (Android), matching against registered public keys to execute transfers.
* **SR-05 (Financial Account PII Redaction)**: Auto-masking logging wrapper types to prevent leakages of bank account details, routing codes, or card values.

### Out of Scope
* Custom core banking ledger database engines.
* Bank staff back-office account creation workflows.
* External inter-bank clearing interfaces.

---

## 🛠️ 2. Functional Requirements

### FR-01: Device Attestation Checks (Edge Gate)
* Before establishing an active OIDC PKCE session, the DMZ Edge Gateway must validate the device integrity token (Play Integrity / App Attest) to verify that the client app is untampered, genuine, and operating on non-jailbroken hardware.

### FR-02: Public-to-Internal Token Swap (BFF Translator)
* The public mobile app communicates strictly using short-lived, low-privilege Edge BFF session tokens.
* When forwarding a money-transfer request, the DMZ Gateway verifies the BFF token, validates request parameters, and translates it by signing a highly privileged, short-lived **Internal JWT** containing specific transaction scopes (e.g. `scopes: ["transfer:execute"]`).
* The internal request must be forwarded over private peer lines using **Mutual TLS (mTLS)** to the Money Transfer Service.

### FR-03: Hardware Signature Verification (Non-Repudiation)
* For executing money transfers, the mobile app must sign the transaction payload using the hardware-backed private key stored in the device's Secure Enclave.
* The core Money Transfer service inside the Secure Intranet VPC must directly verify the payload's signature using the stored public key associated with that specific user account during device binding.
* Even if the DMZ edge app is completely compromised, unauthorized money transfers must fail if they lack a valid Secure Enclave signature.

---

## 🔒 3. Data Security & Handling Classifications

All financial and credential data must be handled according to strict classification tables:

| Data Type | Classification | Policy & Security Handling |
| :--- | :--- | :--- |
| **BFF Session Token** | Ephemeral Secret | Valid only at the Edge DMZ. Transmitted via secure, HTTP-only, `SameSite=Strict` browser session cookies. |
| **Internal JWT** | High Privilege Secret | Short-lived (< 1 minute), bound to mTLS peer connections. Never exposed to the public internet or logged. |
| **Enclave Private Key** | Hardware Secret | Stored strictly inside the client device's physical Secure Enclave. Never transmitted, logged, or printed. |
| **Enclave Public Key** | Cryptographic Asset | Persisted inside the Secure VPC Intranet database, tied strictly to the user identity profile. |
| **Bank Account Numbers / PII** | Highly Sensitive PII | Must be redacted in all system logging paths. Account numbers must mask all digits except the first two and last four (e.g. `12******3456`). |

---

## 📊 4. Non-Functional Requirements & SLAs

### NFR-01: Latency Budgets
* **SafetyNet/App Attest Check**: p95 < 80ms.
* **Token Swap Translation & Forwarding**: p95 < 15ms.
* **Hardware Enclave Signature Verification**: p95 < 5ms.

### NFR-02: Zero-Trust Inter-Zone Egress
* All connections leaving the DMZ Zone into the Secure Intranet VPC must utilize mutual TLS (mTLS) with standard CA-signed certificates and enforce strict IP whitelisting.

---

## 🧬 5. API Endpoints Specification

### DMZ Gateway Endpoints (Public)
* `POST /auth/register-device` - Binds a client device's Secure Enclave public key to an authenticated user profile.
* `POST /auth/login` - Initiates session, verifies device attestation, sets cookies.
* `POST /transfer/initiate` - Edge endpoint accepting money-transfer payloads with Secure Enclave signatures, swaps tokens, and forwards to Secure Intranet.

### Private Core Banking Endpoints (Secure VPC Only)
* `POST /private/transfers` - Receives internal JWTs, validates the Secure Enclave signature against stored keys, and executes the ledger transfer.

---
*Last Updated: 2026-05-23*
