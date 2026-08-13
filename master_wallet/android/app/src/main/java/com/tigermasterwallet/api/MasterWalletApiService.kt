package com.tigermasterwallet.api

import okhttp3.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.io.IOException
import java.util.concurrent.TimeUnit

/**
 * TigerWallet MasterWallet API Service - Complete Android Implementation
 * Handles all API communications with the MasterWallet backend
 */
class MasterWalletApiService(private val baseUrl: String, private var authToken: String? = null) {

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
     * Create a new master wallet (backend creates HD wallet, returns mnemonic once)
     * POST /api/v1/master-wallet — {name, password, chain_id}
     */
    fun createWallet(name: String, chain: String, walletType: String, password: String, chainId: Long, callback: ApiCallback<Wallet>) {
        val body = JSONObject().apply {
            put("name", name)
            put("password", password)
            put("chain_id", chainId)
        }

        val request = Request.Builder()
            .url("$baseUrl/api/v1/master-wallet")
            .post(body.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()

        executeRequest(request, callback) { parseWallet(it) }
    }

    /**
     * List all master wallets
     * GET /api/v1/master-wallet
     */
    fun listWallets(chain: String? = null, walletType: String? = null, callback: ApiCallback<ListResponse<Wallet>>) {
        val params = mutableListOf<String>()
        chain?.let { params.add("chain=$it") }
        walletType?.let { params.add("wallet_type=$it") }

        val url = if (params.isNotEmpty()) {
            "$baseUrl/api/v1/master-wallet?${params.joinToString("&")}"
        } else {
            "$baseUrl/api/v1/master-wallet"
        }

        val request = Request.Builder()
            .url(url)
            .get()
            .headers(getHeaders())
            .build()

        executeListRequest(request, callback) { parseWalletList(it) }
    }

    /**
     * Get wallet details
     * GET /api/v1/master-wallet/:id
     */
    fun getWallet(walletId: String, callback: ApiCallback<Wallet>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/master-wallet/$walletId")
            .get()
            .headers(getHeaders())
            .build()

        executeRequest(request, callback) { parseWallet(it) }
    }

    /**
     * Get wallet balance
     * GET /api/v1/master-wallet/:id/balance
     */
    fun getWalletBalance(walletId: String, token: String? = null, callback: ApiCallback<BalanceResponse>) {
        val url = if (token != null) {
            "$baseUrl/api/v1/master-wallet/$walletId/balance?token=$token"
        } else {
            "$baseUrl/api/v1/master-wallet/$walletId/balance"
        }

        val request = Request.Builder()
            .url(url)
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
                        val balances = mutableListOf<Balance>()
                        val balancesArray = json.optJSONArray("balances")
                        balancesArray?.let {
                            for (i in 0 until it.length()) {
                                val b = it.getJSONObject(i)
                                balances.add(Balance(
                                    token = b.optString("token", ""),
                                    balance = b.optString("balance", "0"),
                                    available = b.optString("available", "0"),
                                    locked = b.optString("locked", "0"),
                                    usdValue = b.optString("usd_value", "0")
                                ))
                            }
                        }
                        callback.onSuccess(BalanceResponse(walletId = walletId, balances = balances))
                    } else {
                        callback.onError(json.optString("error", "Request failed"))
                    }
                } catch (e: Exception) {
                    callback.onError(e.message ?: "Parse error")
                }
            }
        })
    }

    /**
     * Update wallet
     * PUT /api/v1/master-wallet/:id
     */
    fun updateWallet(walletId: String, name: String?, isActive: Boolean?, callback: ApiCallback<Wallet>) {
        val body = JSONObject()
        name?.let { body.put("name", it) }
        isActive?.let { body.put("is_active", it) }

        val request = Request.Builder()
            .url("$baseUrl/api/v1/master-wallet/$walletId")
            .put(body.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()

        executeRequest(request, callback) { parseWallet(it) }
    }

    /**
     * Delete wallet
     * DELETE /api/v1/master-wallet/:id
     */
    fun deleteWallet(walletId: String, callback: ApiCallback<Unit>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/master-wallet/$walletId")
            .delete()
            .headers(getHeaders())
            .build()

        client.newCall(request).enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                callback.onError(e.message ?: "Network error")
            }
            override fun onResponse(call: Call, response: Response) {
                if (response.isSuccessful) {
                    callback.onSuccess(Unit)
                } else {
                    callback.onError("Failed to delete wallet")
                }
            }
        })
    }

    // ==================== TRANSACTIONS ====================

    /**
     * Send transaction (real secp256k1 sign + broadcast on backend)
     * POST /api/v1/master-wallet/:id/sign  — {to, amount, password, token?}
     */
    fun sendTransaction(
        walletId: String,
        to: String,
        amount: Double,
        token: String,
        chain: String,
        password: String,
        callback: ApiCallback<MasterTransaction>
    ) {
        val body = JSONObject().apply {
            put("to", to)
            put("amount", amount)
            put("password", password)
            if (token.isNotEmpty()) put("token", token)
        }

        val request = Request.Builder()
            .url("$baseUrl/api/v1/master-wallet/$walletId/sign")
            .post(body.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()

        executeRequest(request, callback) { parseMasterTransaction(it) }
    }

    /**
     * Get transactions
     * GET /api/v1/master-wallet/:id/transactions
     */
    fun getTransactions(walletId: String, status: String? = null, callback: ApiCallback<ListResponse<MasterTransaction>>) {
        val url = if (status != null) {
            "$baseUrl/api/v1/master-wallet/$walletId/transactions?status=$status"
        } else {
            "$baseUrl/api/v1/master-wallet/$walletId/transactions"
        }

        val request = Request.Builder()
            .url(url)
            .get()
            .headers(getHeaders())
            .build()

        executeListRequest(request, callback) { parseMasterTransactionList(it) }
    }

    /**
     * Get transaction details. The canonical contract exposes transaction
     * history only as a list (GET /api/v1/master-wallet/:id/transactions) and
     * approve/reject actions — there is no single-tx fetch route. Calling this
     * therefore fails closed rather than hitting a non-canonical endpoint.
     */
    fun getTransaction(txId: String, callback: ApiCallback<MasterTransaction>) {
        callback.onError(
            "getTransaction('$txId') has no canonical backend route; " +
            "use listTransactions(walletId) instead"
        )
    }

    /**
     * Cancel transaction (mapped to canonical reject route)
     * POST /api/v1/master-wallet/:id/transactions/:tid/reject
     */
    fun cancelTransaction(txId: String, callback: ApiCallback<MasterTransaction>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/master-wallet/transactions/$txId/reject")
            .post("{}".toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()

        executeRequest(request, callback) { parseMasterTransaction(it) }
    }

    // ==================== GAS ====================

    /**
     * Get gas price
     * GET /api/v1/gas?chain_id=N  → {gas_price, max_fee, priority_fee}
     */
    fun getGasPrice(chainId: Long, callback: ApiCallback<GasPrice>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/gas?chain_id=$chainId")
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
                        val gasPrice = json.optString("gas_price", "0")
                        callback.onSuccess(GasPrice(
                            chain = chainId.toString(),
                            slow = gasPrice,
                            standard = json.optString("max_fee", gasPrice),
                            fast = json.optString("priority_fee", gasPrice),
                            unit = "gwei"
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

    /**
     * Set gas strategy. The canonical contract only exposes gas *reading*
     * (GET /api/v1/gas) — there is no gas-strategy persistence route. Calling
     * this fails closed rather than POSTing to a non-canonical endpoint.
     */
    fun setGasStrategy(chain: String, strategy: String, maxGas: String?, callback: ApiCallback<Unit>) {
        callback.onError(
            "setGasStrategy has no canonical backend route; " +
            "read gas via getGasPrice(chainId) instead"
        )
    }

    // ==================== MULTISIG ====================

    /**
     * Create multisig wallet
     * POST /api/v1/master-wallet/:id/multisig/wallets
     */
    fun createMultisig(name: String, chain: String, signers: List<String>, requiredSigs: Int, callback: ApiCallback<MultisigWallet>) {
        val body = JSONObject().apply {
            put("name", name)
            put("owners", JSONArray(signers))
            put("threshold", requiredSigs)
        }

        val request = Request.Builder()
            .url("$baseUrl/api/v1/master-wallet/multisig/wallets")
            .post(body.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()

        executeRequest(request, callback) { parseMultisig(it) }
    }

    /**
     * Sign transaction
     * POST /api/v1/master-wallet/:id/multisig/transactions/:tid/sign
     */
    fun signMultisig(walletId: String, transactionId: String, signer: String, signature: String, callback: ApiCallback<MultisigSignature>) {
        val body = JSONObject().apply {
            put("signer", signer)
            put("signature", signature)
        }

        val request = Request.Builder()
            .url("$baseUrl/api/v1/master-wallet/$walletId/multisig/transactions/$transactionId/sign")
            .post(body.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()

        executeRequest(request, callback) { json ->
            MultisigSignature(
                walletId = json.optString("wallet_id", walletId),
                transactionId = json.optString("transaction_id", transactionId),
                signer = json.optString("signer", signer),
                signature = json.optString("signature", signature)
            )
        }
    }

    /**
     * Execute multisig
     * POST /api/v1/master-wallet/:id/multisig/transactions/:tid/execute
     */
    fun executeMultisig(walletId: String, transactionId: String, callback: ApiCallback<MasterTransaction>) {
        val body = JSONObject().put("transaction_id", transactionId)

        val request = Request.Builder()
            .url("$baseUrl/api/v1/master-wallet/$walletId/multisig/transactions/$transactionId/execute")
            .post(body.toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()

        executeRequest(request, callback) { parseMasterTransaction(it) }
    }

    // ==================== WHITELABEL ====================

    /**
     * Create whitelabel. The canonical contract has no whitelabel routes;
     * calling this fails closed rather than POSTing to a non-canonical endpoint.
     */
    fun createWhitelabel(name: String, domain: String, branding: String?, feePercent: Double, callback: ApiCallback<Whitelabel>) {
        callback.onError("createWhitelabel has no canonical backend route")
    }

    /**
     * List whitelabels. The canonical contract has no whitelabel routes;
     * calling this fails closed rather than GETting a non-canonical endpoint.
     */
    fun listWhitelabels(callback: ApiCallback<ListResponse<Whitelabel>>) {
        callback.onError("listWhitelabels has no canonical backend route")
    }

    // ==================== ANALYTICS ====================

    /**
     * Get analytics. The canonical contract exposes only per-wallet analytics
     * (GET /api/v1/master-wallet/:id/analytics/{volume,transactions,wallets});
     * there is no global analytics route. Calling this fails closed rather than
     * GETting a non-canonical endpoint.
     */
    fun getAnalytics(period: String = "30d", callback: ApiCallback<MasterAnalytics>) {
        callback.onError(
            "getAnalytics has no canonical backend route; " +
            "use per-wallet analytics (MasterWalletViewModel.loadVolumeStats)"
        )
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
                        val arr = json.optJSONArray("data")
                            ?: json.optJSONArray("wallets")
                            ?: json.optJSONArray("transactions")
                            ?: json.optJSONArray("items")
                            ?: JSONArray()
                        val data = parser(arr)
                        callback.onSuccess(ListResponse(
                            data = data,
                            total = json.optInt("total", data.size),
                            page = json.optInt("page", 1),
                            pageSize = json.optInt("page_size", data.size)
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
    private fun parseWallet(json: JSONObject) = Wallet(
        id = json.optString("id", ""),
        name = json.optString("name", ""),
        address = json.optString("address", ""),
        chain = json.optString("chain", ""),
        walletType = json.optString("wallet_type", ""),
        isActive = json.optBoolean("is_active", true),
        balance = json.optString("balance", "0"),
        createdAt = json.optString("created_at", "")
    )

    private fun parseWalletList(json: JSONArray): List<Wallet> = (0 until json.length()).map { parseWallet(json.getJSONObject(it)) }

    private fun parseMasterTransaction(json: JSONObject) = MasterTransaction(
        id = json.optString("id", ""),
        txHash = json.optString("tx_hash", ""),
        walletId = json.optString("wallet_id", ""),
        type = json.optString("type", ""),
        to = json.optString("to", ""),
        from = json.optString("from", ""),
        amount = json.optString("amount", "0"),
        token = json.optString("token", ""),
        chain = json.optString("chain", ""),
        fee = json.optString("fee", "0"),
        status = json.optString("status", ""),
        createdAt = json.optString("created_at", "")
    )

    private fun parseMasterTransactionList(json: JSONArray): List<MasterTransaction> = (0 until json.length()).map { parseMasterTransaction(json.getJSONObject(it)) }

    private fun parseMultisig(json: JSONObject): MultisigWallet {
        val signersArray = json.optJSONArray("signers")
        val signers = mutableListOf<String>()
        signersArray?.let {
            for (i in 0 until it.length()) {
                signers.add(it.getString(i))
            }
        }
        return MultisigWallet(
            id = json.optString("id", ""),
            name = json.optString("name", ""),
            address = json.optString("address", ""),
            chain = json.optString("chain", ""),
            signers = signers,
            requiredSigs = json.optInt("required_sigs", 0),
            isActive = json.optBoolean("is_active", true)
        )
    }

    private fun parseWhitelabel(json: JSONObject) = Whitelabel(
        id = json.optString("id", ""),
        name = json.optString("name", ""),
        domain = json.optString("domain", ""),
        branding = json.optString("branding", ""),
        feePercent = json.optDouble("fee_percent", 0.0),
        isActive = json.optBoolean("is_active", true),
        usersCount = json.optInt("users_count", 0)
    )

    private fun parseWhitelabelList(json: JSONArray): List<Whitelabel> = (0 until json.length()).map { parseWhitelabel(json.getJSONObject(it)) }
}

// API Callback
interface ApiCallback<T> {
    fun onSuccess(data: T)
    fun onError(error: String)
}

// Data classes
data class Wallet(val id: String, val name: String, val address: String, val chain: String, val walletType: String, val isActive: Boolean, val balance: String, val createdAt: String)
data class Balance(val token: String, val balance: String, val available: String, val locked: String, val usdValue: String)
data class BalanceResponse(val walletId: String, val balances: List<Balance>)
data class MasterTransaction(val id: String, val txHash: String, val walletId: String, val type: String, val to: String, val from: String, val amount: String, val token: String, val chain: String, val fee: String, val status: String, val createdAt: String)
data class GasPrice(val chain: String, val slow: String, val standard: String, val fast: String, val unit: String)
data class MultisigWallet(val id: String, val name: String, val address: String, val chain: String, val signers: List<String>, val requiredSigs: Int, val isActive: Boolean)
data class MultisigSignature(val walletId: String, val transactionId: String, val signer: String, val signature: String)
data class Whitelabel(val id: String, val name: String, val domain: String, val branding: String, val feePercent: Double, val isActive: Boolean, val usersCount: Int)
data class MasterAnalytics(val period: String, val totalVolume: String, val incomingVolume: String, val outgoingVolume: String, val totalTransactions: Int, val successRate: Double)
data class ListResponse<T>(val data: List<T>, val total: Int, val page: Int, val pageSize: Int)
