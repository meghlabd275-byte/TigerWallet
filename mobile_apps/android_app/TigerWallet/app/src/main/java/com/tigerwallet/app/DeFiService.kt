package com.tigerwallet.app

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.math.BigInteger
import java.util.concurrent.TimeUnit

/**
 * DeFi Integration Service
 *
 * Fail-closed: every write operation (supply / borrow / swap) is submitted to
 * the REAL backend (go/wallet_api at BACKEND_BASE_URL) and returns the REAL
 * `tx_hash` reported by the backend. No fabricated `"0x"+UUID` transaction
 * hash is ever returned. If the backend is unreachable or rejects the
 * request, the call throws (fail-closed). Read-only quote/pool queries
 * remain direct calls to the upstream protocol APIs.
 */
class DeFiService private constructor() {

    companion object {
        val instance: DeFiService by lazy { DeFiService() }

        /**
         * Real backend base URL (go/wallet_api). Write operations are delegated
         * here so the returned tx_hash is the on-chain hash, never a fabricated
         * UUID. JWT auth is supplied per-request by the host app.
         */
        const val BACKEND_BASE_URL = "http://localhost:8443"

        private val JSON_MEDIA_TYPE = "application/json".toMediaType()
    }

    private val client = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .build()

    /**
     * JWT used to authenticate backend write requests. Must be set by the host
     * app before any write (supply/borrow/swap) is attempted. When empty the
     * backend call is not attempted and the operation fails closed.
     */
    @Volatile
    var authToken: String = ""

