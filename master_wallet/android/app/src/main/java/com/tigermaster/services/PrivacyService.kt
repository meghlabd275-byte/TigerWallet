/**
 * PrivacyService - Android Implementation
 *
 * Real, hardware-backed AES-GCM encryption of sensitive data (below) is retained.
 * The ZK-proof / stealth-address / CoinJoin / address-rotation features require a
 * proving system and secp256k1 ECDH curve math that are not available on this client
 * and have no canonical backend endpoint. They fail closed rather than fabricate
 * proofs, addresses, or signatures.
 */

package com.tigermaster.services

import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.math.BigInteger
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

class PrivacyService {
    private val keyStore: KeyStore = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }

    companion object {
        private const val ANDROID_KEYSTORE = "AndroidKeyStore"
        private const val PRIVACY_KEY_ALIAS = "tigermaster_privacy_key"
        private const val TRANSFORMATION = "AES/GCM/NoPadding"
        private const val GCM_TAG_LENGTH = 128
        private const val GCM_IV_LENGTH = 12

        const val PRIVACY_NONE = 0
        const val PRIVACY_STANDARD = 1
        const val PRIVACY_HIGH = 2
        const val PRIVACY_MAXIMUM = 3
    }

    /**
     * Stealth-address generation requires real ECDH on secp256k1 + keccak256, which is
     * not available on this client and has no canonical backend endpoint. Fail closed.
     */
    suspend fun generateStealthAddress(ownerAddress: String, spendingPublicKey: ByteArray): StealthAddress = withContext(Dispatchers.IO) {
        StealthAddress(success = false, error = "Stealth-address generation is not available on this client")
    }

    /**
     * CoinJoin mixing is not supported by the canonical backend and cannot be safely
     * constructed client-side. Fail closed.
     */
    suspend fun createCoinJoin(
        inputs: List<CoinJoinInput>,
        outputs: List<CoinJoinOutput>,
        privacyLevel: Int
    ): CoinJoinResult = withContext(Dispatchers.IO) {
        CoinJoinResult(success = false, error = "CoinJoin is not supported by the canonical backend")
    }

    /**
     * Real ZK-SNARK proof generation requires a proving system and circuit artifacts
     * not present on this client. Fail closed rather than return a fake proof.
     */
    suspend fun generateZKProof(amount: BigInteger, commitment: ByteArray): ZKProofResult = withContext(Dispatchers.IO) {
        ZKProofResult(success = false, error = "ZK proof generation is not available on this client")
    }

    /**
     * Verify a ZK proof. NEVER accept a proof on a non-empty check; without a real
     * verifier (and the matching verifying key / circuit) verification cannot be
     * performed, so this fails closed.
     */
    suspend fun verifyZKProof(proof: String, commitment: ByteArray): Boolean = withContext(Dispatchers.IO) {
        false
    }

    /**
     * Address rotation requires deriving a new secp256k1 keypair, unavailable here.
     * Fail closed.
     */
    suspend fun rotateAddress(currentAddress: String): RotationResult = withContext(Dispatchers.IO) {
        RotationResult(success = false, error = "Address rotation is not available on this client")
    }

    /**
     * Encrypt sensitive data with a hardware-backed AES-GCM key (real).
     */
    suspend fun encryptSensitiveData(data: ByteArray): EncryptedData = withContext(Dispatchers.IO) {
        try {
            val key = getOrCreatePrivacyKey()
            val cipher = Cipher.getInstance(TRANSFORMATION)
            cipher.init(Cipher.ENCRYPT_MODE, key)

            val iv = cipher.iv
            val encrypted = cipher.doFinal(data)

            val combined = ByteArray(iv.size + encrypted.size)
            System.arraycopy(iv, 0, combined, 0, iv.size)
            System.arraycopy(encrypted, 0, combined, iv.size, encrypted.size)

            EncryptedData(
                success = true,
                encrypted = Base64.encodeToString(combined, Base64.NO_WRAP)
            )
        } catch (e: Exception) {
            EncryptedData(success = false, error = e.message)
        }
    }

    /**
     * Decrypt sensitive data with the hardware-backed AES-GCM key (real).
     */
    suspend fun decryptSensitiveData(encryptedBase64: String): DecryptedData = withContext(Dispatchers.IO) {
        try {
            val key = getOrCreatePrivacyKey()
            val combined = Base64.decode(encryptedBase64, Base64.NO_WRAP)

            val iv = combined.copyOfRange(0, GCM_IV_LENGTH)
            val encrypted = combined.copyOfRange(GCM_IV_LENGTH, combined.size)

            val cipher = Cipher.getInstance(TRANSFORMATION)
            val spec = GCMParameterSpec(GCM_TAG_LENGTH, iv)
            cipher.init(Cipher.DECRYPT_MODE, key, spec)

            val decrypted = cipher.doFinal(encrypted)

            DecryptedData(
                success = true,
                data = decrypted
            )
        } catch (e: Exception) {
            DecryptedData(success = false, error = e.message)
        }
    }

    private fun getOrCreatePrivacyKey(): SecretKey {
        return if (keyStore.containsAlias(PRIVACY_KEY_ALIAS)) {
            keyStore.getKey(PRIVACY_KEY_ALIAS, null) as SecretKey
        } else {
            val keyGenerator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, ANDROID_KEYSTORE)
            val spec = KeyGenParameterSpec.Builder(
                PRIVACY_KEY_ALIAS,
                KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT
            )
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setKeySize(256)
                .build()

            keyGenerator.init(spec)
            keyGenerator.generateKey()
        }
    }
}

// Data classes

data class KeyPairResult(
    val privateKey: ByteArray,
    val publicKey: ByteArray
)

data class StealthAddress(
    val success: Boolean,
    val stealthAddress: String = "",
    val viewingKey: String = "",
    val ephemeralPublicKey: String = "",
    val error: String? = null
)

data class CoinJoinInput(
    val address: String,
    val amount: BigInteger,
    val privateKey: ByteArray
)

data class CoinJoinOutput(
    val address: String,
    val amount: BigInteger
)

data class CoinJoinResult(
    val success: Boolean,
    val mixedOutputs: List<String> = emptyList(),
    val proofs: List<ByteArray> = emptyList(),
    val rounds: Int = 0,
    val error: String? = null
)

data class ZKProofResult(
    val success: Boolean,
    val proof: String = "",
    val commitment: String = "",
    val blindingFactor: String = "",
    val error: String? = null
)

data class RotationResult(
    val success: Boolean,
    val newAddress: String = "",
    val newPublicKey: String = "",
    val viewingKey: String = "",
    val error: String? = null
)

data class EncryptedData(
    val success: Boolean,
    val encrypted: String = "",
    val error: String? = null
)

data class DecryptedData(
    val success: Boolean,
    val data: ByteArray = ByteArray(0),
    val error: String? = null
)
