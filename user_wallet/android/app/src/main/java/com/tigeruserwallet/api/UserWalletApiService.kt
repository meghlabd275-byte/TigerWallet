package com.tigeruserwallet.api

import okhttp3.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.io.IOException
import java.util.concurrent.TimeUnit

/**
 * TigerWallet UserWallet API Service - Complete Android Implementation
 * Handles all API communications with the UserWallet backend
 */
class UserWalletApiService(private val baseUrl: String, private var authToken: String? = null) {
    
    companion object {
        private const val CONNECT_TIMEOUT = 30L
        private const val READ_TIMEOUT = 30L
        private const val WRITE_TIMEOUT = 30L
    }
    
    private val client: OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(CONNECT_TIMEOUT, TimeUnit.SECONDS)
        .readTimeout(READ_TIMEOUT, TimeUnit.SECONDS)
        .writeTimeout(WRITE_TIMEOUT, TimeUnit.SECONDS)
        .build()
    
    private val jsonMediaType = "application/json".toMediaType()
    
    private fun getHeaders(): Headers {
        return Headers.Builder()
            .add("Content-Type", "application/json")
            .add("Accept", "application/json")
            .apply {
                authToken?.let { add("Authorization", "Bearer $it") }
            }
            .build()
    }
    
    // ==================== WALLETS ====================
    
    /**
     * Create user wallet
     * POST /api/v1/wallet
     */
    fun createWallet(chain: String, walletType: String?, callback: ApiCallback<UserWallet>) {
        val body = JSONObject().apply {
            put("chain", chain)
            walletType?.let { put("wallet_type", it) }
        }
        
        val request = Request.Builder()
            .url("$baseUrl/api/v1/wallet")
            .post(body.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseUserWallet(it) }
    }
    
    /**
     * Get user wallets
     * GET /api/v1/wallet
     */
    fun getWallets(callback: ApiCallback<ListResponse<UserWallet>>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/wallet")
            .get()
            .headers(getHeaders())
            .build()
        
        executeListRequest(request, callback) { parseUserWalletList(it) }
    }
    
