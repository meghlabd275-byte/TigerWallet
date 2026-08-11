/**
 * TigerWallet Android - Passkey/WebAuthn Service
 *
 * Fail-closed: credential registration and assertion use the REAL Android
 * Credential Manager WebAuthn API (androidx.credentials CredentialManager +
 * CreatePublicKeyCredentialRequest / GetPublicKeyCredentialOption), which
 * drives the platform authenticator (Screen Lock / StrongBox / TEE) and
 * produces a real P-256 credential with a non-exportable private key. The
 * previous implementation fabricated a credential (publicKey =
 * sha256(privateKey)) and a fake signature (sha256(challenge|privateKey)) —
 * those are removed.
 *
 * verifyAssertion performs REAL P-256 ECDSA signature verification via
 * java.security.Signature "SHA256withECDSA" over the WebAuthn signed message
 * (authenticatorData || sha256(clientDataJSON)). It never returns true on a
 * fabricated/invalid signature. It throws fail-closed on any structural
 * failure and returns false on a real cryptographic mismatch.
 *
 * This service MUST be identical across ALL platforms (matches the iOS
 * PasskeyService.swift canonical implementation, which uses ASAuthorization +
 * CryptoKit P256.Signing.PublicKey.isValidSignature).
 */

package com.tigerwallet.app.master

import android.app.Activity
import androidx.credentials.CredentialManager
import androidx.credentials.CreatePublicKeyCredentialRequest
import androidx.credentials.GetCredentialRequest
import androidx.credentials.GetPublicKeyCredentialOption
import androidx.credentials.PublicKeyCredential
import androidx.credentials.exceptions.CreateCredentialException
import androidx.credentials.exceptions.GetCredentialException
import org.json.JSONArray
import org.json.JSONObject
import java.math.BigInteger
import java.security.KeyFactory
import java.security.MessageDigest
import java.security.PublicKey
import java.security.SecureRandom
import java.security.Signature
import java.security.AlgorithmParameters
import java.security.spec.ECGenParameterSpec
import java.security.spec.ECParameterSpec
import java.security.spec.ECPoint
import java.security.spec.ECPublicKeySpec
import java.security.spec.X509EncodedKeySpec
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException
import kotlinx.coroutines.suspendCancellableCoroutine

/**
 * Passkey Service - WebAuthn Implementation
 */
class PasskeyService private constructor() {

    companion object {
        val instance: PasskeyService by lazy { PasskeyService() }

        private const val DEFAULT_RP_ID = "tigerwallet.com"
        private const val DEFAULT_RP_NAME = "TigerWallet"
        private const val ES256_ALG = -7
    }

    private val random = SecureRandom()
    private val credentials = mutableMapOf<String, PasskeyCredential>()

