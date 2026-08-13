/**
 * MasterWalletService - Android Implementation
 * Delegates HD wallet creation, balance, and signing to the canonical MasterWallet
 * backend at :8450 (see CANONICAL_API_CONTRACT.md). The backend performs the real
 * secp256k1 key derivation + broadcast; this service never fabricates keys, balances,
 * or signatures locally.
 */

package com.tigermaster.services

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject
import java.math.BigInteger
import java.net.HttpURLConnection
import java.net.URL

class MasterWalletService {
    // Canonical MasterWallet backend (see CANONICAL_API_CONTRACT.md)
    private var baseUrl: String = "http://localhost:8450"
    private var authToken: String? = null

    // Locally cached chain metadata fetched from GET /api/v1/chains (no hardcoded RPC URLs).
    private val chainCache = mutableMapOf<Int, ChainConfig>()

    fun setBaseUrl(url: String) {
        baseUrl = url.trimEnd('/')
    }

    fun setAuthToken(token: String?) {
        authToken = token
    }

    private fun requireToken(): String =
        authToken ?: throw IllegalStateException("Not authenticated: JWT token required")

    /**
     * Create a master wallet via POST /api/v1/master-wallet. The backend creates the
     * HD wallet and returns the mnemonic exactly once.
     */
    suspend fun generateWallet(name: String, password: String, chainId: Long = 1L): WalletResult =
        withContext(Dispatchers.IO) {
            try {
                val body = JSONObject()
                    .put("name", name)
                    .put("password", password)
                    .put("chain_id", chainId)
                    .toString()
                val resp = apiPost("/api/v1/master-wallet", body)
                    ?: return@withContext WalletResult(success = false, error = "Wallet creation failed")
                val json = JSONObject(resp)
                WalletResult(
                    success = true,
                    walletId = json.optString("id", json.optString("wallet_id", "")),
                    address = json.optString("address", ""),
                    mnemonic = json.optString("mnemonic", "")
                )
            } catch (e: Exception) {
                WalletResult(success = false, error = e.message)
            }
        }

    /**
     * No canonical import endpoint exists; fail closed rather than fabricating keys.
     */
    suspend fun importWallet(mnemonic: String, password: String): WalletResult =
        withContext(Dispatchers.IO) {
            WalletResult(success = false, error = "Wallet import is not supported by the canonical backend")
        }

    /**
     * GET /api/v1/master-wallet/:id/balance returns real RPC native + token balances.
     */
    suspend fun getBalance(walletId: String, chainId: Int): BalanceResult =
        withContext(Dispatchers.IO) {
            try {
                val resp = apiGet("/api/v1/master-wallet/$walletId/balance")
                    ?: return@withContext BalanceResult(success = false, error = "Balance fetch failed")
                val json = JSONObject(resp)
                val native = json.optJSONObject("native") ?: json
                BalanceResult(
                    success = true,
                    balance = native.optDouble("balance", native.optDouble("amount", 0.0)),
                    symbol = native.optString("symbol", ""),
                    decimals = native.optInt("decimals", 18)
                )
            } catch (e: Exception) {
                BalanceResult(success = false, error = e.message)
            }
        }

    /**
     * Token balances come from the canonical balance endpoint (real RPC), never fabricated.
     */
    suspend fun getTokenBalance(walletId: String, chainId: Int, tokenAddress: String): TokenBalanceResult =
        withContext(Dispatchers.IO) {
            try {
                val resp = apiGet("/api/v1/master-wallet/$walletId/balance")
                    ?: return@withContext TokenBalanceResult(success = false, error = "Balance fetch failed")
                val json = JSONObject(resp)
                val tokens = json.optJSONArray("tokens") ?: JSONArray()
                for (i in 0 until tokens.length()) {
                    val t = tokens.getJSONObject(i)
                    if (t.optString("address", "").equals(tokenAddress, ignoreCase = true)) {
                        return@withContext TokenBalanceResult(
                            success = true,
                            balance = t.optString("balance", "0"),
                            symbol = t.optString("symbol", ""),
                            decimals = t.optInt("decimals", 18)
                        )
                    }
                }
                TokenBalanceResult(success = false, error = "Token not found in balances")
            } catch (e: Exception) {
                TokenBalanceResult(success = false, error = e.message)
            }
        }