    /**
     * Get wallet details
     * GET /api/v1/wallet/:id
     */
    fun getWallet(walletId: String, callback: ApiCallback<UserWallet>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/wallet/$walletId")
            .get()
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseUserWallet(it) }
    }
    
    /**
     * Get wallet balance
     * GET /api/v1/wallet/:id/balance
     */
    fun getWalletBalance(walletId: String, callback: ApiCallback<UserBalanceResponse>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/wallet/$walletId/balance")
            .get()
            .headers(getHeaders())
            .build()
        
        client.newCall(request).enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                callback.onError(e.message ?: "Network error")
            }
            override fun onResponse(call: Call, response: Response) {
                try {
                    val json = JSONObject(response.body?.string() ?: "{}")
                    if (response.isSuccessful) {
                        val balances = mutableListOf<UserBalance>()
                        val balancesArray = json.optJSONArray("balances")
                        balancesArray?.let {
                            for (i in 0 until it.length()) {
                                val b = it.getJSONObject(i)
                                balances.add(UserBalance(
                                    token = b.optString("token", ""),
                                    balance = b.optString("balance", "0"),
                                    available = b.optString("available", "0"),
                                    locked = b.optString("locked", "0"),
                                    usdValue = b.optString("usd_value", "0")
                                ))
                            }
                        }
                        callback.onSuccess(UserBalanceResponse(balances))
                    } else {
                        callback.onError(json.optString("error", "Request failed"))
                    }
                } catch (e: Exception) {
                    callback.onError(e.message ?: "Parse error")
                }
            }
        })
    }
    
    // ==================== TRANSACTIONS ====================
    
    /**
     * Send transaction
     * POST /api/v1/wallet/:id/send
     */
    fun sendTransaction(walletId: String, to: String, amount: Double, token: String, chain: String, callback: ApiCallback<UserTransaction>) {
        val body = JSONObject().apply {
            put("to", to)
            put("amount", amount)
            put("token", token)
            put("chain", chain)
        }
        
        val request = Request.Builder()
            .url("$baseUrl/api/v1/wallet/$walletId/send")
            .post(body.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseUserTransaction(it) }
    }
    
    /**
     * Get transactions
     * GET /api/v1/wallet/:id/transactions
     */
    fun getTransactions(walletId: String, callback: ApiCallback<ListResponse<UserTransaction>>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/wallet/$walletId/transactions")
            .get()
            .headers(getHeaders())
            .build()
        
        executeListRequest(request, callback) { parseUserTransactionList(it) }
    }
    
    /**
     * Get transaction details
     * GET /api/v1/transactions/:id
     */
    fun getTransaction(txId: String, callback: ApiCallback<UserTransaction>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/transactions/$txId")
            .get()
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseUserTransaction(it) }
    }
    
    // ==================== SWAPS ====================
    
    /**
     * Swap tokens
     * POST /api/v1/wallet/swap
     */
    fun swapTokens(fromToken: String, toToken: String, amount: Double, slippage: Double?, callback: ApiCallback<Swap>) {
        val body = JSONObject().apply {
            put("from_token", fromToken)
            put("to_token", toToken)
            put("amount", amount)
            slippage?.let { put("slippage", it) }
        }
        
        val request = Request.Builder()
            .url("$baseUrl/api/v1/wallet/swap")
            .post(body.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { json ->
            Swap(
                id = json.optString("id", ""),
                fromToken = json.optString("from_token", ""),
                toToken = json.optString("to_token", ""),
                fromAmount = json.optString("from_amount", "0"),
                toAmount = json.optString("to_amount", "0"),
                txHash = json.optString("tx_hash", ""),
                status = json.optString("status", ""),
                createdAt = json.optString("created_at", "")
            )
        }
    }
    
    /**
     * Get swap quote
     * GET /api/v1/wallet/swap/quote
     */
    fun getSwapQuote(fromToken: String, toToken: String, amount: String, callback: ApiCallback<SwapQuote>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/wallet/swap/quote?from_token=$fromToken&to_token=$toToken&amount=$amount")
            .get()
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { json ->
            SwapQuote(
                fromToken = json.optString("from_token", ""),
                toToken = json.optString("to_token", ""),
                fromAmount = json.optString("from_amount", "0"),
                toAmount = json.optString("to_amount", "0"),
                priceImpact = json.optString("price_impact", "0"),
                slippage = json.optString("slippage", "0"),
                expiresAt = json.optString("expires_at", "")
            )
        }
    }
    
    // ==================== STAKING ====================
    
    /**
     * Stake tokens
     * POST /api/v1/wallet/stake
     */
    fun stake(token: String, amount: Double, chain: String, callback: ApiCallback<Stake>) {
        val body = JSONObject().apply {
            put("token", token)
            put("amount", amount)
            put("chain", chain)
        }
        
        val request = Request.Builder()
            .url("$baseUrl/api/v1/wallet/stake")
            .post(body.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { json ->
            Stake(
                id = json.optString("id", ""),
                token = json.optString("token", ""),
                amount = json.optString("amount", "0"),
                reward = json.optString("reward", "0"),
                chain = json.optString("chain", ""),
                status = json.optString("status", ""),
                stakedAt = json.optString("staked_at", "")
            )
        }
    }
    
    /**
     * Unstake
     * POST /api/v1/wallet/unstake
     */
    fun unstake(stakeId: String, callback: ApiCallback<Stake>) {
        val body = JSONObject().put("stake_id", stakeId)
        
        val request = Request.Builder()
            .url("$baseUrl/api/v1/wallet/unstake")
            .post(body.toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { json ->
            Stake(
                id = json.optString("id", stakeId),
                token = json.optString("token", ""),
                amount = json.optString("amount", "0"),
                reward = json.optString("reward", "0"),
                chain = json.optString("chain", ""),
                status = json.optString("status", ""),
                stakedAt = json.optString("staked_at", "")
            )
        }
    }
    
    /**
     * Get stakes
     * GET /api/v1/wallet/stakes
     */
    fun getStakes(callback: ApiCallback<ListResponse<Stake>>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/wallet/stakes")
            .get()
            .headers(getHeaders())
            .build()
        
        executeListRequest(request, callback) { jsonArray ->
            (0 until jsonArray.length()).map { i ->
                val json = jsonArray.getJSONObject(i)
                Stake(
                    id = json.optString("id", ""),
                    token = json.optString("token", ""),
                    amount = json.optString("amount", "0"),
                    reward = json.optString("reward", "0"),
                    chain = json.optString("chain", ""),
                    status = json.optString("status", ""),
                    stakedAt = json.optString("staked_at", "")
                )
            }
        }
    }
    
    // ==================== NFTs ====================
    
    /**
     * Get NFTs
     * GET /api/v1/wallet/nfts
     */
    fun getNFTs(callback: ApiCallback<ListResponse<NFT>>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/wallet/nfts")
            .get()
            .headers(getHeaders())
            .build()
        
        executeListRequest(request, callback) { jsonArray ->
            (0 until jsonArray.length()).map { i ->
                val json = jsonArray.getJSONObject(i)
                NFT(
                    id = json.optString("id", ""),
                    tokenId = json.optString("token_id", ""),
                    contract = json.optString("contract", ""),
                    name = json.optString("name", ""),
                    imageUrl = json.optString("image_url", ""),
                    chain = json.optString("chain", "")
                )
            }
        }
    }
    
    /**
     * Transfer NFT
     * POST /api/v1/wallet/nft/transfer
     */
    fun transferNFT(nftId: String, toAddress: String, callback: ApiCallback<UserTransaction>) {
        val body = JSONObject().apply {
            put("nft_id", nftId)
            put("to_address", toAddress)
        }
        
        val request = Request.Builder()
            .url("$baseUrl/api/v1/wallet/nft/transfer")
            .post(body.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseUserTransaction(it) }
    }
    
    // ==================== PORTFOLIO ====================
    
    /**
     * Get portfolio
     * GET /api/v1/wallet/portfolio
     */
    fun getPortfolio(callback: ApiCallback<Portfolio>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/wallet/portfolio")
            .get()
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { json ->
            val assetsArray = json.optJSONArray("assets")
            val assets = mutableListOf<PortfolioAsset>()
            assetsArray?.let {
                for (i in 0 until it.length()) {
                    val a = it.getJSONObject(i)
                    assets.add(PortfolioAsset(
                        token = a.optString("token", ""),
                        balance = a.optString("balance", "0"),
                        valueUsd = a.optString("value_usd", "0"),
                        percentage = a.optDouble("percentage", 0.0)
                    ))
                }
            }
            Portfolio(
                totalValueUsd = json.optString("total_value_usd", "0"),
                change24h = json.optDouble("change_24h", 0.0),
                change7d = json.optDouble("change_7d", 0.0),
                assets = assets
            )
        }
    }
    
    /**
     * Get history
     * GET /api/v1/wallet/history
     */
    fun getHistory(type: String?, callback: ApiCallback<ListResponse<HistoryItem>>) {
        val url = if (type != null) {
            "$baseUrl/api/v1/wallet/history?type=$type"
        } else {
            "$baseUrl/api/v1/wallet/history"
        }
        
        val request = Request.Builder()
            .url(url)
            .get()
            .headers(getHeaders())
            .build()
        
        executeListRequest(request, callback) { jsonArray ->
            (0 until jsonArray.length()).map { i ->
                val json = jsonArray.getJSONObject(i)
                HistoryItem(
                    id = json.optString("id", ""),
                    type = json.optString("type", ""),
                    subtype = json.optString("subtype", ""),
                    token = json.optString("token", ""),
                    amount = json.optString("amount", "0"),
                    status = json.optString("status", ""),
                    createdAt = json.optString("created_at", "")
                )
            }
        }
    }
    
    // ==================== HELPERS ====================
    
    private fun <T> executeRequest(request: Request, callback: ApiCallback<T>, parser: (JSONObject) -> T) {
        client.newCall(request).enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                callback.onError(e.message ?: "Network error")
            }
            override fun onResponse(call: Call, response: Response) {
                try {
                    val json = JSONObject(response.body?.string() ?: "{}")
                    if (response.isSuccessful) {
                        callback.onSuccess(parser(json))
                    } else {
                        callback.onError(json.optString("error", "Request failed"))
                    }
                } catch (e: Exception) {
                    callback.onError(e.message ?: "Parse error")
                }
            }
        })
    }
    
    private fun <T> executeListRequest(request: Request, callback: ApiCallback<ListResponse<T>>, parser: (JSONArray) -> List<T>) {
        client.newCall(request).enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                callback.onError(e.message ?: "Network error")
            }
            override fun onResponse(call: Call, response: Response) {
                try {
                    val json = JSONObject(response.body?.string() ?: "{}")
                    if (response.isSuccessful) {
                        val data = parser(json.optJSONArray("data") ?: JSONArray())
                        callback.onSuccess(ListResponse(
                            data = data,
                            total = json.optInt("total", 0),
                            page = json.optInt("page", 1),
                            pageSize = json.optInt("page_size", 20)
                        ))
                    } else {
                        callback.onError(json.optString("error", "Request failed"))
                    }
                } catch (e: Exception) {
                    callback.onError(e.message ?: "Parse error")
                }
            }
        })
    }
    
    // Parsers
    private fun parseUserWallet(json: JSONObject) = UserWallet(
        id = json.optString("id", ""),
        address = json.optString("address", ""),
        chain = json.optString("chain", ""),
        walletType = json.optString("wallet_type", ""),
        isActive = json.optBoolean("is_active", true),
        balance = json.optString("balance", "0"),
        createdAt = json.optString("created_at", "")
    )
    
    private fun parseUserWalletList(json: JSONArray): List<UserWallet> = (0 until json.length()).map { parseUserWallet(json.getJSONObject(it)) }
    
    private fun parseUserTransaction(json: JSONObject) = UserTransaction(
        id = json.optString("id", ""),
        txHash = json.optString("tx_hash", ""),
        walletId = json.optString("wallet_id", ""),
        type = json.optString("type", ""),
        to = json.optString("to", ""),
        from = json.optString("from", ""),
        amount = json.optString("amount", "0"),
        token = json.optString("token", ""),
        fee = json.optString("fee", "0"),
        status = json.optString("status", ""),
        createdAt = json.optString("created_at", "")
    )
    
    private fun parseUserTransactionList(json: JSONArray): List<UserTransaction> = (0 until json.length()).map { parseUserTransaction(json.getJSONObject(it)) }
}