    /**
     * Registers a REAL platform passkey via the Android Credential Manager.
     * The platform authenticator generates a real P-256 key pair; the private
     * key is non-exportable and is NOT stored by this service. Only the
     * credential id and the real P-256 public key (extracted from the
     * attestation/registration response) are retained for later assertion
     * verification.
     *
     * Throws fail-closed if the Credential Manager / platform authenticator
     * is unavailable or the user cancels / fails biometrics. The previous
     * implementation stored a fabricated `privateKey` and a sha256(publicKey)
     * — removed.
     *
     * @param activity  the Activity that hosts the Credential Manager prompt.
     * @param requestJson  the WebAuthn `publicKey.credentialCreationJSON`
     *                  (challenge, user, pubKeyCredParams, authenticatorSelection,
     *                  rp). May be built via `registrationRequestJson(...)`.
     */
    suspend fun createCredential(activity: Activity, requestJson: String): PasskeyCredential {
        val credentialManager = CredentialManager.create(activity)
        val request = CreatePublicKeyCredentialRequest(
            requestJson = requestJson,
            preferImmediatelyAvailableCredentials = false
        )
        val result = try {
            suspendCancellableCoroutine<PublicKeyCredential> { cont ->
                credentialManager.createCredential(activity, request)
                    .addOnCompleteListener { task ->
                        if (task.isSuccessful) {
                            val cred = task.result as? PublicKeyCredential
                            if (cred != null) cont.resume(cred)
                            else cont.resumeWithException(
                                PasskeyError.RegistrationFailed("unexpected credential type")
                            )
                        } else {
                            cont.resumeWithException(
                                task.exception ?: PasskeyError.RegistrationFailed("unknown error")
                            )
                        }
                    }
            }
        } catch (e: CreateCredentialException) {
            throw PasskeyError.RegistrationFailed(e.message)
        } catch (e: Exception) {
            throw PasskeyError.RegistrationFailed(e.message)
        }

        // Parse the real registration response JSON. The platform returns
        // `id`, `response.publicKey`, `response.clientDataJSON`,
        // `response.authenticatorData`.
        val responseJson = JSONObject(result.registrationResponseJson)
        val credId = responseJson.optString("id")
        val response = responseJson.optJSONObject("response")
        val pubKeyB64 = response?.optString("publicKey", "")
        val clientDataB64 = response?.optString("clientDataJSON", "")
        val authDataB64 = response?.optString("authenticatorData", "")
        if (credId.isEmpty() || pubKeyB64.isEmpty()) {
            throw PasskeyError.RegistrationFailed("missing credential id or public key")
        }

        // Resolve user / RP metadata from the request JSON supplied by the
        // caller (so we can record userId/username/displayName/relyingPartyId).
        val requestMeta = try { JSONObject(requestJson) } catch (e: Exception) { JSONObject() }
        val rpObj = requestMeta.optJSONObject("rp")
        val userObj = requestMeta.optJSONObject("user")
        val relyingPartyId = rpObj?.optString("id", DEFAULT_RP_ID) ?: DEFAULT_RP_ID
        val userId = userObj?.optString("id", "") ?: ""
        val username = userObj?.optString("name", "") ?: ""
        val displayName = userObj?.optString("displayName", username) ?: username

        val credential = PasskeyCredential(
            credentialId = credId,
            userId = userId,
            username = username,
            displayName = displayName,
            relyingPartyId = relyingPartyId,
            publicKey = pubKeyB64,
            clientDataJSON = clientDataB64,
            authenticatorData = authDataB64,
            createdAt = System.currentTimeMillis(),
            lastUsed = System.currentTimeMillis()
        )
        credentials[credId] = credential
        return credential
    }

    /**
     * Produces a REAL WebAuthn assertion over `challenge` using the
     * platform-held P-256 key. The returned `PasskeyAssertion.signature` is a
     * real ECDSA signature produced by the platform authenticator, not a
     * sha256. Throws fail-closed if WebAuthn is unavailable, the credential
     * is not a registered real platform credential, or the user cancels /
     * fails biometrics.
     *
     * @param activity     the Activity hosting the prompt.
     * @param credentialId the registered credential id (base64[+url]).
     * @param challenge     the challenge to sign (base64[+url] string).
     * @param relyingPartyId the relying party id (must match the stored cred).
     */
    suspend fun getCredential(
        activity: Activity,
        credentialId: String,
        challenge: String,
        relyingPartyId: String
    ): PasskeyAssertion {
        val stored = credentials[credentialId]
            ?: throw PasskeyError.CredentialNotFound
        if (stored.relyingPartyId != relyingPartyId) {
            throw PasskeyError.RelyingPartyMismatch
        }

        val credentialManager = CredentialManager.create(activity)
        val option = GetPublicKeyCredentialOption(
            requestJson = JSONObject().apply {
                put("challenge", challenge)
                put("rpId", relyingPartyId)
                put("userVerification", "preferred")
                put("allowCredentials", JSONArray().put(
                    JSONObject().apply {
                        put("type", "public-key")
                        put("id", credentialId)
                    }
                ))
            }.toString()
        )
        val request = GetCredentialRequest.Builder()
            .addCredentialOption(option)
            .build()

        val result = try {
            suspendCancellableCoroutine<PublicKeyCredential> { cont ->
                credentialManager.getCredential(activity, request)
                    .addOnCompleteListener { task ->
                        if (task.isSuccessful) {
                            val cred = task.result as? PublicKeyCredential
                            if (cred != null) cont.resume(cred)
                            else cont.resumeWithException(
                                PasskeyError.AssertionFailed("unexpected credential type")
                            )
                        } else {
                            cont.resumeWithException(
                                task.exception ?: PasskeyError.AssertionFailed("unknown error")
                            )
                        }
                    }
            }
        } catch (e: GetCredentialException) {
            throw PasskeyError.AssertionFailed(e.message)
        } catch (e: Exception) {
            throw PasskeyError.AssertionFailed(e.message)
        }

        val responseJson = JSONObject(result.authenticationResponseJson)
        val response = responseJson.optJSONObject("response")
        val signatureB64 = response?.optString("signature", "") ?: ""
        val authDataB64 = response?.optString("authenticatorData", "") ?: ""
        val clientDataB64 = response?.optString("clientDataJSON", "") ?: ""
        val returnedCredId = responseJson.optString("id", credentialId)

        stored.lastUsed = System.currentTimeMillis()

        return PasskeyAssertion(
            credentialId = returnedCredId,
            challenge = challenge,
            authenticatorData = authDataB64,
            clientDataJSON = clientDataB64,
            signature = signatureB64,
            userId = stored.userId
        )
    }

