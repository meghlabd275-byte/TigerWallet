package com.tigeradminpanel.api

import okhttp3.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.io.IOException
import java.util.concurrent.TimeUnit

/**
 * TigerWallet Admin API Service - Complete Android Implementation
 * Handles all API communications with the admin backend
 */
class AdminApiService(private val baseUrl: String, private var authToken: String? = null) {
    
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
    
    // Headers
    private fun getHeaders(): Headers {
        return Headers.Builder()
            .add("Content-Type", "application/json")
            .add("Accept", "application/json")
            .apply {
                authToken?.let { add("Authorization", "Bearer $it") }
            }
            .build()
    }
    
    // ==================== AUTH ====================
    
    /**
     * Admin login
     * POST /api/v1/admin/login
     */
    fun login(email: String, password: String, callback: ApiCallback<LoginResponse>) {
        val body = JSONObject().apply {
            put("email", email)
            put("password", password)
        }.toString()
        
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/login")
            .post(body.toRequestBody(jsonMediaType))
            .build()
        
        client.newCall(request).enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                callback.onError(e.message ?: "Network error")
            }
            override fun onResponse(call: Call, response: Response) {
                try {
                    val json = JSONObject(response.body?.string() ?: "{}")
                    if (response.isSuccessful) {
                        authToken = json.optString("token")
                        callback.onSuccess(LoginResponse(
                            token = json.optString("token"),
                            refreshToken = json.optString("refresh_token"),
                            admin = parseAdmin(json.optJSONObject("admin"))
                        ))
                    } else {
                        callback.onError(json.optString("error", "Login failed"))
                    }
                } catch (e: Exception) {
                    callback.onError(e.message ?: "Parse error")
                }
            }
        })
    }
    
    /**
     * Admin logout
     * POST /api/v1/admin/logout
     */
    fun logout(callback: ApiCallback<Unit>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/logout")
            .post("{}".toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        client.newCall(request).enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                callback.onError(e.message ?: "Network error")
            }
            override fun onResponse(call: Call, response: Response) {
                authToken = null
                callback.onSuccess(Unit)
            }
        })
    }
    
    // ==================== USERS ====================
    
    /**
     * List users with pagination and filters
     * GET /api/v1/admin/users
     */
    fun getUsers(page: Int = 1, pageSize: Int = 20, search: String? = null, 
                 status: String? = null, kycStatus: String? = null, callback: ApiCallback<ListResponse<User>>) {
        val params = mutableListOf<String>()
        params.add("page=$page")
        params.add("page_size=$pageSize")
        search?.let { params.add("search=$it") }
        status?.let { params.add("status=$it") }
        kycStatus?.let { params.add("kyc_status=$it") }
        
        val url = "$baseUrl/api/v1/admin/users?${params.joinToString("&")}"
        
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
                        val users = parseUserList(json.optJSONArray("data") ?: JSONArray())
                        callback.onSuccess(ListResponse(
                            data = users,
                            total = json.optInt("total", 0),
                            page = json.optInt("page", 1),
                            pageSize = json.optInt("page_size", pageSize)
                        ))
                    } else {
                        callback.onError(json.optString("error", "Failed to fetch users"))
                    }
                } catch (e: Exception) {
                    callback.onError(e.message ?: "Parse error")
                }
            }
        })
    }
    
    /**
     * Get user details
     * GET /api/v1/admin/users/:id
     */
    fun getUser(userId: String, callback: ApiCallback<User>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/users/$userId")
            .get()
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseUser(it) }
    }
    
    /**
     * Update user
     * PUT /api/v1/admin/users/:id
     */
    fun updateUser(userId: String, updates: JSONObject, callback: ApiCallback<User>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/users/$userId")
            .put(updates.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseUser(it) }
    }
    
    /**
     * Ban user
     * PUT /api/v1/admin/users/:id/ban
     */
    fun banUser(userId: String, reason: String, callback: ApiCallback<User>) {
        val body = JSONObject().put("reason", reason)
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/users/$userId/ban")
            .put(body.toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseUser(it) }
    }
    
    /**
     * Unban user
     * PUT /api/v1/admin/users/:id/unban
     */
    fun unbanUser(userId: String, callback: ApiCallback<User>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/users/$userId/unban")
            .put("{}".toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseUser(it) }
    }
    
    // ==================== KYC ====================
    
    /**
     * List KYC requests
     * GET /api/v1/admin/kyc
     */
    fun getKYCRequests(page: Int = 1, status: String? = null, callback: ApiCallback<ListResponse<KYCRequest>>) {
        val params = mutableListOf<String>()
        params.add("page=$page")
        status?.let { params.add("status=$it") }
        
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/kyc?${params.joinToString("&")}")
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
                    val kycList = parseKYCList(json.optJSONArray("data") ?: JSONArray())
                    callback.onSuccess(ListResponse(
                        data = kycList,
                        total = json.optInt("total", 0),
                        page = json.optInt("page", 1),
                        pageSize = json.optInt("page_size", 20)
                    ))
                } catch (e: Exception) {
                    callback.onError(e.message ?: "Parse error")
                }
            }
        })
    }
    
    /**
     * Approve KYC
     * PUT /api/v1/admin/kyc/:id/approve
     */
    fun approveKYC(kycId: String, callback: ApiCallback<KYCRequest>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/kyc/$kycId/approve")
            .put("{}".toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseKYC(it) }
    }
    
    /**
     * Reject KYC
     * PUT /api/v1/admin/kyc/:id/reject
     */
    fun rejectKYC(kycId: String, reason: String, callback: ApiCallback<KYCRequest>) {
        val body = JSONObject().put("reason", reason)
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/kyc/$kycId/reject")
            .put(body.toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseKYC(it) }
    }
    
    // ==================== TRANSACTIONS ====================
    
    /**
     * List transactions
     * GET /api/v1/admin/transactions
     */
    fun getTransactions(page: Int = 1, status: String? = null, 
                       token: String? = null, chain: String? = null, callback: ApiCallback<ListResponse<Transaction>>) {
        val params = mutableListOf<String>()
        params.add("page=$page")
        status?.let { params.add("status=$it") }
        token?.let { params.add("token=$it") }
        chain?.let { params.add("chain=$it") }
        
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/transactions?${params.joinToString("&")}")
            .get()
            .headers(getHeaders())
            .build()
        
        executeListRequest(request, callback) { parseTransactionList(it) }
    }
    
    /**
     * Get transaction details
     * GET /api/v1/admin/transactions/:id
     */
    fun getTransaction(txId: String, callback: ApiCallback<Transaction>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/transactions/$txId")
            .get()
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseTransaction(it) }
    }
    
    // ==================== WITHDRAWALS ====================
    
    /**
     * List withdrawals
     * GET /api/v1/admin/withdrawals
     */
    fun getWithdrawals(page: Int = 1, status: String? = null, callback: ApiCallback<ListResponse<Withdrawal>>) {
        val params = mutableListOf<String>()
        params.add("page=$page")
        status?.let { params.add("status=$it") }
        
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/withdrawals?${params.joinToString("&")}")
            .get()
            .headers(getHeaders())
            .build()
        
        executeListRequest(request, callback) { parseWithdrawalList(it) }
    }
    
    /**
     * Approve withdrawal
     * POST /api/v1/admin/withdrawals/:id/approve
     */
    fun approveWithdrawal(withdrawalId: String, callback: ApiCallback<Withdrawal>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/withdrawals/$withdrawalId/approve")
            .post("{}".toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseWithdrawal(it) }
    }
    
    /**
     * Reject withdrawal
     * POST /api/v1/admin/withdrawals/:id/reject
     */
    fun rejectWithdrawal(withdrawalId: String, reason: String, callback: ApiCallback<Withdrawal>) {
        val body = JSONObject().put("reason", reason)
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/withdrawals/$withdrawalId/reject")
            .post(body.toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseWithdrawal(it) }
    }
    
    // ==================== TOKENS ====================
    
    /**
     * List tokens
     * GET /api/v1/admin/tokens
     */
    fun getTokens(callback: ApiCallback<ListResponse<Token>>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/tokens")
            .get()
            .headers(getHeaders())
            .build()
        
        executeListRequest(request, callback) { parseTokenList(it) }
    }
    
    /**
     * Create token
     * POST /api/v1/admin/tokens
     */
    fun createToken(tokenData: JSONObject, callback: ApiCallback<Token>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/tokens")
            .post(tokenData.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseToken(it) }
    }
    
    /**
     * Update token
     * PUT /api/v1/admin/tokens/:id
     */
    fun updateToken(tokenId: String, tokenData: JSONObject, callback: ApiCallback<Token>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/tokens/$tokenId")
            .put(tokenData.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseToken(it) }
    }
    
    /**
     * Activate token
     * PUT /api/v1/admin/tokens/:id/activate
     */
    fun activateToken(tokenId: String, callback: ApiCallback<Token>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/tokens/$tokenId/activate")
            .put("{}".toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseToken(it) }
    }
    
    /**
     * Deactivate token
     * PUT /api/v1/admin/tokens/:id/deactivate
     */
    fun deactivateToken(tokenId: String, callback: ApiCallback<Token>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/tokens/$tokenId/deactivate")
            .put("{}".toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseToken(it) }
    }
    
    // ==================== FEES ====================
    
    /**
     * List fee rules
     * GET /api/v1/admin/fees
     */
    fun getFeeRules(callback: ApiCallback<ListResponse<FeeRule>>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/fees")
            .get()
            .headers(getHeaders())
            .build()
        
        executeListRequest(request, callback) { parseFeeRuleList(it) }
    }
    
    /**
     * Create fee rule
     * POST /api/v1/admin/fees
     */
    fun createFeeRule(feeData: JSONObject, callback: ApiCallback<FeeRule>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/fees")
            .post(feeData.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseFeeRule(it) }
    }
    
    /**
     * Update fee rule
     * PUT /api/v1/admin/fees/:id
     */
    fun updateFeeRule(feeId: String, feeData: JSONObject, callback: ApiCallback<FeeRule>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/fees/$feeId")
            .put(feeData.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseFeeRule(it) }
    }
    
    /**
     * Delete fee rule
     * DELETE /api/v1/admin/fees/:id
     */
    fun deleteFeeRule(feeId: String, callback: ApiCallback<Unit>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/fees/$feeId")
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
                    callback.onError("Failed to delete fee rule")
                }
            }
        })
    }
    
    // ==================== BOTS ====================
    
    /**
     * List bots
     * GET /api/v1/admin/bots
     */
    fun getBots(callback: ApiCallback<ListResponse<Bot>>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/bots")
            .get()
            .headers(getHeaders())
            .build()
        
        executeListRequest(request, callback) { parseBotList(it) }
    }
    
    /**
     * Create bot
     * POST /api/v1/admin/bots
     */
    fun createBot(botData: JSONObject, callback: ApiCallback<Bot>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/bots")
            .post(botData.toString().toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseBot(it) }
    }
    
    /**
     * Start bot
     * POST /api/v1/admin/bots/:id/start
     */
    fun startBot(botId: String, callback: ApiCallback<Bot>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/bots/$botId/start")
            .post("{}".toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseBot(it) }
    }
    
    /**
     * Stop bot
     * POST /api/v1/admin/bots/:id/stop
     */
    fun stopBot(botId: String, callback: ApiCallback<Bot>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/bots/$botId/stop")
            .post("{}".toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseBot(it) }
    }
    
    // ==================== ANALYTICS ====================
    
    /**
     * Get dashboard stats
     * GET /api/v1/admin/analytics/dashboard
     */
    fun getDashboardStats(callback: ApiCallback<DashboardStats>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/analytics/dashboard")
            .get()
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseDashboardStats(it) }
    }
    
    /**
     * Get volume analytics
     * GET /api/v1/admin/analytics/volume
     */
    fun getVolumeAnalytics(period: String = "7d", callback: ApiCallback<VolumeAnalytics>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/analytics/volume?period=$period")
            .get()
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseVolumeAnalytics(it) }
    }
    
    /**
     * Get revenue analytics
     * GET /api/v1/admin/analytics/revenue
     */
    fun getRevenueAnalytics(period: String = "30d", callback: ApiCallback<RevenueAnalytics>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/analytics/revenue?period=$period")
            .get()
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseRevenueAnalytics(it) }
    }
    
    // ==================== SUPPORT ====================
    
    /**
     * List tickets
     * GET /api/v1/admin/support/tickets
     */
    fun getTickets(status: String? = null, callback: ApiCallback<ListResponse<SupportTicket>>) {
        val url = if (status != null) {
            "$baseUrl/api/v1/admin/support/tickets?status=$status"
        } else {
            "$baseUrl/api/v1/admin/support/tickets"
        }
        
        val request = Request.Builder()
            .url(url)
            .get()
            .headers(getHeaders())
            .build()
        
        executeListRequest(request, callback) { parseTicketList(it) }
    }
    
    /**
     * Add ticket message
     * POST /api/v1/admin/support/tickets/:id/messages
     */
    fun addTicketMessage(ticketId: String, message: String, isInternal: Boolean = false, 
                        callback: ApiCallback<TicketMessage>) {
        val body = JSONObject().apply {
            put("content", message)
            put("is_internal", isInternal)
        }
        
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/support/tickets/$ticketId/messages")
            .post(body.toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseTicketMessage(it) }
    }
    
    /**
     * Close ticket
     * POST /api/v1/admin/support/tickets/:id/close
     */
    fun closeTicket(ticketId: String, callback: ApiCallback<SupportTicket>) {
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/support/tickets/$ticketId/close")
            .post("{}".toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        executeRequest(request, callback) { parseTicket(it) }
    }
    
    // ==================== NOTIFICATIONS ====================
    
    /**
     * Send notification
     * POST /api/v1/admin/notifications
     */
    fun sendNotification(title: String, message: String, userId: Int? = null,
                         sendEmail: Boolean = false, sendPush: Boolean = false, callback: ApiCallback<Unit>) {
        val body = JSONObject().apply {
            put("title", title)
            put("message", message)
            userId?.let { put("user_id", it) }
            put("send_email", sendEmail)
            put("send_push", sendPush)
        }
        
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/notifications")
            .post(body.toRequestBody(jsonMediaType))
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
                    callback.onError("Failed to send notification")
                }
            }
        })
    }
    
    /**
     * Broadcast notification to all users
     * POST /api/v1/admin/notifications/broadcast
     */
    fun broadcastNotification(title: String, message: String, callback: ApiCallback<BroadcastResponse>) {
        val body = JSONObject().apply {
            put("title", title)
            put("message", message)
        }
        
        val request = Request.Builder()
            .url("$baseUrl/api/v1/admin/notifications/broadcast")
            .post(body.toRequestBody(jsonMediaType))
            .headers(getHeaders())
            .build()
        
        client.newCall(request).enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                callback.onError(e.message ?: "Network error")
            }
            override fun onResponse(call: Call, response: Response) {
                try {
                    val json = JSONObject(response.body?.string() ?: "{}")
                    callback.onSuccess(BroadcastResponse(
                        totalUsers = json.optInt("total_users", 0),
                        notified = json.optInt("notified", 0)
                    ))
                } catch (e: Exception) {
                    callback.onError(e.message ?: "Parse error")
                }
            }
        })
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
    
    private fun <T> executeListRequest(request: Request, callback: ApiCallback<ListResponse<T>>, 
                                   parser: (JSONArray) -> List<T>) {
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
    private fun parseAdmin(json: JSONObject?) = json?.let {
        Admin(
            id = it.optInt("id", 0),
            email = it.optString("email", ""),
            username = it.optString("username", ""),
            role = it.optString("role", ""),
            twoFactorEnabled = it.optBoolean("two_factor_enabled", false),
            isActive = it.optBoolean("is_active", true)
        )
    } ?: Admin(0, "", "", "", false, true)
    
    private fun parseUserList(json: JSONArray): List<User> = (0 until json.length()).map { parseUser(json.getJSONObject(it)) }
    private fun parseUser(json: JSONObject) = User(
        id = json.optInt("id", 0),
        email = json.optString("email", ""),
        username = json.optString("username", ""),
        walletAddress = json.optString("wallet_address", ""),
        status = json.optString("status", ""),
        kycStatus = json.optString("kyc_status", ""),
        balance = json.optString("balance", "{}"),
        createdAt = json.optString("created_at", "")
    )
    
    private fun parseKYCList(json: JSONArray): List<KYCRequest> = (0 until json.length()).map { parseKYC(json.getJSONObject(it)) }
    private fun parseKYC(json: JSONObject) = KYCRequest(
        id = json.optInt("id", 0),
        userId = json.optInt("user_id", 0),
        documentType = json.optString("document_type", ""),
        status = json.optString("status", ""),
        submittedAt = json.optString("submitted_at", "")
    )
    
    private fun parseTransactionList(json: JSONArray): List<Transaction> = (0 until json.length()).map { parseTransaction(json.getJSONObject(it)) }
    private fun parseTransaction(json: JSONObject) = Transaction(
        id = json.optInt("id", 0),
        txHash = json.optString("tx_hash", ""),
        userId = json.optInt("user_id", 0),
        type = json.optString("type", ""),
        amount = json.optDouble("amount", 0.0),
        fee = json.optDouble("fee", 0.0),
        token = json.optString("token", ""),
        chain = json.optString("chain", ""),
        status = json.optString("status", ""),
        createdAt = json.optString("created_at", "")
    )
    
    private fun parseWithdrawalList(json: JSONArray): List<Withdrawal> = (0 until json.length()).map { parseWithdrawal(json.getJSONObject(it)) }
    private fun parseWithdrawal(json: JSONObject) = Withdrawal(
        id = json.optInt("id", 0),
        userId = json.optInt("user_id", 0),
        amount = json.optDouble("amount", 0.0),
        token = json.optString("token", ""),
        address = json.optString("address", ""),
        status = json.optString("status", ""),
        createdAt = json.optString("created_at", "")
    )
    
    private fun parseTokenList(json: JSONArray): List<Token> = (0 until json.length()).map { parseToken(json.getJSONObject(it)) }
    private fun parseToken(json: JSONObject) = Token(
        id = json.optInt("id", 0),
        name = json.optString("name", ""),
        symbol = json.optString("symbol", ""),
        decimals = json.optInt("decimals", 18),
        contractAddress = json.optString("contract_address", ""),
        isActive = json.optBoolean("is_active", true),
        createdAt = json.optString("created_at", "")
    )
    
    private fun parseFeeRuleList(json: JSONArray): List<FeeRule> = (0 until json.length()).map { parseFeeRule(json.getJSONObject(it)) }
    private fun parseFeeRule(json: JSONObject) = FeeRule(
        id = json.optInt("id", 0),
        name = json.optString("name", ""),
        feeType = json.optString("fee_type", ""),
        feeValue = json.optDouble("fee_value", 0.0),
        isActive = json.optBoolean("is_active", true)
    )
    
    private fun parseBotList(json: JSONArray): List<Bot> = (0 until json.length()).map { parseBot(json.getJSONObject(it)) }
    private fun parseBot(json: JSONObject) = Bot(
        id = json.optInt("id", 0),
        name = json.optString("name", ""),
        botType = json.optString("bot_type", ""),
        status = json.optString("status", ""),
        totalPnl = json.optDouble("total_pnl", 0.0),
        createdAt = json.optString("created_at", "")
    )
    
    private fun parseTicketList(json: JSONArray): List<SupportTicket> = (0 until json.length()).map { parseTicket(json.getJSONObject(it)) }
    private fun parseTicket(json: JSONObject) = SupportTicket(
        id = json.optInt("id", 0),
        ticketId = json.optString("ticket_id", ""),
        userId = json.optInt("user_id", 0),
        subject = json.optString("subject", ""),
        status = json.optString("status", ""),
        priority = json.optString("priority", ""),
        createdAt = json.optString("created_at", "")
    )
    
    private fun parseTicketMessage(json: JSONObject) = TicketMessage(
        id = json.optInt("id", 0),
        message = json.optString("message", ""),
        senderName = json.optString("sender_name", ""),
        createdAt = json.optString("created_at", "")
    )
    
    private fun parseDashboardStats(json: JSONObject) = DashboardStats(
        totalUsers = json.optInt("total_users", 0),
        activeUsers = json.optInt("active_users", 0),
        totalVolume = json.optDouble("total_volume", 0.0),
        todayVolume = json.optDouble("today_volume", 0.0),
        totalTransactions = json.optInt("total_transactions", 0),
        todayTransactions = json.optInt("today_transactions", 0),
        pendingWithdrawals = json.optInt("pending_withdrawals", 0),
        pendingKYC = json.optInt("pending_kyc", 0)
    )
    
    private fun parseVolumeAnalytics(json: JSONObject): VolumeAnalytics {
        val dailyVolumes = mutableListOf<DailyVolume>()
        val dailyArray = json.optJSONArray("daily_volumes")
        dailyArray?.let {
            for (i in 0 until it.length()) {
                val item = it.getJSONObject(i)
                dailyVolumes.add(DailyVolume(
                    date = item.optString("date", ""),
                    volume = item.optDouble("volume", 0.0),
                    count = item.optInt("count", 0)
                ))
            }
        }
        return VolumeAnalytics(dailyVolumes)
    }
    
    private fun parseRevenueAnalytics(json: JSONObject): RevenueAnalytics {
        return RevenueAnalytics(
            totalRevenue = json.optDouble("total_revenue", 0.0),
            period = json.optString("period", "")
        )
    }
}

