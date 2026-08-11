/**
 * TigerWallet Android - Paymaster Service
 *
 * Fail-closed: gas sponsorship requires a REAL paymaster that signs the
 * userOpHash with a real secp256k1 ECDSA key (the on-chain Paymaster contract
 * verifies ecrecover(signature, hash) == paymaster owner). There is no
 * paymaster/sponsor endpoint on the backend (go/wallet_api) and no on-device
 * secp256k1 sponsor signer, so a real sponsor signature cannot be produced or
 * verified here. sponsorUserOp therefore POSTs to a configurable real sponsor
 * endpoint, or throws fail-closed rather than returning "0xPaymasterAddress"
 * + a fabricated hash. getBalance throws rather than returning a fabricated
 * "1000000000000000000". withdraw throws rather than returning a fabricated
 * tx hash.
 *
 * The duplicate `UserOperation` data class that collided with
 * AccountAbstractionService.kt is removed; this file reuses the canonical
 * `UserOperation` defined in AccountAbstractionService.kt.
 *
 * This service MUST be identical across ALL platforms (matches the iOS
 * PaymasterService.swift canonical implementation).
 */

package com.tigerwallet.app.master

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.math.BigInteger
import java.util.concurrent.TimeUnit

class PaymasterService private constructor() {

    companion object {
        val instance: PaymasterService by lazy { PaymasterService() }

        private const val ENTRY_POINT = "0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3"
        private val JSON_MEDIA_TYPE = "application/json".toMediaType()
    }

    private val whitelistedDApps = mutableMapOf<String, WhitelistEntry>()
    private var gasToken: String? = null

    /**
     * Optional backend paymaster/sponsor endpoint. If a real sponsor endpoint
     * is configured, sponsorUserOp POSTs the full userOp to it and uses the
     * returned real sponsor signature + paymaster address. Empty by default
     * → sponsorUserOp throws fail-closed.
     */
    @Volatile
    var sponsorEndpoint: String = ""

    /** Optional JWT used to authenticate the sponsor endpoint. */
    @Volatile
    var authToken: String? = null

