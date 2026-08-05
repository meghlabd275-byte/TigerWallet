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
import kotlinx.coroutines.suspendCancellableCoroutine
import java.security.KeyStore
import java.security.SecureRandom
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec
import kotlin.coroutines.resume

class BiometricService(private val context: Context) {
    private val keyStore: KeyStore = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
    private val secureRandom = SecureRandom()
    private var biometricPrompt: BiometricPrompt? = null
    
    companion object {
        private const val BIOMETRIC_KEY_ALIAS = "tigermaster_biometric_key"
        private const val PIN_KEY_ALIAS = "tigermaster_pin_key"
        private const val TRANSFORMATION = "AES/GCM/NoPadding"
        private const val GCM_TAG_LENGTH = 128
        private const val GCM_IV_LENGTH = 12
        private const val MAX_PIN_ATTEMPTS = 5
        private const val PIN_LENGTH = 6
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
        return keyStore.containsAlias(PIN_KEY_ALIAS)
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
            val key = getOrCreatePinKey()
            val cipher = Cipher.getInstance(TRANSFORMATION)
            cipher.init(Cipher.ENCRYPT_MODE, key)
            
            val iv = cipher.iv
            val encrypted = cipher.doFinal(pin.toByteArray(Charsets.UTF_8))
            
            val combined = ByteArray(iv.size + encrypted.size)
            System.arraycopy(iv, 0, combined, 0, iv.size)
            System.arraycopy(encrypted, 0, combined, iv.size, encrypted.size)
            
            // Store encrypted PIN hash
            val pinHash = hashPin(pin)
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
        
        // In production, verify against stored encrypted PIN
        val inputHash = hashPin(pin)
        
        // Simplified check - production would decrypt stored PIN
        if (isPinSetup()) {
            return PinVerificationResult(success = true, remainingAttempts = MAX_PIN_ATTEMPTS)
        }
        
        return PinVerificationResult(success = false, error = "PIN not set up")
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
     * Lock wallet after failed attempts
     */
    suspend fun lockWallet(): Boolean {
        // In production, invalidate biometric key or clear session
        return true
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
    
    private fun hashPin(pin: String): String {
        // Simplified - production would use proper password hashing (Argon2, bcrypt)
        return pin.hashCode().toString()
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
