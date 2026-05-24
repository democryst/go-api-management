# Secure Banking Gateway: Client-Side Integration Guide
## Mobile Developer Manual (iOS & Android)

This manual provides mobile application developers (iOS and Android) with standard integration specifications, sequential workflows, and concrete code blocks to securely bind devices and execute money transfers through our Secure Banking Gateway.

---

## 🏛️ 1. High-Level Cryptographic Architecture

To protect core banking ledgers against API Gateway compromises and Man-in-the-Middle (MitM) interceptions, the gateway operates on **Hardware-Backed Non-Repudiation**:

```text
 [ Mobile App / Secure Enclave ]              [ DMZ / Edge BFF ]                 [ Secure Intranet VPC ]
 
   1. Perform OIDC PKCE Login  ────────────────► [ Cookie Set ]
   
   2. Attest & Bind Public Key ────────────────► Verify Attestation ────────────► Save Public Key in DB
      (App Attest / Play Integrity)
      
   3. Sign Transaction Payload ────────────────► Token Swap ───────────────────► Retrieve Public Key,
      SHA256(sender:receiver:cents:unix)         (Issue Internal JWT)            Verify Enclave Signature,
      (Signed by Secure Enclave)                                                 Finalize Ledger Entry
```

Every money transfer request must carry a cryptographic signature generated *physically inside the mobile device's Secure Enclave/Keystore chip* using an ECDSA P-256 private key. The private key never leaves the physical silicon of the phone, meaning transactions cannot be forged even if our edge servers are completely compromised.

---

## 🔐 2. Phase 1: OIDC PKCE Authentication

All client-facing REST routes (except OIDC redirects) are protected by validated HTTP-only session cookies. Clients must authenticate using standard OIDC Authorization Code Flow with PKCE (RFC 7636):

1. **Initiate Login**: Direct the user to the OIDC login endpoint:
   `GET https://api.bank.com/auth/login?redirect_uri=app://callback`
   * The gateway will return a standard `302 Found` redirecting to the Identity Provider (Auth0), and sets a secure `session_id` cookie on the user's browser.

2. **Handle Callback**: Listen for the redirect in your mobile app schema, capture the `code` and `state` query parameters, and forward them back to the gateway:
   `GET https://api.bank.com/auth/callback?code=AUTH_CODE&state=STATE_VALUE`
   * The Gateway performs OIDC token exchange, destroys the transient session row (preventing replay attacks), sets a rotated, secure `refresh_token` cookie, and returns the JSON payload:
     ```json
     {
       "user": {
         "id": "auth0|user-12345",
         "email": "user@example.com"
       },
       "access_token": "eyJhbGciOi...",
       "expires_in": 86400
     }
     ```

3. **Session Maintenance**: Store the `access_token` in the device's secure Keychain. Set the `Authorization: Bearer <access_token>` header on all subsequent edge requests.

---

## 🛡️ 3. Phase 2: Secure Enclave Key Binding

Before initiating money transfers, the client app must generate an ECDSA P-256 key pair backed by hardware, extract a device integrity token, and bind the public key to their profile.

### Step 1: Generate Keypair & Attestation Token
* **iOS**: Generate an asymmetric keypair inside the **Secure Enclave** using `kSecAttrTokenIDSecureEnclave`. Generate an attestation certificate chain using the **Apple App Attest Service**.
* **Android**: Generate a hardware-backed keypair in the **Android Keystore** with strong box / TEE backing. Generate an integrity verdict token via the **Google Play Integrity API**.

### Step 2: Post Binding Request to Edge Gateway
Send the public key PEM and attestation credentials to the public gateway:
* **Route**: `POST /auth/register-device`
* **Headers**: `Authorization: Bearer <access_token>`
* **Payload**:
```json
{
  "id": "device-enclave-key-uuid-1",
  "public_key_pem": "-----BEGIN PUBLIC KEY-----\nMIIBMzCB7gYHKoZIzj0CAQY...\n-----END PUBLIC KEY-----",
  "algorithm": "ECDSA_P256",
  "platform": "ios",
  "attestation_token": "eyJoZWFkZXIiOi..."
}
```

---

## ⛓️ 4. Phase 3: Executing Signed Money-Transfers

To execute a transfer, construct the transaction JSON payload, canonicalize it into a deterministic byte block, sign the SHA-256 hash using the Secure Enclave private key, and post the request.

