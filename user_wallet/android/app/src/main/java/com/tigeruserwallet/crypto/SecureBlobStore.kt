package com.tigeruserwallet.crypto

import android.content.Context
import android.content.SharedPreferences
import android.util.Base64
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

/**
 * SecureBlobStore — AES-256-GCM encrypted SharedPreferences.
 *
 * Stores the transparent no-registration session (random device-bound identity
 * + JWT) and the local wallet-id list. Mirrors the web OnboardingContext's
 * localStorage-backed session blob, but never persists secrets in plaintext:
 * every value is sealed with an Android-Keystore-backed AES-256-GCM key.
 *
 * Fail-closed: corrupt/missing key => treat as no stored value (the caller
 * re-provisions a fresh transparent session), never a guessed value.
 */
object SecureBlobStore {
    private const val PREFS = "userwallet_secure_prefs"
    private const val KEYSTORE = "AndroidKeyStore"
    private const val ALIAS = "userwallet_blob_master_key"
    private const val TRANSFORMATION = "AES/GCM/NoPadding"
    private const val GCM_TAG_BITS = 128

    @Volatile
    private var prefs: SharedPreferences? = null

    fun init(context: Context) {
        if (prefs != null) return
        prefs = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
    }

    private fun requirePrefs(): SharedPreferences =
        prefs ?: throw IllegalStateException("SecureBlobStore.init() not called")

    private fun getOrCreateKey(): SecretKey {
        val ks = KeyStore.getInstance(KEYSTORE).apply { load(null) }
        ks.getKey(ALIAS, null)?.let { return it as SecretKey }
        val gen = KeyGenerator.getInstance("AES", KEYSTORE)
        gen.init(256)
        return gen.generateKey().also { /* stored in keystore by the provider */ }
    }

    private fun encrypt(plain: String): String {
        val key = getOrCreateKey()
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.ENCRYPT_MODE, key)
        val iv = cipher.iv
        val ct = cipher.doFinal(plain.toByteArray(Charsets.UTF_8))
        val combined = iv + ct
        return Base64.encodeToString(combined, Base64.NO_WRAP)
    }

    private fun decrypt(blob: String): String? {
        if (blob.isEmpty()) return null
        return try {
            val key = getOrCreateKey()
            val combined = Base64.decode(blob, Base64.NO_WRAP)
            val iv = combined.copyOfRange(0, 12)
            val ct = combined.copyOfRange(12, combined.size)
            val cipher = Cipher.getInstance(TRANSFORMATION)
            cipher.init(Cipher.DECRYPT_MODE, key, GCMParameterSpec(GCM_TAG_BITS, iv))
            String(cipher.doFinal(ct), Charsets.UTF_8)
        } catch (e: Exception) {
            null
        }
    }

    fun putString(key: String, value: String?) {
        val ed = requirePrefs().edit()
        if (value == null) {
            ed.remove(key)
        } else {
            ed.putString(key, encrypt(value))
        }
        ed.apply()
    }

    fun getString(key: String): String? {
        val raw = requirePrefs().getString(key, null) ?: return null
        return decrypt(raw)
    }

    fun contains(key: String): Boolean = requirePrefs().contains(key)

    fun remove(key: String) {
        requirePrefs().edit().remove(key).apply()
    }
}
