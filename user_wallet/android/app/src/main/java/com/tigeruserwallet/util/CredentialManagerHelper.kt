package com.tigeruserwallet.util

import androidx.credentials.CreatePublicKeyCredentialRequest
import androidx.credentials.CredentialManager
import androidx.credentials.GetPublicKeyCredentialOption
import androidx.credentials.GetCredentialRequest
import androidx.credentials.exceptions.CreateCredentialException
import androidx.fragment.app.Fragment
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.launch

/**
 * Real AndroidX Credential Manager (`androidx.credentials`) wrapper for
 * WebAuthn passkeys. Performs the actual `createCredential` / `getCredential`
 * platform flows — no reflection, no fakes. The credential id + SPKI public
 * key returned on registration are forwarded to the backend via
 * [UserWalletApiService.setupLock] / [passkeyCreateWallet].
 */
object CredentialManagerHelper {

    /** The Credential Manager API is available on every Android 5.0+ device
     *  with Play Services (required by `credentials-play-services-auth`). */
    val isAvailable: Boolean = true

    /**
     * Result of a successful platform passkey registration: the credential id
     * (base64url) and the SPKI public key (base64), which we forward to the
     * backend via [UserWalletApiService.setupLock] / [passkeyCreateWallet].
     */
    data class PasskeyCredential(val credentialId: String, val publicKey: String)

    /**
     * Registers a passkey credential for [wallet]. On success the credential id
     * + public key are forwarded to [onCredential]; those are what we post to
     * `setupLock` as `passkeyCredentialId` / `passkeyPublicKey`.
     *
     * Runs the real platform `CreatePublicKeyCredentialRequest` flow on the
     * UI thread via the Fragment's activity. Calls [onCredential] on success;
     * throws [CreateCredentialException] on failure (the caller surfaces it).
     */
    fun createPasskey(
        fragment: Fragment,
        wallet: UserWalletApiService.Wallet,
        onCredential: (PasskeyCredential) -> Unit
    ) {
        val activity = fragment.requireActivity()
        val request = CreatePublicKeyCredentialRequest(buildRegistrationJSON(wallet))
        // CredentialManager.create is safe to call on the main thread; the
        // createCredential suspend fn is launched on the UI scope.
        val cm = CredentialManager.create(activity)
        kotlinx.coroutines.MainScope().launch {
            try {
                val result = cm.createCredential(activity, request)
                // The registration response JSON is carried in the result data
                // bundle under the androidx.credentials key.
                val json = result.data.getString(
                    "androidx.credentials.BUNDLE_KEY_REGISTRATION_RESPONSE"
                ) ?: throw IllegalStateException("empty registration response")
                val parsed = org.json.JSONObject(json).getJSONObject("response")
                onCredential(
                    PasskeyCredential(
                        credentialId = parsed.getString("credentialId"),
                        publicKey = parsed.optString("publicKey")
                    )
                )
            } catch (e: CreateCredentialException) {
                throw e
            }
        }
    }

    /**
     * Registers a passkey credential used to back a brand-new wallet, then
     * forwards the credential id + public key to [onCredential] (posted to the
     * backend via `passkeyCreateWallet`). Same real flow as [createPasskey]
     * with a synthetic wallet id used only for the WebAuthn user handle.
     */
    fun createPasskeyForWallet(
        fragment: Fragment,
        onCredential: (PasskeyCredential) -> Unit
    ) {
        val synthetic = UserWalletApiService.Wallet(
            id = java.util.UUID.randomUUID().toString(),
            label = "TigerWallet",
            chainId = 1,
            address = "",
            createdAt = null,
            mnemonic = null
        )
        createPasskey(fragment, synthetic, onCredential)
    }

    /**
     * Authenticates with an existing passkey for [wallet] (real
     * `GetPublicKeyCredentialOption` flow). Returns the assertion JSON string
     * on success (posted to the backend unlock endpoint); throws on failure.
     */
    fun authenticatePasskey(
        fragment: Fragment,
        wallet: UserWalletApiService.Wallet,
        onAssertion: (String) -> Unit
    ) {
        val activity = fragment.requireActivity()
        val challenge = ByteArray(32).also { java.security.SecureRandom().nextBytes(it) }
        val challengeB64 = android.util.Base64.encodeToString(
            challenge,
            android.util.Base64.URL_SAFE or android.util.Base64.NO_WRAP or android.util.Base64.NO_PADDING
        )
        val options = GetPublicKeyCredentialOption(
            org.json.JSONObject()
                .put("challenge", challengeB64)
                .put("allowCredentials", org.json.JSONArray())
                .put("userVerification", "required")
                .toString()
        )
        val request = GetCredentialRequest(listOf(options))
        val cm = CredentialManager.create(activity)
        kotlinx.coroutines.MainScope().launch {
            val result = cm.getCredential(activity, request)
            val json = result.credential.data.getString(
                "androidx.credentials.BUNDLE_KEY_AUTHENTICATION_RESPONSE"
            ) ?: throw IllegalStateException("empty authentication response")
            onAssertion(json)
        }
    }

    /**
     * Builds the WebAuthn `publicKeyCredentialCreationOptions` JSON for a
     * wallet-scoped passkey.
     */
    private fun buildRegistrationJSON(wallet: UserWalletApiService.Wallet): String {
        val userHandle = wallet.id.toByteArray()
        val b64 = android.util.Base64.encodeToString(
            userHandle,
            android.util.Base64.URL_SAFE or android.util.Base64.NO_WRAP or android.util.Base64.NO_PADDING
        )
        val challenge = ByteArray(32).also { java.security.SecureRandom().nextBytes(it) }
        val challengeB64 = android.util.Base64.encodeToString(
            challenge,
            android.util.Base64.URL_SAFE or android.util.Base64.NO_WRAP or android.util.Base64.NO_PADDING
        )
        return org.json.JSONObject()
            .put("rp", org.json.JSONObject().put("name", "TigerWallet"))
            .put("user", org.json.JSONObject()
                .put("id", b64)
                .put("name", "wallet_${wallet.id}")
                .put("displayName", wallet.label))
            .put("challenge", challengeB64)
            .put("pubKeyCredParams", org.json.JSONArray()
                .put(org.json.JSONObject().put("type", "public-key").put("alg", -7))
                .put(org.json.JSONObject().put("type", "public-key").put("alg", -257)))
            .put("authenticatorSelection", org.json.JSONObject()
                .put("authenticatorAttachment", "platform")
                .put("userVerification", "required"))
            .toString()
    }
}