### Step 1: The Canonicalization Format
To prevent signature verification failure due to differences in whitespace or JSON ordering, client applications and backend engines must hash the **exact same** colon-separated byte block:
$$\text{Canonical Block} = \text{SenderAccount} + ":" + \text{ReceiverAccount} + ":" + \text{AmountCents} + ":" + \text{Currency} + ":" + \text{Timestamp}$$

* **Example Fields**:
  * `SenderAccount`: `"1234567890"`
  * `ReceiverAccount`: `"0987654321"`
  * `AmountCents`: `50000` (representing $500.00 to avoid float precision losses)
  * `Currency`: `"USD"`
  * `Timestamp`: `1716482400` (Unix Epoch timestamp in seconds)
* **Derived Canonical String**:
  `"1234567890:0987654321:50000:USD:1716482400"`

### Step 2: Sign and Send
Sign the SHA-256 hash of the derived canonical string. Convert the signature to **Base64 or Hexadecimal encoding** and set the `signature` field.
* **Route**: `POST /transfer/initiate`
* **Headers**: `Authorization: Bearer <access_token>`
* **Payload**:
```json
{
  "id": "transfer-transaction-uuid-1",
  "sender_account": "1234567890",
  "receiver_account": "0987654321",
  "amount_cents": 50000,
  "currency": "USD",
  "timestamp": "2026-05-23T21:00:00Z",
  "signature": "MEQCIB+5XzIq7l635B9w2o7..."
}
```

---

## 🍏 5. iOS Integration Blueprint (Swift)

Below is the Swift implementation to generate Secure Enclave keys, build the canonical string, sign it, and output standard ASN.1 DER base64-encoded signatures:

```swift
import Foundation
import Security

class SecureEnclaveSigner {
    private let keyTag = "com.bank.enclavekey"
    
    // Generates a hardware-backed ECDSA P-256 private key inside iOS Secure Enclave
    func generateEnclaveKey() throws -> SecKey {
        let flags: SecAccessControlCreateFlags = [.privateKeyUsage, .biometryAny]
        let access = SecAccessControlCreateWithFlags(
            kCFAllocatorDefault,
            kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
            flags,
            nil
        )!
        
        let attributes: [String: Any] = [
            kSecAttrKeyType as String: kSecAttrKeyTypeECSECPrimeRandom,
            kSecAttrKeySizeInBits as String: 256,
            kSecAttrTokenID as String: kSecAttrTokenIDSecureEnclave,
            kSecAttrPrivateKeyAttrs as String: [
                kSecAttrIsPermanent as String: true,
                kSecAttrApplicationTag as String: keyTag.data(using: .utf8)!,
                kSecAttrAccessControl as String: access
            ]
        ]
        
        var error: Unmanaged<CFError>?
        guard let privateKey = SecKeyCreateRandomKey(attributes as CFDictionary, &error) else {
            throw error!.takeRetainedValue() as Error
        }
        return privateKey
    }
    
    // Builds canonical byte block and signs it inside Secure Enclave using TouchID/FaceID
    func signTransfer(
        privateKey: SecKey,
        sender: String,
        receiver: String,
        cents: Int64,
        currency: String,
        timestamp: Int64
    ) throws -> String {
        // 1. Build canonical block matching backend specifications
        let canonicalString = "\(sender):\(receiver):\(cents):\(currency):\(timestamp)"
        guard let canonicalData = canonicalString.data(using: .utf8) else {
            throw NSError(domain: "SecureEnclaveSigner", code: 1, userInfo: [NSLocalizedDescriptionKey: "Encoding error"])
        }
        
        // 2. Hash payload using SHA-256
        var hash = [uint8](repeating: 0, count: Int(CC_SHA256_DIGEST_LENGTH))
        canonicalData.withUnsafeBytes {
            _ = CC_SHA256($0.baseAddress, CC_LONG(canonicalData.count), &hash)
        }
        let hashData = Data(hash)
        
        // 3. Cryptographically sign hash inside hardware chip
        var error: Unmanaged<CFError>?
        guard let signature = SecKeyCreateSignature(
            privateKey,
            .ecdsaSignatureMessageX962SHA256,
            hashData as CFData,
            &error
        ) else {
            throw error!.takeRetainedValue() as Error
        }
        
        // Secure Enclave SecKeyCreateSignature outputs ASN.1 DER signature bytes
        return (signature as Data).base64EncodedString()
    }
}
```

---

## 🤖 6. Android Integration Blueprint (Kotlin)

Below is the Kotlin implementation to generate Android Keystore backed keys, build the canonical string, sign the SHA-256 block, and output raw signature bytes:

