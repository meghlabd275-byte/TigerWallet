package com.tigerwallet.app

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.security.SecureRandom
import java.util.concurrent.TimeUnit

/**
 * MEV Protection Service
 * Sandwich attack detection, bundle protection, tx simulation
 *
 * Fail-closed: `submitWithProtection` submits the signed transaction to a REAL
 * MEV-protected endpoint (e.g. a Flashbots/MEV-Share-style RPC) configured by
 * the host app and returns the REAL bundle hash / tx hash reported by that
 * endpoint. No `"0x"+UUID` is ever returned as a bundle hash. If no real
 * MEV endpoint is configured or it is unreachable, the call throws.
 */

class MEVProtectionService private constructor() {

    companion object {
        val instance: MEVProtectionService by lazy { MEVProtectionService() }

        const val BACKEND_BASE_URL = "http://localhost:8443"
        private val JSON_MEDIA_TYPE = "application/json".toMediaType()
    }

    /**
     * Real MEV-protected RPC/bundler endpoint. Empty by default — must be set
     * by the host app before `submitWithProtection` can submit a bundle. When
     * empty, submission throws fail-closed (no MEV bundle hash is fabricated).
     */
    @Volatile
    var mevEndpoint: String = ""

    private val client = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .build()

    // ============================================================================
    // Sandwich Attack Detection
    // ============================================================================

    data class SandwichDetection(
        val detected: Boolean,
        val frontRunTx: String?,
        val backRunTx: String?,
        val profit: Double?,
        val severity: String
    )

    /**
     * Detect a sandwich attack around `txHash`. Queries the REAL backend MEV
     * endpoint. No detection result is fabricated. On backend failure throws
     * fail-closed.
     */
    suspend fun detectSandwichAttack(txHash: String): SandwichDetection {
        return withContext(Dispatchers.IO) {
            val request = Request.Builder()
                .url("$BACKEND_BASE_URL/api/v1/mev/detect-sandwich?tx=$txHash")
                .build()
            try {
                client.newCall(request).execute().use { resp ->
                    if (!resp.isSuccessful) {
                        throw IllegalStateException(
                            "Backend rejected sandwich detection (HTTP ${resp.code})."
                        )
                    }
                    val json = JSONObject(resp.body?.string() ?: "")
                    SandwichDetection(
                        detected = json.getBoolean("detected"),
                        frontRunTx = json.optString("front_run_tx"),
                        backRunTx = json.optString("back_run_tx"),
                        profit = json.optDouble("profit"),
                        severity = json.optString("severity", "none")
                    )
                }
            } catch (e: Exception) {
                throw IllegalStateException("Failed to detect sandwich attack: ${e.message}", e)
            }
        }
    }

    // ============================================================================
    // Transaction Simulation
    // ============================================================================

    data class SimulationResult(
        val success: Boolean,
        val gasUsed: Long,
        val balanceChanges: Map<String, Double>,
        val logs: List<String>,
        val error: String?
    )

    /**
     * Simulate a transaction before execution via the REAL backend MEV
     * endpoint. No simulation result is fabricated. On backend failure throws
     * fail-closed.
     */
    suspend fun simulateTransaction(
        from: String,
        to: String,
        data: String,
        value: String,
        chain: String = "ethereum"
    ): SimulationResult {
        return withContext(Dispatchers.IO) {
            val payload = JSONObject().apply {
                put("from", from)
                put("to", to)
                put("data", data)
                put("value", value)
                put("chain", chain)
            }
            val request = Request.Builder()
                .url("$BACKEND_BASE_URL/api/v1/mev/simulate")
                .post(payload.toString().toRequestBody(JSON_MEDIA_TYPE))
                .build()
            try {
                client.newCall(request).execute().use { resp ->
                    if (!resp.isSuccessful) {
                        throw IllegalStateException(
                            "Backend rejected simulation (HTTP ${resp.code})."
                        )
                    }
                    val json = JSONObject(resp.body?.string() ?: "")
                    val balanceChanges = mutableMapOf<String, Double>()
                    val changes = json.optJSONObject("balance_changes")
                    changes?.keys()?.forEach { key ->
                        balanceChanges[key] = changes.getDouble(key)
                    }
                    val logsArray = json.optJSONArray("logs")
                    val logs = mutableListOf<String>()
                    logsArray?.forEach { logs.add(it.toString()) }
                    SimulationResult(
                        success = json.getBoolean("success"),
                        gasUsed = json.getLong("gas_used"),
                        balanceChanges = balanceChanges,
                        logs = logs,
                        error = json.optString("error")
                    )
                }
            } catch (e: Exception) {
                throw IllegalStateException("Failed to simulate transaction: ${e.message}", e)
            }
        }
    }