    private val httpClient: OkHttpClient by lazy {
        OkHttpClient.Builder()
            .connectTimeout(15, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .build()
    }

    /**
     * Gas sponsorship requires a REAL paymaster signature over the userOpHash
     * (verified on-chain by ecrecover). With a sponsor endpoint configured,
     * this POSTs the full userOp fields to the sponsor (which computes the
     * real Keccak userOpHash server-side and signs it with real secp256k1)
     * and returns the real paymasterAndData. Without a sponsor endpoint, this
     * throws fail-closed. The previous implementation returned
     * "0xPaymasterAddress" + a sha256 hash as `paymasterAndData` — that is
     * removed. We never fabricate a hash or signature locally.
     */
    suspend fun sponsorUserOp(userOp: UserOperation): PaymasterData =
        withContext(Dispatchers.Default) {
            if (sponsorEndpoint.isEmpty()) {
                throw PaymasterError.NoSponsorConfigured
            }
            checkWhitelist(userOp.sender)

            val userOpJson = JSONObject().apply {
                put("sender", userOp.sender)
                put("nonce", userOp.nonce)
                put("init_code", userOp.initCode)
                put("call_data", userOp.callData)
                put("call_gas_limit", userOp.callGasLimit)
                put("verification_gas_limit", userOp.verificationGasLimit)
                put("pre_verification_gas", userOp.preVerificationGas)
                put("max_fee_per_gas", userOp.maxFeePerGas)
                put("max_priority_fee_per_gas", userOp.maxPriorityFeePerGas)
                put("paymaster_and_data", userOp.paymasterAndData)
                put("signature", userOp.signature)
            }
            val body = JSONObject().apply {
                put("user_op", userOpJson)
                put("entry_point", ENTRY_POINT)
            }

            val builder = Request.Builder()
                .url(sponsorEndpoint)
                .post(body.toString().toRequestBody(JSON_MEDIA_TYPE))
            authToken?.takeIf { it.isNotEmpty() }?.let {
                builder.header("Authorization", "Bearer $it")
            }

            val respData: String
            val statusCode: Int
            try {
                httpClient.newCall(builder.build()).execute().use { resp ->
                    statusCode = resp.code
                    respData = resp.body?.string() ?: ""
                }
            } catch (e: Exception) {
                throw PaymasterError.SponsorUnreachable(e.message ?: e.toString())
            }
            if (statusCode !in 200..299) {
                throw PaymasterError.SponsorRejected(statusCode, errorMessage(respData))
            }
            val json: JSONObject
            try {
                json = JSONObject(respData)
            } catch (e: Exception) {
                throw PaymasterError.SponsorUnreachable("malformed JSON")
            }
            // The sponsor returns the REAL paymaster address + its REAL
            // secp256k1 ECDSA signature over the Keccak userOpHash it computed
            // server-side. paymasterAndData is paymasterAddress || validUntil
            // || signature, per EIP-4337. The signature is verified on-chain by
            // the EntryPoint's ecrecover — the client never trusts it blindly.
            val paymasterAddress = json.optString("paymaster_address", "")
            val signature = json.optString("signature", "")
            val validUntil = json.optString("valid_until", "")
            if (!paymasterAddress.startsWith("0x") || paymasterAddress.length != 42 ||
                !signature.startsWith("0x") || validUntil.isEmpty()
            ) {
                throw PaymasterError.SponsorUnreachable("missing paymaster address or signature")
            }
            val paymasterAndData = paymasterAddress + validUntil + signature.removePrefix("0x")
            PaymasterData(
                paymasterAndData = paymasterAndData,
                preVerificationGas = userOp.preVerificationGas,
                verificationGasLimit = userOp.verificationGasLimit,
                callGasLimit = userOp.callGasLimit
            )
        }

    fun setPaymentToken(tokenAddress: String): Boolean {
        gasToken = tokenAddress
        return true
    }

    fun getPaymentToken(): String? = gasToken

    fun whitelistDApp(dAppAddress: String, limit: BigInteger, expiry: Long): Boolean {
        whitelistedDApps[dAppAddress] = WhitelistEntry(
            address = dAppAddress,
            sponsorLimit = limit,
            expiry = expiry,
            isActive = true
        )
        return true
    }

    fun removeWhitelist(dAppAddress: String): Boolean {
        return whitelistedDApps.remove(dAppAddress) != null
    }

    fun getWhitelistStatus(address: String): WhitelistStatus? {
        val entry = whitelistedDApps[address] ?: return null
        return WhitelistStatus(
            isWhitelisted = entry.isActive,
            limit = entry.sponsorLimit,
            expiry = entry.expiry,
            used = BigInteger.ZERO
        )
    }

    /**
     * Real paymaster balance requires an on-chain `balanceOf(paymaster)`
     * eth_call against the EntryPoint. With no paymaster configured, this
     * throws fail-closed rather than returning a fabricated
     * "1000000000000000000".
     */
    fun getBalance(): BigInteger {
        throw PaymasterError.NoSponsorConfigured
    }

    /**
     * Withdrawing paymaster deposit requires a real on-chain transaction
     * signed and broadcast by the paymaster owner. No fabricated tx hash is
     * ever returned. Fail-closed until a real withdrawal path is wired.
     */
    suspend fun withdraw(amount: BigInteger, recipient: String): String =
        withContext(Dispatchers.Default) {
            throw PaymasterError.NoSponsorConfigured
        }

    // ============================================================================
    // PRIVATE HELPERS
    // ============================================================================

    private fun checkWhitelist(userAddress: String) {
        val entry = whitelistedDApps[userAddress] ?: return
        if (entry.expiry < System.currentTimeMillis()) {
            throw IllegalStateException("Whitelist expired")
        }
    }

    private fun errorMessage(data: String): String? {
        return try {
            val json = JSONObject(data)
            json.optString("error", null)
        } catch (e: Exception) {
            data
        }
    }
}

sealed class PaymasterError(message: String) : Exception(message) {
    object NoSponsorConfigured :
        PaymasterError("No real paymaster/sponsor endpoint is configured; cannot sponsor a UserOperation or report a balance.")
    data class SponsorUnreachable(val detail: String) : PaymasterError("Sponsor endpoint unreachable: $detail")
    data class SponsorRejected(val code: Int, val detail: String?) :
        PaymasterError("Sponsor rejected the request (HTTP $code${detail?.let { ": $it" } ?: ""}).")
    object InvalidSignature : PaymasterError("Sponsor signature failed real secp256k1 ECDSA verification.")
}

data class PaymasterData(
    val paymasterAndData: String,
    val preVerificationGas: String,
    val verificationGasLimit: String,
    val callGasLimit: String
)

data class WhitelistEntry(
    val address: String,
    val sponsorLimit: BigInteger,
    val expiry: Long,
    val isActive: Boolean
)

data class WhitelistStatus(
    val isWhitelisted: Boolean,
    val limit: BigInteger,
    val expiry: Long,
    val used: BigInteger
)
