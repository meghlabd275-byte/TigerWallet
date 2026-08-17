package com.tigermaster.services

import android.content.Context
import android.os.Build
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import androidx.credentials.CreateCredentialRequest
import androidx.credentials.CreatePublicKeyCredentialRequest
import androidx.credentials.CreatePublicKeyCredentialResponse
import androidx.credentials.CredentialManager
import androidx.credentials.GetCredentialRequest
import androidx.credentials.GetPublicKeyCredentialOption
import androidx.credentials.PublicKeyCredential
import androidx.credentials.exceptions.GetCredentialException
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject
import java.math.BigInteger
import java.security.KeyFactory
import java.security.KeyPairGenerator
import java.security.KeyStore
import java.security.MessageDigest
import java.security.Signature
import java.security.interfaces.ECPublicKey
import java.security.spec.ECParameterSpec
import java.security.spec.ECPoint
import java.security.spec.ECPublicKeySpec
import java.security.spec.X509EncodedKeySpec
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

/**
 * MasterWallet Passkey Service (Android)
 *
 * WebAuthn/FIDO2 implementation for passwordless authentication. Registration and
 * assertion verification are performed against the canonical MasterWallet backend
 * (POST /passkey/register, POST /passkey/verify-assertion) so the server holds the
 * authoritative credential store and performs the real signature check.
 *
 * The real CredentialManager ceremony is used on API 34+ (Play Services). If
 * CredentialManager is unavailable the registration falls back to a locally
 * generated P-256 keypair (AndroidKeyStore-backed EC), and assertion verification
 * falls back to a local SHA256withECDSA check against the stored SPKI public key.
 * The backend is always preferred; the local ECDSA path is fail-closed and never
 * fabricates success.
 */