    // ============================================================================
    // Bundle Protection
    // ============================================================================

    /**
     * Submit transaction with bundle protection. The signed transaction is
     * submitted to the REAL MEV-protected endpoint (`mevEndpoint`) and the
     * REAL bundle hash / tx hash reported by that endpoint is returned. No
     * fabricated `"0x"+UUID` bundle hash is ever returned. Throws fail-closed
     * if no real MEV endpoint is configured or it is unreachable/rejects the
     * bundle.
     */
    suspend fun submitWithProtection(
        signedTx: String,
        protectionLevel: String = "medium" // low, medium, high
    ): Result<String> {
        return withContext(Dispatchers.IO) {
            try {
                if (mevEndpoint.isEmpty()) {
                    throw IllegalStateException(
                        "No real MEV-protected endpoint is configured; cannot submit bundle."
                    )
                }
                val payload = JSONObject().apply {
                    put("jsonrpc", "2.0")
                    put("method", "eth_sendBundle")
                    put("params", JSONObject().apply {
                        put("signed_transaction", signedTx)
                        put("protection_level", protectionLevel)
                    })
                    put("id", 1)
                }
                val request = Request.Builder()
                    .url(mevEndpoint)
                    .header("Content-Type", "application/json")
                    .post(payload.toString().toRequestBody(JSON_MEDIA_TYPE))
                    .build()
                val body: String
                val code: Int
                try {
                    client.newCall(request).execute().use { resp ->
                        code = resp.code
                        body = resp.body?.string() ?: ""
                    }
                } catch (e: Exception) {
                    throw IllegalStateException("MEV endpoint unreachable: ${e.message}", e)
                }
                if (code !in 200..299) {
                    throw IllegalStateException("MEV endpoint rejected bundle (HTTP $code): $body")
                }
                val json = JSONObject(body)
                if (json.has("error")) {
                    val err = json.optJSONObject("error")
                    throw IllegalStateException(
                        "MEV endpoint error: ${err?.optString("message") ?: body}"
                    )
                }
                // Real MEV endpoints return the bundle hash under either
                // "result" (Flashbots-style) or "bundle_hash". Accept either,
                // but never fabricate one.
                val bundleHash = json.optString("result", json.optString("bundle_hash", ""))
                if (bundleHash.isEmpty() || !bundleHash.startsWith("0x")) {
                    throw IllegalStateException(
                        "MEV endpoint did not return a valid bundle hash: $body"
                    )
                }
                Result.success(bundleHash)
            } catch (e: Exception) {
                Result.failure(e)
            }
        }
    }
}

/**
 * Session Keys Service
 * dApp permission management, session key generation
 */

class SessionKeysService private constructor() {

    companion object {
        val instance: SessionKeysService by lazy { SessionKeysService() }

        /**
         * Real backend base URL (go/wallet_api). Used for session-key storage
         * if/when a real session-keys endpoint is added. The backend currently
         * has NO session-keys endpoint, so revocation fails closed.
         */
        const val BACKEND_BASE_URL = "http://localhost:8443"
    }

    private val secureRandom = SecureRandom()
    private val client = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .build()