```kotlin
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import java.security.KeyPairGenerator
import java.security.KeyStore
import java.security.Signature
import java.util.Base64

class AndroidKeystoreSigner {
    private val keyAlias = "BankEnclaveKeyAlias"

    // Generates ECDSA P-256 keypair locked in Android Trusted Execution Environment / StrongBox
    func generateHardwareKey() {
        val kpg = KeyPairGenerator.getInstance(
            KeyProperties.KEY_ALGORITHM_EC,
            "AndroidKeyStore"
        )
        val spec = KeyGenParameterSpec.Builder(
            keyAlias,
            KeyProperties.PURPOSE_SIGN or KeyProperties.PURPOSE_VERIFY
        ).setDigests(KeyProperties.DIGEST_SHA256)
          .setAlgorithmParameterSpec(ECGenParameterSpec("secp256r1"))
          .setUserAuthenticationRequired(true) // Requires biometrics (Fingerprint/Face)
          .build()
          
        kpg.initialize(spec)
        kpg.generateKeyPair()
    }

    // Signs canonical transaction payload via hardware chip
    func signTransfer(
        sender: String,
        receiver: String,
        cents: Long,
        currency: String,
        timestamp: Long
    ): String {
        val ks = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
        val entry = ks.getEntry(keyAlias, null) as KeyStore.PrivateKeyEntry
        val privateKey = entry.privateKey

        // 1. Build canonical block matching specifications
        val canonicalString = "$sender:$receiver:$cents:$currency:$timestamp"
        val dataBytes = canonicalString.toByteArray(Charsets.UTF_8)

        // 2. Cryptographically sign SHA256 inside Android TEE
        val s = Signature.getInstance("SHA256withECDSA")
        s.initSign(privateKey)
        s.update(dataBytes)
        val rawSignature = s.sign()

        // Outputs raw R || S signature bytes, base64 encoded
        return Base64.getEncoder().encodeToString(rawSignature)
    }
}
```

---

## 🚨 7. Error Taxonomy & Troubleshooting

When interacting with the Secure Banking Gateway, applications should handle the following HTTP error responses gracefully:

### Rate Limiting Telemetry Headers
Every response from the DMZ Edge Gateway carries these standard headers to let client applications dynamically track their rate-limit quotas:
* `X-RateLimit-Limit`: Maximum requests allowed within the sliding window.
* `X-RateLimit-Remaining`: Remaining request allowance inside the current window.
* `Retry-After`: Emitted on `429 Too Many Requests`, representing the sleep cooldown (in seconds) the mobile client must observe before retrying.

### Standard Response Taxonomy

| Status Code | Error Message | Scenario | Client Action |
| :--- | :--- | :--- | :--- |
| **`400 Bad Request`** | `"Invalid request body"` or `"Missing required parameters"` | The transfer payload or binding payload has malformed formats or missing properties. | Assert correct JSON parsing and parameters presence before calling edge APIs. |
| **`401 Unauthorized`** | `"unauthorized: missing token claims in request context"` | The edge `access_token` bearer JWT is expired or missing. | Initiate OIDC PKCE token refresh cycle or prompt user login. |
| **`403 Forbidden` (Attestation)** | `"device integrity check failed: ..."` | The device attestation token (App Attest/Play Integrity) failed verification, indicating tampered runtime or jailbreak. | Restrict execution. Warn the user that banking features are unavailable on modified operating systems. |
| **`403 Forbidden` (OPA Policy)** | `"access denied by system security policy"` | Dynamic OPA evaluation rejected the request (e.g. transfer >= $10,000 initiated by a user lacking the `transfer:high_value` scope). | Advise the user that the transaction bounds exceed standard authorization. Offer scope elevation or review amount. |
| **`429 Too Many Requests`** | `"rate limit exceeded, please retry later"` | Distributed Valkey Token Bucket rate limits are exceeded for this user or IP. | Capture the `Retry-After` header value, disable the submit button, and schedule an exponential backoff retry. |
| **`500 Internal Server Error`** | `"unauthorized: cryptographic secure enclave signature is invalid or tampered"` | The transaction's biometric signature failed verification against registered keys. | Prompt TouchID/FaceID again. Re-canonicalize the byte block to ensure no mismatch in timestamp or amount formatting. |
| **`503 Service Unavailable`** | `"transfer execution failed: ..."` | The edge BFF gateway experienced network timeouts to secure intranet zones, or the circuit breaker tripped. | Wait and retry after exponential backoff. Do not spam requests. |

---
*Manual Version: 1.1.0*
*Last Reviewed: 2026-05-24*