// API Callback
interface UserApiCallback<T> {
    fun onSuccess(data: T)
    fun onError(error: String)
}

// Data classes
data class UserWallet(val id: String, val address: String, val chain: String, val walletType: String, val isActive: Boolean, val balance: String, val createdAt: String)
data class UserBalance(val token: String, val balance: String, val available: String, val locked: String, val usdValue: String)
data class UserBalanceResponse(val balances: List<UserBalance>)
data class UserTransaction(val id: String, val txHash: String, val walletId: String, val type: String, val to: String, val from: String, val amount: String, val token: String, val fee: String, val status: String, val createdAt: String)
data class Swap(val id: String, val fromToken: String, val toToken: String, val fromAmount: String, val toAmount: String, val txHash: String, val status: String, val createdAt: String)
data class SwapQuote(val fromToken: String, val toToken: String, val fromAmount: String, val toAmount: String, val priceImpact: String, val slippage: String, val expiresAt: String)
data class Stake(val id: String, val token: String, val amount: String, val reward: String, val chain: String, val status: String, val stakedAt: String)
data class NFT(val id: String, val tokenId: String, val contract: String, val name: String, val imageUrl: String, val chain: String)
data class PortfolioAsset(val token: String, val balance: String, val valueUsd: String, val percentage: Double)
data class Portfolio(val totalValueUsd: String, val change24h: Double, val change7d: Double, val assets: List<PortfolioAsset>)
data class HistoryItem(val id: String, val type: String, val subtype: String, val token: String, val amount: String, val status: String, val createdAt: String)
data class ListResponse<T>(val data: List<T>, val total: Int, val page: Int, val pageSize: Int)