    /**
     * Submits a write operation to the REAL backend and returns the REAL
     * on-chain tx_hash reported by the backend. Never fabricates a hash.
     * Throws on any network/HTTP/logic failure (fail-closed).
     */
    private suspend fun postBackendTx(
        path: String,
        payload: JSONObject,
        txHashField: String = "tx_hash"
    ): String = withContext(Dispatchers.IO) {
        if (authToken.isEmpty()) {
            throw IllegalStateException("Backend auth token not configured; cannot submit DeFi transaction.")
        }
        val request = Request.Builder()
            .url(BACKEND_BASE_URL + path)
            .header("Authorization", "Bearer $authToken")
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
            throw IllegalStateException("Backend unreachable: ${e.message}", e)
        }
        if (code !in 200..299) {
            throw IllegalStateException("Backend rejected request (HTTP $code): $body")
        }
        val json = try {
            JSONObject(body)
        } catch (e: Exception) {
            throw IllegalStateException("Malformed backend response: $body")
        }
        if (json.has("error")) {
            throw IllegalStateException("Backend error: ${json.optString("error", body)}")
        }
        val txHash = json.optString(txHashField, json.optString("txHash", ""))
        if (txHash.isEmpty() || !txHash.startsWith("0x")) {
            throw IllegalStateException("Backend did not return a valid tx_hash: $body")
        }
        txHash
    }

    // ============================================================================
    // Protocol Types
    // ============================================================================

    data class Pool(
        val protocol: String,
        val chain: String,
        val token0: TokenInfo,
        val token1: TokenInfo?,
        val tvl: Double,
        val apy: Double,
        val rewardsApy: Double?,
        val poolAddress: String
    )

    data class TokenInfo(
        val address: String,
        val symbol: String,
        val name: String,
        val decimals: Int,
        val logoUrl: String
    )

    data class Position(
        val protocol: String,
        val chain: String,
        val poolAddress: String,
        val token0: TokenInfo,
        val token1: TokenInfo?,
        val deposited0: Double,
        val deposited1: Double?,
        val valueUsd: Double,
        val apy: Double,
        val rewards: List<Reward>?
    )

    data class Reward(
        val token: TokenInfo,
        val amount: Double,
        val valueUsd: Double,
        val apy: Double
    )

    data class SwapQuote(
        val fromToken: TokenInfo,
        val toToken: TokenInfo,
        val fromAmount: Double,
        val toAmount: Double,
        val toAmountMin: Double,
        val priceImpact: Double,
        val route: List<String>,
        val gasCostUsd: Double,
        val protocol: String
    )

    // ============================================================================
    // Aave Methods
    // ============================================================================

    /**
     * Get Aave pools
     */
    suspend fun getAavePools(chain: String = "ethereum"): List<Pool> {
        return withContext(Dispatchers.IO) {
            try {
                val pools = mutableListOf<Pool>()

                // Aave V3 Pool Addresses
                val aavePools = mapOf(
                    "ethereum" to "0x87870Bca3F3f6335e32cdC0d59b7b238621C8292",
                    "polygon" to "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
                    "avalanche" to "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
                    "arbitrum" to "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
                    "optimism" to "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
                    "base" to "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5"
                )

                // Get reserve data from Aave API
                val request = Request.Builder()
                    .url("https://aave-api-v2.aave.com/data/pools/${aavePools[chain] ?: aavePools["ethereum"]}")
                    .build()

                val response = client.newCall(request).execute()

                if (response.isSuccessful) {
                    val data = JSONObject(response.body?.string() ?: "")
                    val reserves = data.getJSONArray("reserves")

                    for (i in 0 until reserves.length()) {
                        val reserve = reserves.getJSONObject(i)
                        val symbol = reserve.getString("symbol")
                        
                        pools.add(
                            Pool(
                                protocol = "Aave V3",
                                chain = chain,
                                token0 = TokenInfo(
                                    address = reserve.getString("underlyingAsset"),
                                    symbol = symbol,
                                    name = reserve.getString("name"),
                                    decimals = reserve.getInt("decimals"),
                                    logoUrl = "https://raw.githubusercontent.com/spothq/cryptocurrency-icons/master/128/color/${symbol.lowercase()}.png"
                                ),
                                token1 = null,
                                tvl = reserve.optDouble("totalLiquidityUSD", 0.0),
                                apy = reserve.optDouble("supplyAPY", 0.0),
                                rewardsApy = reserve.optDouble("incentivesAPY", null),
                                poolAddress = aavePools[chain] ?: ""
                            )
                        )
                    }
                }
                pools
            } catch (e: Exception) {
                emptyList()
            }
        }
    }

    /**
     * Supply to Aave. Submits the supply operation to the REAL backend
     * (POST /api/v1/send) and returns the REAL on-chain tx_hash. Throws
     * fail-closed if the backend is unreachable or rejects the request.
     */
    suspend fun supplyToAave(
        walletAddress: String,
        poolAddress: String,
        tokenAddress: String,
        amount: BigInteger,
        chain: String
    ): Result<String> {
        return withContext(Dispatchers.IO) {
            try {
                val payload = JSONObject().apply {
                    put("wallet_address", walletAddress)
                    put("chain", chain)
                    put("action", "supply")
                    put("protocol", "aave")
                    put("pool_address", poolAddress)
                    put("token_address", tokenAddress)
                    put("amount", amount.toString())
                }
                Result.success(postBackendTx("/api/v1/send", payload))
            } catch (e: Exception) {
                Result.failure(e)
            }
        }
    }

    /**
     * Borrow from Aave. Submits the borrow operation to the REAL backend
     * (POST /api/v1/send) and returns the REAL on-chain tx_hash. Throws
     * fail-closed if the backend is unreachable or rejects the request.
     */
    suspend fun borrowFromAave(
        walletAddress: String,
        poolAddress: String,
        tokenAddress: String,
        amount: BigInteger,
        interestRateMode: Int, // 1 = stable, 2 = variable
        chain: String
    ): Result<String> {
        return withContext(Dispatchers.IO) {
            try {
                val payload = JSONObject().apply {
                    put("wallet_address", walletAddress)
                    put("chain", chain)
                    put("action", "borrow")
                    put("protocol", "aave")
                    put("pool_address", poolAddress)
                    put("token_address", tokenAddress)
                    put("amount", amount.toString())
                    put("interest_rate_mode", interestRateMode)
                }
                Result.success(postBackendTx("/api/v1/send", payload))
            } catch (e: Exception) {
                Result.failure(e)
            }
        }
    }

    // ============================================================================
    // Uniswap Methods
    // ============================================================================

    /**
     * Get swap quote from Uniswap
     */
    suspend fun getUniswapQuote(
        tokenIn: String,
        tokenOut: String,
        amount: String,
        chain: String = "ethereum"
    ): SwapQuote? {
        return withContext(Dispatchers.IO) {
            try {
                // Use Uniswap V3 Quoter
                val request = Request.Builder()
                    .url("https://api.uniswap.org/v1/quote?tokenIn=$tokenIn&tokenOut=$tokenOut&amount=$amount&type=exactIn")
                    .header("Accept", "application/json")
                    .build()

                val response = client.newCall(request).execute()

                if (response.isSuccessful) {
                    val data = JSONObject(response.body?.string() ?: "")
                    
                    SwapQuote(
                        fromToken = TokenInfo(tokenIn, "", "", 18, ""),
                        toToken = TokenInfo(tokenOut, "", "", 18, ""),
                        fromAmount = amount.toDouble(),
                        toAmount = data.optDouble("quote", 0.0),
                        toAmountMin = data.optDouble("quote", 0.0) * 0.995, // with slippage
                        priceImpact = data.optDouble("priceImpact", 0.0),
                        route = emptyList(),
                        gasCostUsd = data.optDouble("gasUseEstimateUSD", 0.0),
                        protocol = "Uniswap V3"
                    )
                } else null
            } catch (e: Exception) {
                null
            }
        }
    }

    /**
     * Execute swap on Uniswap. Submits the swap to the REAL backend
     * (POST /api/v1/swap/execute) and returns the REAL on-chain tx_hash.
     * Throws fail-closed if the backend is unreachable or rejects the request.
     */
    suspend fun swapOnUniswap(
        walletAddress: String,
        tokenIn: String,
        tokenOut: String,
        amountIn: BigInteger,
        amountOutMin: BigInteger,
        path: List<String>,
        chain: String
    ): Result<String> {
        return withContext(Dispatchers.IO) {
            try {
                val payload = JSONObject().apply {
                    put("wallet_address", walletAddress)
                    put("chain", chain)
                    put("protocol", "uniswap")
                    put("token_in", tokenIn)
                    put("token_out", tokenOut)
                    put("amount_in", amountIn.toString())
                    put("amount_out_min", amountOutMin.toString())
                    put("path", JSONArray(path))
                }
                Result.success(postBackendTx("/api/v1/swap/execute", payload))
            } catch (e: Exception) {
                Result.failure(e)
            }
        }
    }

    // ============================================================================
    // Compound Methods
    // ============================================================================

    /**
     * Get Compound pools
     */
    suspend fun getCompoundPools(): List<Pool> {
        return withContext(Dispatchers.IO) {
            try {
                val request = Request.Builder()
                    .url("https://api.compound.finance/api/v2/ctoken")
                    .build()

                val response = client.newCall(request).execute()

                if (response.isSuccessful) {
                    val data = JSONObject(response.body?.string() ?: "")
                    val cTokens = data.getJSONArray("cToken")
                    val pools = mutableListOf<Pool>()

                    for (i in 0 until cTokens.length()) {
                        val cToken = cTokens.getJSONObject(i)
                        val underlying = cToken.getJSONObject("underlying")

                        pools.add(
                            Pool(
                                protocol = "Compound V3",
                                chain = "ethereum",
                                token0 = TokenInfo(
                                    address = underlying.getString("address"),
                                    symbol = underlying.getString("symbol"),
                                    name = underlying.getString("name"),
                                    decimals = underlying.getInt("decimals"),
                                    logoUrl = ""
                                ),
                                token1 = null,
                                tvl = cToken.optDouble("totalSupplyUSD", 0.0),
                                apy = cToken.optDouble("supplyRate", 0.0) * 100,
                                rewardsApy = null,
                                poolAddress = cToken.getString("address")
                            )
                        )
                    }
                    pools
                } else emptyList()
            } catch (e: Exception) {
                emptyList()
            }
        }
    }

    // ============================================================================
    // Yearn Vaults
    // ============================================================================

    /**
     * Get Yearn vaults
     */
    suspend fun getYearnVaults(): List<Pool> {
        return withContext(Dispatchers.IO) {
            try {
                val request = Request.Builder()
                    .url("https://api.yearn.finance/v1/chains/1/vaults")
                    .build()

                val response = client.newCall(request).execute()

                if (response.isSuccessful) {
                    val vaults = JSONArray(response.body?.string() ?: "")
                    val pools = mutableListOf<Pool>()

                    for (i in 0 until vaults.length()) {
                        val vault = vaults.getJSONObject(i)
                        
                        pools.add(
                            Pool(
                                protocol = "Yearn Vault",
                                chain = "ethereum",
                                token0 = TokenInfo(
                                    address = vault.getString("token"),
                                    symbol = vault.getString("symbol"),
                                    name = vault.getString("name"),
                                    decimals = vault.getInt("decimals"),
                                    logoUrl = ""
                                ),
                                token1 = null,
                                tvl = vault.optDouble("tvl", 0.0),
                                apy = vault.optDouble("apy", 0.0),
                                rewardsApy = null,
                                poolAddress = vault.getString("address")
                            )
                        )
                    }
                    pools
                } else emptyList()
            } catch (e: Exception) {
                emptyList()
            }
        }
    }

    // ============================================================================
    // Portfolio Aggregation
    // ============================================================================

    /**
     * Get all DeFi positions for a wallet. The backend currently exposes no
     * positions-aggregation endpoint, so this fails closed rather than
     * returning a fabricated (empty) list. When a real endpoint exists it
     * should be queried here.
     */
    suspend fun getAllPositions(walletAddress: String): List<Position> {
        return withContext(Dispatchers.IO) {
            throw IllegalStateException(
                "No real DeFi positions endpoint is configured; cannot aggregate positions."
            )
        }
    }

    // ============================================================================
    // Utility Methods
    // ============================================================================

    /**
     * Calculate APY from APR
     */
    fun apyFromApr(apr: Double): Double {
        return Math.pow(1 + apr / 100, 12 * 30 / 365) - 1
    }

    /**
     * Format percentage
     */
    fun formatPercentage(value: Double): String {
        return String.format("%.2f%%", value)
    }

    /**
     * Format TVL
     */
    fun formatTvl(value: Double): String {
        return when {
            value >= 1_000_000_000 -> String.format("$%.2fB", value / 1_000_000_000)
            value >= 1_000_000 -> String.format("$%.2fM", value / 1_000_000)
            value >= 1_000 -> String.format("$%.2fK", value / 1_000)
            else -> String.format("$%.2f", value)
        }
    }
}