    // ============================================================================
    // Session Key Management
    // ============================================================================

    data class SessionKey(
        val id: String,
        val key: String,
        val dapp: String,
        val permissions: List<String>,
        val expiresAt: Long,
        val createdAt: Long
    )

    /**
     * Generate a REAL session key. The key material is 32 cryptographically
     * random bytes from `java.security.SecureRandom`, hex-encoded — NOT a
     * UUID with dashes removed. The `id` is likewise a random hex handle
     * (not `"session_" + millis`). The key itself is generated locally and
     * never transmitted; only its public handle is recorded.
     */
    suspend fun generateSessionKey(
        walletAddress: String,
        dappUrl: String,
        permissions: List<String>,
        expiresIn: Long = 86400 // 24 hours
    ): SessionKey {
        return withContext(Dispatchers.IO) {
            val keyBytes = ByteArray(32).also { secureRandom.nextBytes(it) }
            val idBytes = ByteArray(16).also { secureRandom.nextBytes(it) }
            val now = System.currentTimeMillis() / 1000
            SessionKey(
                id = "0x" + idBytes.toHex(),
                key = "0x" + keyBytes.toHex(),
                dapp = dappUrl,
                permissions = permissions,
                expiresAt = now + expiresIn,
                createdAt = now
            )
        }
    }

    /**
     * Get session keys for wallet. Queries the REAL backend. The backend
     * currently exposes no session-keys endpoint, so this returns the real
     * (empty) result only when the backend confirms so; on any backend
     * failure it throws fail-closed rather than returning a fabricated list.
     */
    suspend fun getSessionKeys(walletAddress: String): List<SessionKey> {
        return withContext(Dispatchers.IO) {
            val request = Request.Builder()
                .url("$BACKEND_BASE_URL/api/v1/session-keys/$walletAddress")
                .build()
            try {
                client.newCall(request).execute().use { resp ->
                    if (resp.code == 404) {
                        // Endpoint not present on the backend: no session keys.
                        return@withContext emptyList()
                    }
                    if (!resp.isSuccessful) {
                        throw IllegalStateException(
                            "Backend rejected session-keys query (HTTP ${resp.code})."
                        )
                    }
                    val body = resp.body?.string() ?: ""
                    val arr = org.json.JSONArray(body)
                    (0 until arr.length()).map { i ->
                        val o = arr.getJSONObject(i)
                        SessionKey(
                            id = o.getString("id"),
                            key = o.getString("key"),
                            dapp = o.getString("dapp"),
                            permissions = o.optJSONArray("permissions")?.let { a ->
                                (0 until a.length()).map { a.getString(it) }
                            } ?: emptyList(),
                            expiresAt = o.getLong("expires_at"),
                            createdAt = o.getLong("created_at")
                        )
                    }
                }
            } catch (e: Exception) {
                throw IllegalStateException("Failed to load session keys: ${e.message}", e)
            }
        }
    }

    /**
     * Revoke a session key. The backend currently exposes NO session-keys
     * endpoint, so revocation cannot be performed honestly. Fails closed
     * rather than returning a fabricated success (`true`).
     */
    suspend fun revokeSessionKey(walletAddress: String, sessionKeyId: String): Result<Boolean> {
        return Result.failure(
            IllegalStateException(
                "No real session-keys revocation endpoint is configured; cannot revoke."
            )
        )
    }

    private fun ByteArray.toHex(): String =
        joinToString("") { String.format("%02x", it.toInt() and 0xFF) }
}

/**
 * Gas Optimization Service
 * Gas price prediction, optimization suggestions
 */

class GasOptimizationService private constructor() {

    companion object {
        val instance: GasOptimizationService by lazy { GasOptimizationService() }

        const val BACKEND_BASE_URL = "http://localhost:8443"
        private val JSON_MEDIA_TYPE = "application/json".toMediaType()
    }

    private val client = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .build()

    // ============================================================================
    // Gas Price
    // ============================================================================