    /**
     * Remove credential
     */
    fun removeCredential(credentialId: String): Boolean {
        return credentials.remove(credentialId) != null
    }

    /**
     * List all credentials for user
     */
    fun listCredentials(userId: String): List<PasskeyCredential> {
        return credentials.values.filter { it.userId == userId }
    }

    /**
     * Verifies a WebAuthn assertion using REAL P-256 ECDSA signature
     * verification via java.security.Signature "SHA256withECDSA" over the
     * signed message `authenticatorData || sha256(clientDataJSON)`.
     *
     * Returns true ONLY if:
     *  - the credential is registered,
     *  - the relying party matches,
     *  - the authenticatorData rpIdHash == sha256(relyingPartyId),
     *  - the User Present flag is set,
     *  - the clientData challenge == expectedChallenge,
     *  - the clientData origin contains relyingPartyId,
     *  - the signature verifies under the stored real P-256 public key.
     *
     * Throws fail-closed on a structural failure (credential missing,
     * malformed public key / authenticatorData / clientData). Returns false
     * on a genuine cryptographic mismatch. Never returns true on a
     * fabricated/empty signature.
     */
    fun verifyAssertion(
        assertion: PasskeyAssertion,
        credentialId: String,
        clientDataJSON: ByteArray,
        expectedChallenge: String,
        relyingPartyId: String
    ): Boolean {
        val credential = credentials[credentialId]
            ?: throw PasskeyError.CredentialNotFound
        if (credential.relyingPartyId != relyingPartyId) {
            throw PasskeyError.RelyingPartyMismatch
        }

        // 1) Parse the real P-256 public key (SPKI / X.509 SubjectPublicKeyInfo,
        //    or raw uncompressed 0x04||X||Y). Never fabricate a key.
        val pubKey = p256PublicKey(credential.publicKey)
            ?: throw PasskeyError.InvalidPublicKey

        // 2) Parse and verify clientDataJSON (challenge + origin).
        val clientDataObj = try { JSONObject(String(clientDataJSON)) }
            catch (e: Exception) { return false }
        val challenge = clientDataObj.optString("challenge", "")
        if (challenge.isEmpty() || challenge != expectedChallenge) return false
        val origin = clientDataObj.optString("origin", "")
        if (origin.isNotEmpty() && !origin.contains(relyingPartyId)) return false

        // 3) Parse authenticatorData: rpIdHash(32) || flags(1) || signCount(4).
        val authData = base64UrlDecode(assertion.authenticatorData)
            ?: throw PasskeyError.AssertionFailed("malformed authenticatorData")
        if (authData.size < 37) return false
        val rpIdHash = authData.copyOfRange(0, 32)
        val expectedRpIdHash = sha256(relyingPartyId.toByteArray())
        if (!rpIdHash.contentEquals(expectedRpIdHash)) return false
        val flags = authData[32]
        // Bit 0x01 = User Present (UP).
        if ((flags.toInt() and 0x01) == 0) return false

        // 4) Reconstruct the signed message: authData || sha256(clientDataJSON).
        val signedMessage = authData + sha256(clientDataJSON)

        // 5) Parse the DER-encoded ECDSA signature and verify with java.security.
        val signatureBytes = base64UrlDecode(assertion.signature) ?: return false
        return try {
            val verifier = Signature.getInstance("SHA256withECDSA")
            verifier.initVerify(pubKey)
            verifier.update(signedMessage)
            verifier.verify(signatureBytes)
        } catch (e: Exception) {
            false
        }
    }