// API Callback interface
interface ApiCallback<T> {
    fun onSuccess(data: T)
    fun onError(error: String)
}

// Data classes
data class LoginResponse(val token: String, val refreshToken: String?, val admin: Admin?)
data class Admin(val id: Int, val email: String, val username: String, val role: String, 
                val twoFactorEnabled: Boolean, val isActive: Boolean)
data class User(val id: Int, val email: String, val username: String, val walletAddress: String,
               val status: String, val kycStatus: String, val balance: String, val createdAt: String)
data class KYCRequest(val id: Int, val userId: Int, val documentType: String, val status: String, val submittedAt: String)
data class Transaction(val id: Int, val txHash: String, val userId: Int, val type: String, 
                      val amount: Double, val fee: Double, val token: String, val chain: String, 
                      val status: String, val createdAt: String)
data class Withdrawal(val id: Int, val userId: Int, val amount: Double, val token: String, 
                     val address: String, val status: String, val createdAt: String)
data class Token(val id: Int, val name: String, val symbol: String, val decimals: Int,
                val contractAddress: String, val isActive: Boolean, val createdAt: String)
data class FeeRule(val id: Int, val name: String, val feeType: String, 
                  val feeValue: Double, val isActive: Boolean)
data class Bot(val id: Int, val name: String, val botType: String, val status: String,
              val totalPnl: Double, val createdAt: String)
data class SupportTicket(val id: Int, val ticketId: String, val userId: Int, val subject: String,
                        val status: String, val priority: String, val createdAt: String)
data class TicketMessage(val id: Int, val message: String, val senderName: String, val createdAt: String)
data class DashboardStats(val totalUsers: Int, val activeUsers: Int, val totalVolume: Double,
                         val todayVolume: Double, val totalTransactions: Int, val todayTransactions: Int,
                         val pendingWithdrawals: Int, val pendingKYC: Int)
data class VolumeAnalytics(val dailyVolumes: List<DailyVolume>)
data class DailyVolume(val date: String, val volume: Double, val count: Int)
data class RevenueAnalytics(val totalRevenue: Double, val period: String)
data class BroadcastResponse(val totalUsers: Int, val notified: Int)
data class ListResponse<T>(val data: List<T>, val total: Int, val page: Int, val pageSize: Int)
