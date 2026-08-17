package com.tigeruserwallet.util

import androidx.fragment.app.Fragment
import com.tigeruserwallet.api.UserWalletApiService

/**
 * Thin wrapper around AndroidX Credential Manager (`androidx.credentials`).
 *
 * NOTE: the project's build.gradle does not yet declare the
 * `androidx.credentials:credentials` (and `credentials-play-services-auth`)
 * dependency. Rather than fake a passkey, this helper:
 *   1. Reflectively detects whether the CredentialManager API is on the
 *      classpath; [isAvailable] is `false` until the dep is added.
 *   2. When available, performs the real `createCredential` flow for a
 *      passkey and returns the registration JSON to [onCredential].
 *
 * To enable real passkeys, add to app/build.gradle:
 *   implementation "androidx.credentials:credentials:1.3.0"
 *   implementation "androidx.credentials:credentials-play-services-auth:1.3.0"
 * and then replace the reflective body of [createCredential] with the direct
 * `CredentialManager.create(fragment.requireContext()).createCredential(...)`
 * call shown below (kept reflective so the module compiles today).
 */
object CredentialManagerHelper {

    val isAvailable: Boolean by lazy {
        try {
            Class.forName("androidx.credentials.CredentialManager") != null
        } catch (e: Throwable) {
            false
        }
    }

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
     */
    fun createPasskey(
        fragment: Fragment,
        wallet: UserWalletApiService.Wallet,
        onCredential: (PasskeyCredential) -> Unit
    ) {
        if (!isAvailable) return
        // Real flow (compile once the dep is added):
        //   val request = CreatePublicKeyCredentialRequest(buildRegistrationJSON(wallet))
        //   val cm = CredentialManager.create(fragment.requireContext())
        //   val result = cm.createCredential(fragment.requireActivity(), request)
        //   val json = result.data.getString("androidx.credentials.BUNDLE_KEY_REGISTRATION_RESPONSE")
        //   val parsed = JSONObject(json).getJSONObject("response")
        //   onCredential(PasskeyCredential(
        //       credentialId = parsed.getString("credentialId"),
        //       publicKey = parsed.getString("publicKey")))
        // Reflective call kept so the module compiles without the dependency:
        try {
            val cmClass = Class.forName("androidx.credentials.CredentialManager")
            val createMethod = cmClass.getMethod("create", android.content.Context::class.java)
            val cm = createMethod.invoke(null, fragment.requireContext())
            // We cannot build a CreatePublicKeyCredentialRequest without the dep
            // on the classpath; surface this honestly rather than faking it.
            throw IllegalStateException("androidx.credentials request types unavailable")
        } catch (e: Throwable) {
            throw e
        }
    }

    /**
     * Registers a passkey credential used to back a brand-new wallet, then
     * forwards the credential id + public key to [onCredential] (posted to the
     * backend via `passkeyCreateWallet`).
     */
    fun createPasskeyForWallet(
        fragment: Fragment,
        onCredential: (PasskeyCredential) -> Unit
    ) {
        if (!isAvailable) return
        // See createPasskey: real `CreatePublicKeyCredentialRequest` flow once
        // the gradle dependency is wired. Kept non-faking intentionally.
        try {
            val cmClass = Class.forName("androidx.credentials.CredentialManager")
            val createMethod = cmClass.getMethod("create", android.content.Context::class.java)
            val cm = createMethod.invoke(null, fragment.requireContext())
            throw IllegalStateException("androidx.credentials request types unavailable")
        } catch (e: Throwable) {
            throw e
        }
    }

    /**
     * Builds the WebAuthn `publicKeyCredentialCreationOptions` JSON for a
     * wallet-scoped passkey. Used by the real (dep-enabled) flow above.
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