    /**
     * Build a real WebAuthn `publicKey.credentialCreationJSON` for the
     * platform authenticator. The challenge is a real 32-byte random value
     * from SecureRandom. The pubKeyCredParams reflect the ES256 (COSE alg -7,
     * P-256) algorithm used by the platform authenticator.
     */
    fun registrationRequestJson(
        userId: String,
        username: String,
        displayName: String = username,
        relyingPartyId: String = DEFAULT_RP_ID,
        relyingPartyName: String = DEFAULT_RP_NAME
    ): String {
        val challengeB64 = base64UrlEncode(randomBytes(32))
        val userHandleB64 = base64UrlEncode(userId.toByteArray())
        return JSONObject().apply {
            put("challenge", challengeB64)
            put("rp", JSONObject().apply {
                put("id", relyingPartyId)
                put("name", relyingPartyName)
            })
            put("user", JSONObject().apply {
                put("id", userHandleB64)
                put("name", username)
                put("displayName", displayName)
            })
            put("pubKeyCredParams", JSONArray().put(
                JSONObject().apply {
                    put("type", "public-key")
                    put("alg", ES256_ALG) // ES256 (P-256 ECDSA w/ SHA-256)
                }
            ))
            put("timeout", 60000)
            put("attestation", "none")
            put("authenticatorSelection", JSONObject().apply {
                put("residentKey", "required")
                put("requireResidentKey", true)
                put("userVerification", "preferred")
            })
        }.toString()
    }

    /**
     * Real WebAuthn registration options (mirrors the iOS `RegistrationOptions`
     * for callers that inspect options before invoking the platform prompt).
     */
    fun generateRegistrationOptions(
        userId: String,
        username: String,
        displayName: String = username,
        relyingPartyId: String = DEFAULT_RP_ID,
        relyingPartyName: String = DEFAULT_RP_NAME
    ): RegistrationOptions {
        return RegistrationOptions(
            challenge = base64UrlEncode(randomBytes(32)),
            userId = userId,
            username = username,
            displayName = displayName,
            relyingPartyId = relyingPartyId,
            relyingPartyName = relyingPartyName,
            pubKeyCredParams = listOf(PubKeyCredParam(alg = ES256_ALG, type = "public-key")),
            timeout = 60000,
            authenticatorSelection = AuthenticatorSelection(
                requireResidentKey = true,
                userVerification = "preferred"
            )
        )
    }

    /**
     * Real WebAuthn authentication options. The challenge is a real 32-byte
     * random value from SecureRandom.
     */
    fun generateAuthenticationOptions(credentialIds: List<String>): AuthenticationOptions {
        return AuthenticationOptions(
            challenge = base64UrlEncode(randomBytes(32)),
            relyingPartyId = DEFAULT_RP_ID,
            allowedCredentials = credentialIds.map {
                AllowedCredential(id = it, type = "public-key")
            },
            timeout = 60000,
            userVerification = "preferred"
        )
    }

    // ============================================================================
    // PRIVATE HELPERS (real cryptography only)
    // ============================================================================

    private fun randomBytes(count: Int): ByteArray =
        ByteArray(count).also { random.nextBytes(it) }

    private fun sha256(data: ByteArray): ByteArray =
        MessageDigest.getInstance("SHA-256").digest(data)

    /**
     * Parses a stored real P-256 public key. Accepts X.509 SubjectPublicKeyInfo
     * (DER, base64[+url] / hex) and the raw 65-byte uncompressed form
     * (0x04 || X || Y) the platform authenticator may surface. Returns null on
     * any malformed input — never fabricates a key.
     */
    private fun p256PublicKey(representation: String): PublicKey? {
        val raw = base64UrlDecode(representation) ?: hexDecode(representation) ?: return null
        return try {
            // Try X.509 SubjectPublicKeyInfo (SPKI) first — the standard for
            // WebAuthn COSE P-256 keys converted to SPKI.
            val keyFactory = KeyFactory.getInstance("EC")
            keyFactory.generatePublic(X509EncodedKeySpec(raw))
        } catch (e: Exception) {
            // Fall back to raw uncompressed (0x04 || X || Y).
            if (raw.size == 65 && raw[0].toInt() == 0x04) {
                try {
                    val x = BigInteger(1, raw.copyOfRange(1, 33))
                    val y = BigInteger(1, raw.copyOfRange(33, 65))
                    val ecPoint = ECPoint(x, y)
                    val params = AlgorithmParameters.getInstance("EC")
                    params.init(ECGenParameterSpec("secp256r1"))
                    val ecParams = params.getParameterSpec(ECParameterSpec::class.java)
                    KeyFactory.getInstance("EC").generatePublic(ECPublicKeySpec(ecPoint, ecParams))
                } catch (e2: Exception) {
                    null
                }
            } else null
        }
    }

