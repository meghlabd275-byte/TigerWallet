/**
 * BiometricService - Android Implementation
 * Biometric and PIN authentication for Master Wallet
 */

package com.tigermaster.services

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import androidx.biometric.BiometricManager
import androidx.biometric.BiometricPrompt
import androidx.core.content.ContextCompat
import androidx.fragment.app.FragmentActivity
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import kotlinx.coroutines.suspendCancellableCoroutine
import java.security.KeyStore
import java.security.SecureRandom
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.SecretKeyFactory
import javax.crypto.spec.GCMParameterSpec
import javax.crypto.spec.PBEKeySpec
import kotlin.coroutines.resume

class BiometricService(private val context: Context) {
    private val keyStore: KeyStore = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
    private val secureRandom = SecureRandom()
    private var biometricPrompt: BiometricPrompt? = null

    private val masterKey: MasterKey by lazy {
        MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
    }

    private val pinPrefs by lazy {
        EncryptedSharedPreferences.create(
            context,
            "pin_auth_prefs",
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
        )
    }

    companion object {
        private const val BIOMETRIC_KEY_ALIAS = "tigermaster_biometric_key"
        private const val PIN_KEY_ALIAS = "tigermaster_pin_key"
        private const val TRANSFORMATION = "AES/GCM/NoPadding"
        private const val GCM_TAG_LENGTH = 128
        private const val GCM_IV_LENGTH = 12
        private const val MAX_PIN_ATTEMPTS = 5
        private const val PIN_LENGTH = 6
        private const val PIN_SALT_KEY = "pin_pbkdf2_salt"
        private const val PIN_HASH_KEY = "pin_pbkdf2_hash"
        private const val PIN_ATTEMPTS_KEY = "pin_failed_attempts"
        private const val PBKDF2_ITERATIONS = 200_000
        private const val PBKDF2_KEY_LENGTH = 256
        private const val SALT_LENGTH = 16
    }
    
    /**
     * Check if biometric authentication is available
     */
    fun isBiometricAvailable(): BiometricStatus {
        val biometricManager = BiometricManager.from(context)
        return when (biometricManager.canAuthenticate(BiometricManager.Authenticators.BIOMETRIC_STRONG)) {
            BiometricManager.BIOMETRIC_SUCCESS -> BiometricStatus.AVAILABLE
            BiometricManager.BIOMETRIC_ERROR_NO_HARDWARE -> BiometricStatus.NO_HARDWARE
            BiometricManager.BIOMETRIC_ERROR_HW_UNAVAILABLE -> BiometricStatus.HARDWARE_UNAVAILABLE
            BiometricManager.BIOMETRIC_ERROR_NONE_ENROLLED -> BiometricStatus.NOT_ENROLLED
            else -> BiometricStatus.UNAVAILABLE
        }
    }
    
    /**
     * Check if PIN is set up
     */
    fun isPinSetup(): Boolean {
        return pinPrefs.contains(PIN_SALT_KEY) && pinPrefs.contains(PIN_HASH_KEY)
    }
    
    /**
     * Set up biometric authentication
     */
    suspend fun setupBiometric(title: String, subtitle: String): Boolean = suspendCancellableCoroutine { continuation ->
        try {
            val keyGenerator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore")
            val spec = KeyGenParameterSpec.Builder(
                BIOMETRIC_KEY_ALIAS,
                KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT
            )
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setKeySize(256)
                .setUserAuthenticationRequired(true)
                .setInvalidatedByBiometricEnrollment(true)
                .build()
            
            keyGenerator.init(spec)
            keyGenerator.generateKey()
            
            continuation.resume(true)
        } catch (e: Exception) {
            continuation.resume(false)
        }
    }
    
    /**
     * Authenticate with biometric
     */
    suspend fun authenticateWithBiometric(
        activity: FragmentActivity,
        title: String = "Authenticate",
        subtitle: String = "Use biometric to unlock",
        negativeButtonText: String = "Use PIN"
    ): BiometricResult = suspendCancellableCoroutine { continuation ->
        val executor = ContextCompat.getMainExecutor(context)
        
        val callback = object : BiometricPrompt.AuthenticationCallback() {
            override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                continuation.resume(BiometricResult(success = true))
            }
            
            override fun onAuthenticationError(errorCode: Int, errString: CharSequence) {
                if (errorCode == BiometricPrompt.ERROR_NEGATIVE_BUTTON) {
                    continuation.resume(BiometricResult(success = false, error = "User chose PIN"))
                } else {
                    continuation.resume(BiometricResult(success = false, error = errString.toString()))
                }
            }
            
            override fun onAuthenticationFailed() {
                // Don't resume here - let user retry
            }
        }
        