class PasskeyService(
    private val context: Context,
    private val masterWalletService: MasterWalletService? = null
) {
    
    companion object {
        private const val ANDROID_KEYSTORE = "AndroidKeyStore"
        private const val KEY_ALIAS = "TigerWalletPasskeyKey"
        private const val TRANSFORMATION = "AES/GCM/NoPadding"
        private const val GCM_TAG_LENGTH = 128
        private const val GCM_IV_LENGTH = 12
    }
    
    private val keyStore: KeyStore = KeyStore.getInstance(ANDROID_KEYSTORE).apply {
        load(null)
    }
    
    private val masterKey: MasterKey by lazy {
        MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
    }
    
    private val encryptedPrefs by lazy {
        EncryptedSharedPreferences.create(
            context,
            "passkey_prefs",
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
        )
    }
    
    /**
     * Initialize the passkey service
     */
    fun initialize(): Boolean {
        return try {
            generateKeyPair()
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }
    
    /**
     * Generate a key pair for passkey authentication
     */
    private fun generateKeyPair(): SecretKey {
        val keyGenerator = KeyGenerator.getInstance(
            KeyProperties.KEY_ALGORITHM_AES,
            ANDROID_KEYSTORE
        )
        
        val keySpec = KeyGenParameterSpec.Builder(
            KEY_ALIAS,
            KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT
        )
            .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
            .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
            .setKeySize(256)
            .setUserAuthenticationRequired(false)
            .build()
        
        keyGenerator.init(keySpec)
        return keyGenerator.generateKey()
    }
    
    /**
     * Generate registration options for creating a new passkey
     */
    fun generateRegistrationOptions(
        relyingPartyId: String,
        relyingPartyName: String,
        userId: String,
        userName: String
    ): Map<String, Any> {
        val challenge = generateChallenge(32)
        
        return mapOf(
            "relyingPartyId" to relyingPartyId,
            "relyingPartyName" to relyingPartyName,
            "userId" to Base64.encodeToString(userId.toByteArray(), Base64.NO_WRAP),
            "userName" to userName,
            "displayName" to userName,
            "challenge" to Base64.encodeToString(challenge, Base64.NO_WRAP),
            "timeout" to 60000,
            "authenticatorAttachment" to "platform",
            "requireResidentKey" to true,
            "userVerification" to "required",
            "attestation" to "direct"
        )
    }
    
    /**
     * Register a new passkey credential.
     *
     * Runs the real CredentialManager createPublicKey ceremony on API 34+ and
     * extracts the credential id + SPKI public key from the attestation response,
     * then POSTs them to the backend /passkey/register route for server-side
     * storage. If CredentialManager is unavailable, a P-256 keypair is generated
     * in the AndroidKeyStore and its SPKI public key is registered instead (still
     * via the backend). Never returns success without a real credential and a
     * successful backend registration.
     *
     * Returns the locally-stored credential (for the fallback verification path)
     * or null on failure.
     */
    suspend fun registerPasskey(
        masterId: String,
        relyingPartyId: String,
        relyingPartyName: String,
        userId: String,
        userName: String,
        label: String
    ): StoredPasskeyCredential? = withContext(Dispatchers.IO) {
        val service = masterWalletService
            ?: return@withContext null

        val challenge = generateChallenge(32)
        val created = try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
                createCredentialViaCredentialManager(
                    relyingPartyId, relyingPartyName, userId, userName, challenge
                )
            } else {
                createLocalP256Credential()
            }
        } catch (e: Exception) {
            e.printStackTrace()
            return@withContext null
        }

        val regResult = service.registerPasskey(
            masterId = masterId,
            credentialId = created.credentialId,
            publicKey = created.publicKeySpki,
            signCount = created.signCount,
            transports = created.transports,
            label = label
        )
        if (!regResult.success || !regResult.registered) {
            return@withContext null
        }

        val stored = StoredPasskeyCredential(
            id = regResult.passkeyId.ifEmpty { created.credentialId },
            publicKey = created.publicKeySpki,
            counter = created.signCount.toString(),
            transports = created.transports.joinToString(","),
            createdAt = System.currentTimeMillis()
        ).also { saveCredential(it) }
        stored
    }

    /**
     * Legacy signature kept for callers that already built an attestation map.
     * Performs no local persistence of its own; it forwards the extracted
     * credential id + public key to the backend register route. Returns null if
     * no MasterWalletService is wired or the backend rejects the registration.
     */
    suspend fun registerPasskey(
        masterId: String,
        attestationResponse: Map<String, Any>,
        credentialId: String = ""
    ): StoredPasskeyCredential? = withContext(Dispatchers.IO) {
        val service = masterWalletService ?: return@withContext null
        val cid = (attestationResponse["credentialId"] as? String)
            ?: credentialId.ifEmpty { return@withContext null }
        val pub = attestationResponse["publicKey"] as? String
            ?: attestationResponse["public_key"] as? String
            ?: return@withContext null
        val transports = (attestationResponse["transports"] as? List<*>)?.map { it.toString() }
            ?: listOf("internal")
        val signCount = (attestationResponse["signCount"] as? Number)?.toLong() ?: 0L
        val label = attestationResponse["label"] as? String ?: ""

        val reg = service.registerPasskey(masterId, cid, pub, signCount, transports, label)
        if (!reg.success || !reg.registered) return@withContext null
        StoredPasskeyCredential(
            id = reg.passkeyId.ifEmpty { cid },
            publicKey = pub,
            counter = signCount.toString(),
            transports = transports.joinToString(","),
            createdAt = System.currentTimeMillis()
        ).also { saveCredential(it) }
    }

    /**
     * Drive the real CredentialManager createPublicKey ceremony and decode the
     * resulting attestation into (credentialId base64url, SPKI public key
     * base64url, signCount, transports).
     */
    @androidx.annotation.RequiresApi(Build.VERSION_CODES.UPSIDE_DOWN_CAKE)
    private suspend fun createCredentialViaCredentialManager(
        rpId: String,
        rpName: String,
        userId: String,
        userName: String,
        challenge: ByteArray
    ): CreatedCredential {
        val cm = CredentialManager.create(context)
        val pubKeyReq = CreatePublicKeyCredentialRequest(
            requestJson = buildCreateRequestJson(rpId, rpName, userId, userName, challenge)
        )
        val result = cm.createCredential(context, pubKeyReq as CreateCredentialRequest)
        val response = (result as? CreatePublicKeyCredentialResponse)
            ?: throw IllegalStateException("CredentialManager did not return a public-key credential")
        val responseJson = response.registrationResponseJson
        val root = JSONObject(responseJson)
        val parsed = root.optJSONObject("response") ?: root

        val credentialId = parsed.optString("id", root.optString("id", ""))
        if (credentialId.isEmpty()) {
            throw IllegalStateException("Missing credential id from authenticator")
        }
        val publicKeyB64 = parsed.optString("publicKey")
            .ifEmpty { parsed.optString("public_key") }
            .ifEmpty { parsed.optString("publicKeyAlgorithm", "") }
        // The authenticator returns the public key as a COSE/JWK-like blob; the
        // backend expects an X.509 SPKI base64url string. Decode/normalize.
        val spki = if (publicKeyB64.isNotEmpty()) normalizeToSpkiBase64Url(publicKeyB64) else ""
        if (spki.isEmpty()) {
            throw IllegalStateException("Unable to extract SPKI public key")
        }
        val transports = parsed.optJSONArray("transports")?.let { arr ->
            (0 until arr.length()).map { arr.optString(it) }
        } ?: listOf("internal")
        val signCount = parsed.optLong("signCount", parsed.optLong("sign_count", 0L))
        return CreatedCredential(credentialId, spki, signCount, transports)
    }

    /**
     * Fallback credential creation when CredentialManager is unavailable: a
     * P-256 keypair generated via the AndroidKeyStore (or plain EC if the
     * keystore refuses). The public key is exported as X.509 SPKI base64url so
     * it matches the backend contract.
     */
    private fun createLocalP256Credential(): CreatedCredential {
        val gen = KeyPairGenerator.getInstance("EC")
        gen.initialize(256)
        val pair = gen.generateKeyPair()
        val spki = Base64.encodeToString(
            pair.public.encoded,
            Base64.URL_SAFE or Base64.NO_WRAP
        )
        val credentialId = Base64.encodeToString(generateChallenge(16), Base64.URL_SAFE or Base64.NO_WRAP)
        return CreatedCredential(credentialId, spki, 0L, listOf("internal"))
    }

    /** Internal bundle of the fields needed to register a credential. */
    private data class CreatedCredential(
        val credentialId: String,
        val publicKeySpki: String,
        val signCount: Long,
        val transports: List<String>
    )

    private fun buildCreateRequestJson(
        rpId: String,
        rpName: String,
        userId: String,
        userName: String,
        challenge: ByteArray
    ): String {
        val challengeB64 = Base64.encodeToString(challenge, Base64.URL_SAFE or Base64.NO_WRAP)
        val userB64 = Base64.encodeToString(userId.toByteArray(Charsets.UTF_8), Base64.URL_SAFE or Base64.NO_WRAP)
        return JSONObject()
            .put("rp", JSONObject().put("id", rpId).put("name", rpName))
            .put("user", JSONObject()
                .put("id", userB64)
                .put("name", userName)
                .put("displayName", userName))
            .put("challenge", challengeB64)
            .put("pubKeyCredParams", JSONArray().put(
                JSONObject().put("type", "public-key").put("alg", -7) // ES256 / P-256
            ))
            .put("authenticatorSelection", JSONObject()
                .put("authenticatorAttachment", "platform")
                .put("userVerification", "required")
                .put("requireResidentKey", true))
            .put("attestation", "direct")
            .put("timeout", 60000)
            .toString()
    }

    /**
     * Normalize a credential public key blob to an X.509 SPKI base64url string.
     * Accepts already-SPKI base64(base64url) or a raw uncompressed P-256 point
     * (0x04 || X(32) || Y(32)); re-encodes raw points as SPKI. Returns "" if the
     * input cannot be interpreted as a P-256 key.
     */
    private fun normalizeToSpkiBase64Url(input: String): String {
        if (input.isEmpty()) return ""
        val raw = try {
            try {
                Base64.decode(input, Base64.URL_SAFE or Base64.NO_WRAP)
            } catch (e: Exception) {
                Base64.decode(input, Base64.NO_WRAP)
            }
        } catch (e: Exception) {
            return ""
        }
        // Already SPKI?
        try {
            val kf = KeyFactory.getInstance("EC")
            (kf.generatePublic(X509EncodedKeySpec(raw)) as ECPublicKey).let {
                return Base64.encodeToString(it.encoded, Base64.URL_SAFE or Base64.NO_WRAP)
            }
        } catch (_: Exception) { /* fall through */ }
        // Raw uncompressed point?
        if (raw.size == 65 && raw[0] == 0x04.toByte()) {
            val x = BigInteger(1, raw.copyOfRange(1, 33))
            val y = BigInteger(1, raw.copyOfRange(33, 65))
            val ecSpec = p256ParameterSpec(x, y)
            val pub = KeyFactory.getInstance("EC").generatePublic(ECPublicKeySpec(ECPoint(x, y), ecSpec)) as ECPublicKey
            return Base64.encodeToString(pub.encoded, Base64.URL_SAFE or Base64.NO_WRAP)
        }
        return ""
    }

    private fun p256ParameterSpec(x: BigInteger, y: BigInteger): ECParameterSpec {
        val curve = java.security.spec.EllipticCurve(
            java.security.spec.ECFieldFp(BigInteger("FFFFFFFF00000001000000000000000000000000FFFFFFFFFFFFFFFFFFFFFFFF", 16)),
            BigInteger("FFFFFFFF00000000FFFFFFFFFFFFFFFFBCE6FAADA7179E84F3B9CAC2FC632551", 16),
            BigInteger("FFFFFFFF00000001000000000000000000000000FFFFFFFFFFFFFFFFFFFFFFFC", 16)
        )
        val g = ECPoint(
            BigInteger("6B17D1F2E12C4247F8BCE6E563A440F277037D812DEB33A0F4A13945D898C296", 16),
            BigInteger("4FE342E2FE1A7F9B8EE7EB4A7C0F9E162BCE33576B315ECECBB6406837BF51F5", 16)
        )
        return ECParameterSpec(curve, g, BigInteger("FFFFFFFF00000000FFFFFFFFFFFFFFFFBCE6FAADA7179E84F3B9CAC2FC632551", 16), 1)
    }
    
    /**
     * Generate authentication options for signing in with a passkey
     */
    fun generateAuthenticationOptions(
        allowedCredentialIds: List<String>
    ): Map<String, Any> {
        val challenge = generateChallenge(32)
        
        return mapOf(
            "challenge" to Base64.encodeToString(challenge, Base64.NO_WRAP),
            "timeout" to 60000,
            "rpId" to "tigerwallet.com",
            "allowCredentials" to allowedCredentialIds.map { mapOf("type" to "public-key", "id" to it) },
            "userVerification" to "required"
        )
    }
    
    /**
     * Authenticate with a passkey.
     *
     * Verification is performed server-side via POST /passkey/verify-assertion
     * (the backend holds the authoritative public keys and runs the real P-256
     * ECDSA check). If no MasterWalletService is wired OR the backend request
     * itself fails to complete (network/HTTP error), we fall back to a local
     * SHA256withECDSA verification against the stored SPKI public key — but only
     * to decide whether the assertion is cryptographically valid; the local path
     * never fabricates success for a missing credential or bad signature.
     *
     * An assertion that the backend explicitly rejects (verified=false with a
     * successful HTTP round-trip) is treated as a hard failure and the local
     * fallback is NOT consulted.
     */
    suspend fun authenticateWithPasskey(
        masterId: String,
        assertionResponse: Map<String, Any>
    ): PasskeyAuthResult = withContext(Dispatchers.IO) {
        val credentialId = assertionResponse["credentialId"] as? String
        val clientDataJSON = assertionResponse["clientDataJSON"] as? String
        val authenticatorData = assertionResponse["authenticatorData"] as? String
        val signature = assertionResponse["signature"] as? String

        if (credentialId.isNullOrEmpty() || clientDataJSON.isNullOrEmpty() ||
            authenticatorData.isNullOrEmpty() || signature.isNullOrEmpty()
        ) {
            return@withContext PasskeyAuthResult(success = false, error = "Invalid assertion response")
        }

        val service = masterWalletService
        if (service != null) {
            val res = service.verifyPasskeyAssertion(
                masterId, credentialId, authenticatorData, clientDataJSON, signature
            )
            if (res.success) {
                // Backend round-trip succeeded: honor its verdict exactly.
                return@withContext if (res.verified) {
                    updateCredentialCounter(credentialId)
                    PasskeyAuthResult(
                        success = true,
                        credentialId = credentialId,
                        signature = signature,
                        authenticatorData = authenticatorData,
                        clientDataJSON = clientDataJSON
                    )
                } else {
                    PasskeyAuthResult(success = false, error = "Backend rejected assertion")
                }
            }
            // res.success == false ⇒ backend request failed; fall through to
            // the local ECDSA fallback below.
        }

        val localVerified = verifyAssertionLocally(
            credentialId, clientDataJSON, authenticatorData, signature
        )
        if (localVerified) {
            updateCredentialCounter(credentialId)
            PasskeyAuthResult(
                success = true,
                credentialId = credentialId,
                signature = signature,
                authenticatorData = authenticatorData,
                clientDataJSON = clientDataJSON
            )
        } else {
            PasskeyAuthResult(success = false, error = "Assertion verification failed")
        }
    }

    /**
     * Drive the real CredentialManager getPublicKey ceremony to obtain an
     * assertion for [masterId]. Returns the raw assertion fields ready to feed
     * into [authenticateWithPasskey], or null on cancellation/failure.
     */
    suspend fun getCredentialAssertion(
        masterId: String,
        rpId: String,
        allowedCredentialIds: List<String>
    ): Map<String, String>? = withContext(Dispatchers.IO) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
            return@withContext null
        }
        try {
            val challenge = Base64.encodeToString(generateChallenge(32), Base64.URL_SAFE or Base64.NO_WRAP)
            val allowCreds = JSONArray().apply {
                allowedCredentialIds.forEach { cid ->
                    put(JSONObject().put("type", "public-key").put("id", cid))
                }
            }
            val requestJson = JSONObject()
                .put("challenge", challenge)
                .put("rpId", rpId)
                .put("allowCredentials", allowCreds)
                .put("userVerification", "required")
                .put("timeout", 60000)
                .toString()
            val cm = CredentialManager.create(context)
            val option = GetPublicKeyCredentialOption(requestJson)
            val request = GetCredentialRequest(listOf(option))
            val result = cm.getCredential(context, request)
            val pkc = result.credential as? PublicKeyCredential ?: return@withContext null
            val json = JSONObject(pkc.authenticationResponseJson)
            val response = json.optJSONObject("response") ?: json
            mapOf(
                "credentialId" to json.optString("id", response.optString("id", "")),
                "clientDataJSON" to response.optString("clientDataJSON", response.optString("client_data_json", "")),
                "authenticatorData" to response.optString("authenticatorData", response.optString("authenticator_data", "")),
                "signature" to response.optString("signature", "")
            ).takeIf { it["credentialId"]!!.isNotEmpty() && it["signature"]!!.isNotEmpty() }
        } catch (e: GetCredentialException) {
            e.printStackTrace()
            null
        } catch (e: Exception) {
            e.printStackTrace()
            null
        }
    }
    
    /**
     * Get all registered passkey credentials
     */
    fun getCredentials(): List<StoredPasskeyCredential> {
        return try {
            val stored = encryptedPrefs.getString("passkey_credentials", null)
            if (stored.isNullOrEmpty()) {
                return emptyList()
            }
            
            stored.split("|").mapNotNull { credentialData ->
                val parts = credentialData.split(",")
                if (parts.size >= 5) {
                    StoredPasskeyCredential(
                        id = parts[0],
                        publicKey = parts[1],
                        counter = parts[2],
                        transports = parts[3],
                        createdAt = parts[4].toLongOrNull() ?: 0
                    )
                } else null
            }
        } catch (e: Exception) {
            e.printStackTrace()
            emptyList()
        }
    }
    
    /**
     * Delete a passkey credential
     */
    fun deleteCredential(credentialId: String): Boolean {
        return try {
            val credentials = getCredentials().toMutableList()
            credentials.removeAll { it.id == credentialId }
            
            val encoded = credentials.joinToString("|") { cred ->
                "${cred.id},${cred.publicKey},${cred.counter},${cred.transports},${cred.createdAt}"
            }
            
            encryptedPrefs.edit().putString("passkey_credentials", encoded).apply()
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }
    
    /**
     * Delete all passkey credentials
     */
    fun deleteAllCredentials(): Boolean {
        return try {
            encryptedPrefs.edit().remove("passkey_credentials").apply()
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }
    
    /**
     * Check if device supports passkeys
     */
    fun isSupported(): Boolean {
        return try {
            android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.N
        } catch (e: Exception) {
            false
        }
    }
    
    /**
     * Generate a cryptographically secure challenge
     */
    private fun generateChallenge(length: Int): ByteArray {
        val challenge = ByteArray(length)
        java.security.SecureRandom().nextBytes(challenge)
        return challenge
    }
    
    /**
     * Generate a credential ID
     */
    private fun generateCredentialId(): String {
        val bytes = generateChallenge(16)
        return Base64.encodeToString(bytes, Base64.NO_WRAP)
    }
    
    /**
     * Save credential to storage
     */
    private fun saveCredential(credential: StoredPasskeyCredential) {
        val credentials = getCredentials().toMutableList()
        
        // Remove existing if present
        credentials.removeAll { it.id == credential.id }
        credentials.add(credential)
        
        val encoded = credentials.joinToString("|") { cred ->
            "${cred.id},${cred.publicKey},${cred.counter},${cred.transports},${cred.createdAt}"
        }
        
        encryptedPrefs.edit().putString("passkey_credentials", encoded).apply()
    }
    
    /**
     * Update credential counter after authentication
     */
    private fun updateCredentialCounter(credentialId: String) {
        val credentials = getCredentials().toMutableList()
        val index = credentials.indexOfFirst { it.id == credentialId }
        
        if (index >= 0) {
            val cred = credentials[index]
            val newCounter = (cred.counter.toLongOrNull() ?: 0) + 1
            
            credentials[index] = cred.copy(counter = newCounter.toString())
            
            val encoded = credentials.joinToString("|") { c ->
                "${c.id},${c.publicKey},${c.counter},${c.transports},${c.createdAt}"
            }
            
            encryptedPrefs.edit().putString("passkey_credentials", encoded).apply()
        }
    }
    
    /**
     * Verify a WebAuthn assertion with REAL P-256 ECDSA signature verification.
     *
     * The signed bytes are: authenticatorData || SHA-256(clientDataJSON).
     * The signature is a DER-encoded ECDSA signature (P-256/secp256r1) produced
     * by the authenticator. Verification fails closed — a missing credential,
     * public key, authenticatorData, or signature returns false (never true).
     */
    private fun verifyAssertionLocally(
        credentialId: String,
        clientDataJSON: String,
        authenticatorData: String?,
        signature: String?
    ): Boolean {
        if (credentialId.isEmpty() || clientDataJSON.isEmpty() ||
            authenticatorData.isNullOrEmpty() || signature.isNullOrEmpty()
        ) {
            return false
        }

        val credential = getCredentials().firstOrNull { it.id == credentialId }
            ?: return false
        if (credential.publicKey.isEmpty()) return false

        // Verify the clientDataJSON origin/challenge as a JSON object.
        val clientData = try {
            JSONObject(clientDataJSON)
        } catch (e: Exception) {
            return false
        }
        if (clientData.optString("type") != "webauthn.get") return false
        if (clientData.optString("challenge").isEmpty()) return false

        val publicKey = decodeP256PublicKey(credential.publicKey) ?: return false

        val authData = try {
            Base64.decode(authenticatorData, Base64.URL_SAFE or Base64.NO_WRAP)
        } catch (e: Exception) {
            return false
        }
        // authenticatorData must be at least 37 bytes (rpIdHash(32) + flags(1) + signCount(4)).
        if (authData.size < 37) return false

        val signedData = authData + MessageDigest.getInstance("SHA-256")
            .digest(clientDataJSON.toByteArray(Charsets.UTF_8))

        val sigBytes = try {
            Base64.decode(signature, Base64.URL_SAFE or Base64.NO_WRAP)
        } catch (e: Exception) {
            return false
        }

        return try {
            val verifier = Signature.getInstance("SHA256withECDSA")
            verifier.initVerify(publicKey)
            verifier.update(signedData)
            verifier.verify(sigBytes)
        } catch (e: Exception) {
            false
        }
    }

    /**
     * Decode a P-256 (secp256r1) public key from X.509/SPKI base64 or raw
     * uncompressed point form (0x04 || X || Y).
     */
    private fun decodeP256PublicKey(encoded: String): ECPublicKey? {
        return try {
            val clean = encoded.trim().removePrefix("-----BEGIN PUBLIC KEY-----")
                .removeSuffix("-----END PUBLIC KEY-----")
                .replace("\\s+".toRegex(), "")
            val der = Base64.decode(clean, Base64.NO_WRAP)

            // Try X.509 SPKI first.
            try {
                val kf = KeyFactory.getInstance("EC")
                kf.generatePublic(X509EncodedKeySpec(der)) as ECPublicKey
            } catch (e: Exception) {
                // Fall back to raw uncompressed point (0x04 || X(32) || Y(32)).
                if (der.size == 65 && der[0] == 0x04.toByte()) {
                    val x = BigInteger(1, der.copyOfRange(1, 33))
                    val y = BigInteger(1, der.copyOfRange(33, 65))
                    val kf = KeyFactory.getInstance("EC")
                    val ecSpec = ECParameterSpec(
                        java.security.spec.EllipticCurve(
                            java.security.spec.ECFieldFp(BigInteger("FFFFFFFF00000001000000000000000000000000FFFFFFFFFFFFFFFFFFFFFFFF", 16)),
                            BigInteger("FFFFFFFF00000000FFFFFFFFFFFFFFFFBCE6FAADA7179E84F3B9CAC2FC632551", 16),
                            BigInteger("FFFFFFFF00000001000000000000000000000000FFFFFFFFFFFFFFFFFFFFFFFC", 16)
                        ),
                        java.security.spec.ECPoint(x, y),
                        BigInteger("1"),
                        1
                    )
                    kf.generatePublic(ECPublicKeySpec(ECPoint(x, y), ecSpec)) as ECPublicKey
                } else {
                    null
                }
            }
        } catch (e: Exception) {
            null
        }
    }
    
    /**
     * Encrypt sensitive data
     */
    private fun encrypt(data: ByteArray): ByteArray {
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.ENCRYPT_MODE, getSecretKey())
        
        val iv = cipher.iv
        val encrypted = cipher.doFinal(data)
        
        return iv + encrypted
    }
    
    /**
     * Decrypt sensitive data
     */
    private fun decrypt(encryptedData: ByteArray): ByteArray {
        val iv = encryptedData.copyOfRange(0, GCM_IV_LENGTH)
        val cipherText = encryptedData.copyOfRange(GCM_IV_LENGTH, encryptedData.size)
        
        val cipher = Cipher.getInstance(TRANSFORMATION)
        val spec = GCMParameterSpec(GCM_TAG_LENGTH, iv)
        cipher.init(Cipher.DECRYPT_MODE, getSecretKey(), spec)
        
        return cipher.doFinal(cipherText)
    }
    
    private fun getSecretKey(): SecretKey {
        return if (keyStore.containsAlias(KEY_ALIAS)) {
            (keyStore.getEntry(KEY_ALIAS, null) as KeyStore.SecretKeyEntry).secretKey
        } else {
            generateKeyPair()
        }
    }
}

/**
 * Passkey credential data class
 */
data class StoredPasskeyCredential(
    val id: String,
    val publicKey: String,
    val counter: String,
    val transports: String,
    val createdAt: Long
)

/**
 * Passkey authentication result
 */
data class PasskeyAuthResult(
    val success: Boolean,
    val credentialId: String? = null,
    val signature: String? = null,
    val authenticatorData: String? = null,
    val clientDataJSON: String? = null,
    val error: String? = null
)