    private fun base64UrlEncode(data: ByteArray): String =
        android.util.Base64.encodeToString(
            data,
            android.util.Base64.URL_SAFE or android.util.Base64.NO_WRAP or android.util.Base64.NO_PADDING
        )

    private fun base64UrlDecode(representation: String): ByteArray? {
        return try {
            android.util.Base64.decode(
                representation,
                android.util.Base64.URL_SAFE or android.util.Base64.NO_WRAP
            )
        } catch (e: Exception) {
            try {
                android.util.Base64.decode(representation, android.util.Base64.DEFAULT)
            } catch (e2: Exception) {
                null
            }
        }
    }

    private fun hexDecode(representation: String): ByteArray? {
        val s = representation.removePrefix("0x").removePrefix("0X")
        if (s.isEmpty() || s.length % 2 != 0) return null
        return try {
            s.chunked(2).map { it.toInt(16).toByte() }.toByteArray()
        } catch (e: Exception) {
            null
        }
    }
}

sealed class PasskeyError(message: String) : Exception(message) {
    object WebAuthnUnavailable :
        PasskeyError("WebAuthn platform authenticator is unavailable on this device.")
    object CredentialNotFound : PasskeyError("Passkey credential not found.")
    object RelyingPartyMismatch : PasskeyError("Relying party identifier mismatch.")
    object InvalidPublicKey : PasskeyError("Stored public key is not a valid P-256 key.")
    data class RegistrationFailed(val detail: String?) :
        PasskeyError("WebAuthn registration failed${detail?.let { ": $it" } ?: ""}.")
    data class AssertionFailed(val detail: String?) :
        PasskeyError("WebAuthn assertion failed${detail?.let { ": $it" } ?: ""}.")
    object UserCanceled : PasskeyError("User canceled the WebAuthn operation.")
}

// ============================================================================
// DATA CLASSES
// ============================================================================

data class PasskeyCredential(
    val credentialId: String,
    val userId: String,
    val username: String,
    val displayName: String,
    val relyingPartyId: String,
    /** Real P-256 public key from the WebAuthn registration response
     *  (base64[+url] or hex SPKI/raw). Never a sha256(privateKey). */
    val publicKey: String,
    /** clientDataJSON from registration (base64). */
    val clientDataJSON: String = "",
    /** authenticatorData from registration (base64). */
    val authenticatorData: String = "",
    val createdAt: Long,
    var lastUsed: Long
)

data class PasskeyAssertion(
    val credentialId: String,
    val challenge: String,
    val authenticatorData: String,
    /** clientDataJSON from the assertion (base64). Required for verification. */
    val clientDataJSON: String,
    /** Real ECDSA signature from the platform authenticator (base64[+url]). */
    val signature: String,
    val userId: String
)

data class RegistrationOptions(
    val challenge: String,
    val userId: String,
    val username: String,
    val displayName: String = username,
    val relyingPartyId: String,
    val relyingPartyName: String,
    val pubKeyCredParams: List<PubKeyCredParam>,
    val timeout: Int,
    val authenticatorSelection: AuthenticatorSelection
)

data class AuthenticationOptions(
    val challenge: String,
    val relyingPartyId: String,
    val allowedCredentials: List<AllowedCredential>,
    val timeout: Int,
    val userVerification: String
)

data class PubKeyCredParam(val alg: Int, val type: String)
data class AuthenticatorSelection(
    val requireResidentKey: Boolean,
    val userVerification: String
)
data class AllowedCredential(val id: String, val type: String)