        biometricPrompt = BiometricPrompt(activity, executor, callback)
        
        val promptInfo = BiometricPrompt.PromptInfo.Builder()
            .setTitle(title)
            .setSubtitle(subtitle)
            .setNegativeButtonText(negativeButtonText)
            .setAllowedAuthenticators(BiometricManager.Authenticators.BIOMETRIC_STRONG)
            .build()
        
        biometricPrompt?.authenticate(promptInfo)
    }
    
    /**
     * Set up PIN
     */
    suspend fun setupPin(pin: String): Boolean {
        if (pin.length != PIN_LENGTH || !pin.all { it.isDigit() }) {
            return false
        }

        return try {
            val salt = ByteArray(SALT_LENGTH).also { secureRandom.nextBytes(it) }
            val hash = pbkdf2(pin, salt)

            pinPrefs.edit()
                .putString(PIN_SALT_KEY, Base64.encodeToString(salt, Base64.NO_WRAP))
                .putString(PIN_HASH_KEY, Base64.encodeToString(hash, Base64.NO_WRAP))
                .putInt(PIN_ATTEMPTS_KEY, 0)
                .apply()
            true
        } catch (e: Exception) {
            false
        }
    }
    
    /**
     * Verify PIN
     */
    suspend fun verifyPin(pin: String): PinVerificationResult {
        if (pin.length != PIN_LENGTH || !pin.all { it.isDigit() }) {
            return PinVerificationResult(success = false, error = "Invalid PIN format")
        }

        if (!isPinSetup()) {
            return PinVerificationResult(success = false, error = "PIN not set up")
        }

        val attempts = pinPrefs.getInt(PIN_ATTEMPTS_KEY, 0)
        if (attempts >= MAX_PIN_ATTEMPTS) {
            return PinVerificationResult(
                success = false,
                remainingAttempts = 0,
                error = "PIN locked due to too many failed attempts"
            )
        }

        val saltBase64 = pinPrefs.getString(PIN_SALT_KEY, null)
        val storedHashBase64 = pinPrefs.getString(PIN_HASH_KEY, null)
        if (saltBase64 == null || storedHashBase64 == null) {
            return PinVerificationResult(success = false, error = "PIN not set up")
        }

        return try {
            val salt = Base64.decode(saltBase64, Base64.NO_WRAP)
            val storedHash = Base64.decode(storedHashBase64, Base64.NO_WRAP)
            val inputHash = pbkdf2(pin, salt)

            // Constant-time comparison to avoid timing side channels.
            val matched = constantTimeEquals(inputHash, storedHash)

            if (matched) {
                pinPrefs.edit().putInt(PIN_ATTEMPTS_KEY, 0).apply()
                PinVerificationResult(success = true, remainingAttempts = MAX_PIN_ATTEMPTS)
            } else {
                val newAttempts = attempts + 1
                pinPrefs.edit().putInt(PIN_ATTEMPTS_KEY, newAttempts).apply()
                PinVerificationResult(
                    success = false,
                    remainingAttempts = (MAX_PIN_ATTEMPTS - newAttempts).coerceAtLeast(0),
                    error = "Incorrect PIN"
                )
            }
        } catch (e: Exception) {
            PinVerificationResult(success = false, error = "PIN verification error")
        }
    }
    
    /**
     * Change PIN
     */
    suspend fun changePin(oldPin: String, newPin: String): Boolean {
        val verifyResult = verifyPin(oldPin)
        if (!verifyResult.success) {
            return false
        }
        
        return setupPin(newPin)
    }
    
    /**
     * Lock the wallet: delete the Android-Keystore biometric key alias so the next
     * unlock requires fresh biometric authentication, and reset the failed-attempt
     * counter. Returns true if the key alias was removed (or already absent).
     */
    suspend fun lockWallet(): Boolean = withContext(Dispatchers.IO) {
        try {
            val keyStore = java.security.KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
            if (keyStore.containsAlias(BIOMETRIC_KEY_ALIAS)) {
                keyStore.deleteEntry(BIOMETRIC_KEY_ALIAS)
            }
            pinPrefs.edit().putInt(PIN_ATTEMPTS_KEY, 0).apply()
            true
        } catch (e: Exception) {
            false
        }
    }
    
    /**
     * Generate secure random PIN
     */
    fun generateRandomPin(): String {
        val pin = StringBuilder()
        repeat(PIN_LENGTH) {
            pin.append(secureRandom.nextInt(10))
        }
        return pin.toString()
    }
    
    /**
     * Encrypt sensitive wallet data
     */
    suspend fun encryptWalletData(data: ByteArray): EncryptedWalletData {
        return try {
            val key = getOrCreateBiometricKey()
            val cipher = Cipher.getInstance(TRANSFORMATION)
            cipher.init(Cipher.ENCRYPT_MODE, key)
            
            val iv = cipher.iv
            val encrypted = cipher.doFinal(data)
            
            val combined = ByteArray(iv.size + encrypted.size)
            System.arraycopy(iv, 0, combined, 0, iv.size)
            System.arraycopy(encrypted, 0, combined, iv.size, encrypted.size)
            
            EncryptedWalletData(
                success = true,
                encryptedData = Base64.encodeToString(combined, Base64.NO_WRAP)
            )
        } catch (e: Exception) {
            EncryptedWalletData(success = false, error = e.message)
        }
    }
    
    /**
     * Decrypt sensitive wallet data
     */
    suspend fun decryptWalletData(encryptedBase64: String): DecryptedWalletData {
        return try {
            val key = getOrCreateBiometricKey()
            val combined = Base64.decode(encryptedBase64, Base64.NO_WRAP)
            
            val iv = combined.copyOfRange(0, GCM_IV_LENGTH)
            val encrypted = combined.copyOfRange(GCM_IV_LENGTH, combined.size)
            
            val cipher = Cipher.getInstance(TRANSFORMATION)
            val spec = GCMParameterSpec(GCM_TAG_LENGTH, iv)
            cipher.init(Cipher.DECRYPT_MODE, key, spec)
            
            val decrypted = cipher.doFinal(encrypted)
            
            DecryptedWalletData(success = true, data = decrypted)
        } catch (e: Exception) {
            DecryptedWalletData(success = false, error = e.message)
        }
    }
    
    private fun getOrCreateBiometricKey(): SecretKey {
        return if (keyStore.containsAlias(BIOMETRIC_KEY_ALIAS)) {
            keyStore.getKey(BIOMETRIC_KEY_ALIAS, null) as SecretKey
        } else {
            val keyGenerator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore")
            val spec = KeyGenParameterSpec.Builder(
                BIOMETRIC_KEY_ALIAS,
                KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT
            )
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setKeySize(256)
                .setUserAuthenticationRequired(true)
                .build()
            
            keyGenerator.init(spec)
            keyGenerator.generateKey()
        }
    }
    
    private fun getOrCreatePinKey(): SecretKey {
        return if (keyStore.containsAlias(PIN_KEY_ALIAS)) {
            keyStore.getKey(PIN_KEY_ALIAS, null) as SecretKey
        } else {
            val keyGenerator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore")
            val spec = KeyGenParameterSpec.Builder(
                PIN_KEY_ALIAS,
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
    
    private fun pbkdf2(pin: String, salt: ByteArray): ByteArray {
        val spec = PBEKeySpec(
            pin.toCharArray(),
            salt,
            PBKDF2_ITERATIONS,
            PBKDF2_KEY_LENGTH
        )
        return try {
            val factory = SecretKeyFactory.getInstance("PBKDF2WithHmacSHA256")
            factory.generateSecret(spec).encoded
        } finally {
            spec.clearPassword()
        }
    }

    private fun constantTimeEquals(a: ByteArray, b: ByteArray): Boolean {
        if (a.size != b.size) return false
        var result = 0
        for (i in a.indices) {
            result = result or (a[i].toInt() xor b[i].toInt())
        }
        return result == 0
    }
}

// Data classes

enum class BiometricStatus {
    AVAILABLE,
    NO_HARDWARE,
    HARDWARE_UNAVAILABLE,
    NOT_ENROLLED,
    UNAVAILABLE
}

data class BiometricResult(
    val success: Boolean,
    val error: String? = null
)

data class PinVerificationResult(
    val success: Boolean,
    val remainingAttempts: Int = 0,
    val error: String? = null
)

data class EncryptedWalletData(
    val success: Boolean,
    val encryptedData: String = "",
    val error: String? = null
)

data class DecryptedWalletData(
    val success: Boolean,
    val data: ByteArray = ByteArray(0),
    val error: String? = null
)