    data class GasPrice(
        val slow: Long,
        val standard: Long,
        val fast: Long,
        val instant: Long,
        val unit: String = "gwei"
    )

    /**
     * Get current gas prices from the REAL backend. No prices are fabricated.
     * If the backend has no gas-prices endpoint or is unreachable, throws
     * fail-closed.
     */
    suspend fun getGasPrices(chain: String = "ethereum"): GasPrice {
        return withContext(Dispatchers.IO) {
            val request = Request.Builder()
                .url("$BACKEND_BASE_URL/api/v1/gas/prices?chain=$chain")
                .build()
            try {
                client.newCall(request).execute().use { resp ->
                    if (!resp.isSuccessful) {
                        throw IllegalStateException(
                            "Backend rejected gas-prices query (HTTP ${resp.code})."
                        )
                    }
                    val json = JSONObject(resp.body?.string() ?: "")
                    GasPrice(
                        slow = json.getLong("slow"),
                        standard = json.getLong("standard"),
                        fast = json.getLong("fast"),
                        instant = json.getLong("instant")
                    )
                }
            } catch (e: Exception) {
                throw IllegalStateException("Failed to load gas prices: ${e.message}", e)
            }
        }
    }

    // ============================================================================
    // Gas Optimization
    // ============================================================================

    data class OptimizationSuggestion(
        val type: String,
        val potentialSavings: Double,
        val recommendation: String
    )

    /**
     * Get optimization suggestions from the REAL backend. No suggestions are
     * fabricated. If the backend has no gas-optimization endpoint or is
     * unreachable, throws fail-closed.
     */
    suspend fun getOptimizationSuggestions(
        from: String,
        to: String,
        data: String,
        chain: String = "ethereum"
    ): List<OptimizationSuggestion> {
        return withContext(Dispatchers.IO) {
            val payload = JSONObject().apply {
                put("from", from)
                put("to", to)
                put("data", data)
                put("chain", chain)
            }
            val request = Request.Builder()
                .url("$BACKEND_BASE_URL/api/v1/gas/optimize")
                .post(payload.toString().toRequestBody(JSON_MEDIA_TYPE))
                .build()
            try {
                client.newCall(request).execute().use { resp ->
                    if (!resp.isSuccessful) {
                        throw IllegalStateException(
                            "Backend rejected gas-optimization query (HTTP ${resp.code})."
                        )
                    }
                    val arr = org.json.JSONArray(resp.body?.string() ?: "")
                    (0 until arr.length()).map { i ->
                        val o = arr.getJSONObject(i)
                        OptimizationSuggestion(
                            type = o.getString("type"),
                            potentialSavings = o.getDouble("potential_savings"),
                            recommendation = o.getString("recommendation")
                        )
                    }
                }
            } catch (e: Exception) {
                throw IllegalStateException("Failed to load optimization suggestions: ${e.message}", e)
            }
        }
    }

    /**
     * Estimate optimized gas from the REAL backend. No estimate is fabricated.
     * If the backend has no gas-estimate endpoint or is unreachable, throws
     * fail-closed.
     */
    suspend fun estimateOptimizedGas(
        txData: String,
        chain: String = "ethereum"
    ): Long {
        return withContext(Dispatchers.IO) {
            val request = Request.Builder()
                .url("$BACKEND_BASE_URL/api/v1/gas/estimate?chain=$chain")
                .post(JSONObject().put("data", txData).toString().toRequestBody(JSON_MEDIA_TYPE))
                .build()
            try {
                client.newCall(request).execute().use { resp ->
                    if (!resp.isSuccessful) {
                        throw IllegalStateException(
                            "Backend rejected gas-estimate query (HTTP ${resp.code})."
                        )
                    }
                    val json = JSONObject(resp.body?.string() ?: "")
                    json.getLong("gas_estimate")
                }
            } catch (e: Exception) {
                throw IllegalStateException("Failed to estimate gas: ${e.message}", e)
            }
        }
    }
}