    /**
     * POST /api/v1/master-wallet/:id/sign performs real secp256k1 signing + broadcast.
     */
    suspend fun sendTransaction(
        walletId: String,
        chainId: Int,
        toAddress: String,
        amount: BigInteger,
        password: String,
        token: String? = null
    ): TransactionResult = withContext(Dispatchers.IO) {
        try {
            val body = JSONObject()
                .put("to", toAddress)
                .put("amount", amount.toString())
                .put("password", password)
                .apply { token?.let { put("token", it) } }
                .toString()
            val resp = apiPost("/api/v1/master-wallet/$walletId/sign", body)
                ?: return@withContext TransactionResult(success = false, error = "Sign request failed")
            val json = JSONObject(resp)
            TransactionResult(
                success = true,
                txHash = json.optString("transaction_hash", json.optString("hash", "")),
                from = json.optString("from", ""),
                to = toAddress,
                amount = amount.toString()
            )
        } catch (e: Exception) {
            TransactionResult(success = false, error = e.message)
        }
    }

    /**
     * GET /api/v1/chains returns the supported chains from the backend (no hardcoded RPC URLs).
     */
    suspend fun getSupportedChains(): List<ChainConfig> = withContext(Dispatchers.IO) {
        try {
            val resp = apiGet("/api/v1/chains") ?: return@withContext chainCache.values.toList()
            val json = JSONObject(resp)
            val arr = json.optJSONArray("chains") ?: JSONArray(resp)
            val list = mutableListOf<ChainConfig>()
            for (i in 0 until arr.length()) {
                val obj = arr.getJSONObject(i)
                val cfg = ChainConfig(
                    id = obj.optInt("id", obj.optInt("chain_id", 0)),
                    name = obj.optString("name", ""),
                    symbol = obj.optString("symbol", ""),
                    rpcUrl = obj.optString("rpc_url", obj.optString("rpcUrl", "")),
                    explorerUrl = obj.optString("explorer_url", obj.optString("explorerUrl", "")),
                    decimals = obj.optInt("decimals", 18),
                    isEVM = obj.optBoolean("is_evm", obj.optBoolean("isEVM", true))
                )
                chainCache[cfg.id] = cfg
                list.add(cfg)
            }
            list
        } catch (e: Exception) {
            chainCache.values.toList()
        }
    }

    suspend fun deleteWallet(walletId: String): Boolean = withContext(Dispatchers.IO) {
        apiDelete("/api/v1/master-wallet/$walletId")
    }

    // -- HTTP helpers (Bearer JWT auth against the canonical backend) --

    private fun apiGet(endpoint: String): String? {
        val token = try { requireToken() } catch (e: Exception) { return null }
        return try {
            val conn = (URL("$baseUrl$endpoint").openConnection() as HttpURLConnection).apply {
                requestMethod = "GET"
                setRequestProperty("Authorization", "Bearer $token")
                connectTimeout = 10000
                readTimeout = 10000
            }
            if (conn.responseCode in 200..299) conn.inputStream.bufferedReader().readText() else null
        } catch (e: Exception) { null }
    }

    private fun apiPost(endpoint: String, body: String): String? {
        val token = try { requireToken() } catch (e: Exception) { return null }
        return try {
            val conn = (URL("$baseUrl$endpoint").openConnection() as HttpURLConnection).apply {
                requestMethod = "POST"
                setRequestProperty("Content-Type", "application/json")
                setRequestProperty("Authorization", "Bearer $token")
                doOutput = true
                connectTimeout = 10000
                readTimeout = 10000
            }
            conn.outputStream.use { it.write(body.toByteArray()) }
            if (conn.responseCode in 200..299) conn.inputStream.bufferedReader().readText() else null
        } catch (e: Exception) { null }
    }

    private fun apiDelete(endpoint: String): Boolean {
        val token = try { requireToken() } catch (e: Exception) { return false }
        return try {
            val conn = (URL("$baseUrl$endpoint").openConnection() as HttpURLConnection).apply {
                requestMethod = "DELETE"
                setRequestProperty("Authorization", "Bearer $token")
                connectTimeout = 10000
                readTimeout = 10000
            }
            conn.responseCode in 200..299
        } catch (e: Exception) { false }
    }
}

// Data classes

data class ChainConfig(
    val id: Int,
    val name: String,
    val symbol: String,
    val rpcUrl: String,
    val explorerUrl: String,
    val decimals: Int,
    val isEVM: Boolean
)

data class WalletResult(
    val success: Boolean,
    val walletId: String? = null,
    val address: String? = null,
    val mnemonic: String? = null,
    val error: String? = null
)

data class BalanceResult(
    val success: Boolean,
    val balance: Double = 0.0,
    val symbol: String = "",
    val decimals: Int = 18,
    val error: String? = null
)

data class TokenBalanceResult(
    val success: Boolean,
    val balance: String = "0",
    val symbol: String = "",
    val decimals: Int = 18,
    val error: String? = null
)

data class TransactionResult(
    val success: Boolean,
    val txHash: String? = null,
    val from: String? = null,
    val to: String? = null,
    val amount: String? = null,
    val error: String? = null
)
