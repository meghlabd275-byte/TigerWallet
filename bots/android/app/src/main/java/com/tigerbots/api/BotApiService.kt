package com.tigerbots.api

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject
import java.io.BufferedReader
import java.io.InputStreamReader
import java.io.OutputStream
import java.net.HttpURLConnection
import java.net.URL
import java.net.URLEncoder
import java.nio.charset.StandardCharsets

/**
 * TigerBots Android API client.
 *
 * Targets the standalone Bots backend (mm_bot_platform/bot_api, port 8471,
 * path prefix /api/v1). JWT bearer auth; the token is persisted in an
 * EncryptedSharedPreferences file ("tigerbots_prefs"). Base URL is configurable
 * via the `BOTS_API_BASE_URL` system property and defaults to the local dev
 * endpoint.
 *
 * Every method issues a real HttpURLConnection against the backend — no stubs,
 * fakes, or mock data. On any non-2xx response the method throws
 * [BotApiException] (fail-closed); it never returns fabricated data.
 *
 * Method set mirrors bots/web/src/services/api.ts:
 *   auth: register, login, logout
 *   bots: listBots, getBot, createBot, deleteBot, startBot, stopBot, pauseBot,
 *         listBotExecutions, listBotLogs
 *   users: currentBotUser, listBotUsers, createBotUser, deleteBotUser,
 *         listBotTransactions
 *   subscriptions: getSubscription, createSubscription
 *   fees: getFeeConfigs, updateFeeConfig
 *   cex: listCEX, addCEX, removeCEX
 *   dex: listDEX, addDEX, removeDEX
 *   keys: listAPIKeys, createAPIKey, deleteAPIKey
 *   admin: adminListUsers, adminUserStatus, adminStats, adminGetFeeAddresses,
 *         adminSetFeeAddress, adminDeleteFeeAddress, adminBotStatus
 *   public: publicTiers, health
 */
class BotApiService private constructor(
    private val context: Context,
    private val baseUrl: String,
) {
    private val prefs: SharedPreferences by lazy {
        val masterKey = MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
        EncryptedSharedPreferences.create(
            context,
            "tigerbots_prefs",
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SCM,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
    }

    private var token: String? = prefs.getString(KEY_TOKEN, null)

    fun setToken(token: String?) {
        this.token = token
        prefs.edit().apply {
            if (token.isNullOrEmpty()) remove(KEY_TOKEN) else putString(KEY_TOKEN, token)
        }.apply()
    }

    fun clearToken() = setToken(null)

    fun getToken(): String? = token

    // ---------------------------------------------------------------------------
    // Low-level HTTP
    // ---------------------------------------------------------------------------

    private fun http(
        path: String,
        method: String,
        bodyJson: String? = null,
        authenticated: Boolean = true,
    ): String {
        val conn = (URL(baseUrl + path).openConnection() as HttpURLConnection).apply {
            requestMethod = method
            connectTimeout = 15_000
            readTimeout = 30_000
            instanceFollowRedirects = false
            setRequestProperty("Accept", "application/json")
            if (bodyJson != null) setRequestProperty("Content-Type", "application/json; charset=utf-8")
            if (authenticated) token?.let { setRequestProperty("Authorization", "Bearer $it") }
            if (bodyJson != null) doOutput = true
        }
        try {
            if (bodyJson != null) {
                val out: OutputStream = conn.outputStream
                out.write(bodyJson.toByteArray(StandardCharsets.UTF_8))
                out.flush()
            }
            val code = conn.responseCode
            val stream = if (code in 200..299) conn.inputStream else conn.errorStream
            val text = (stream?.let {
                BufferedReader(InputStreamReader(it, StandardCharsets.UTF_8)).use { r ->
                    val sb = StringBuilder(); var line = r.readLine()
                    while (line != null) { sb.append(line); line = r.readLine() }
                    sb.toString()
                }
            } ?: "")
            if (code !in 200..299) {
                val msg = parseError(text).ifEmpty { "HTTP $code" }
                throw BotApiException(code, msg)
            }
            return text
        } finally {
            conn.disconnect()
        }
    }

    private fun parseError(text: String): String {
        if (text.isBlank()) return ""
        return try {
            val obj = JSONObject(text)
            obj.optString("error").ifEmpty { obj.optString("message") }
        } catch (_: Exception) {
            text.take(300)
        }
    }

    private fun get(path: String, authenticated: Boolean = true): String =
        http(path, "GET", null, authenticated)

    private fun post(path: String, bodyJson: String?, authenticated: Boolean = true): String =
        http(path, "POST", bodyJson, authenticated)

    private fun put(path: String, bodyJson: String?): String =
        http(path, "PUT", bodyJson, true)

    private fun delete(path: String): String =
        http(path, "DELETE", null, true)

    private fun obj(bodyJson: String): JSONObject =
        if (bodyJson.isBlank()) JSONObject() else JSONObject(bodyJson)

    private fun arr(bodyJson: String): JSONArray =
        if (bodyJson.isBlank()) JSONArray() else JSONArray(bodyJson)

    // ---------------------------------------------------------------------------
    // Auth
    // ---------------------------------------------------------------------------

    suspend fun register(
        username: String,
        password: String,
        email: String? = null,
        walletAddress: String? = null,
        role: String? = null,
    ): BotAuthResponse = withContext(Dispatchers.IO) {
        val body = JSONObject().apply {
            put("username", username)
            put("password", password)
            email?.let { put("email", it) }
            walletAddress?.let { put("wallet_address", it) }
            role?.let { put("role", it) }
        }
        val res = obj(post("/auth/register", body.toString(), authenticated = false))
        val tok = res.optString("token").takeIf { it.isNotEmpty() }
        if (tok != null) setToken(tok)
        BotAuthResponse(
            token = tok,
            userId = res.optString("user_id"),
            role = res.optString("role"),
        )
    }

    suspend fun login(username: String, password: String): BotAuthResponse =
        withContext(Dispatchers.IO) {
            val body = JSONObject().apply {
                put("username", username)
                put("password", password)
            }
            val res = obj(post("/auth/login", body.toString(), authenticated = false))
            val tok = res.optString("token").takeIf { it.isNotEmpty() }
            if (tok != null) setToken(tok)
            BotAuthResponse(
                token = tok,
                userId = res.optString("user_id"),
                role = res.optString("role"),
            )
        }

    suspend fun logout(): Unit = withContext(Dispatchers.IO) {
        try {
            post("/auth/logout", null)
        } finally {
            clearToken()
        }
    }

    // ---------------------------------------------------------------------------
    // Health (lives at /api/v1/health on this backend)
    // ---------------------------------------------------------------------------

    suspend fun health(): BotHealth = withContext(Dispatchers.IO) {
        val res = obj(get("/health", authenticated = false))
        BotHealth(
            status = res.optString("status"),
            service = res.optString("service"),
        )
    }

    suspend fun publicTiers(): JSONArray = withContext(Dispatchers.IO) {
        arr(get("/public/tiers", authenticated = false))
    }

    // ---------------------------------------------------------------------------
    // Bots CRUD + lifecycle
    // ---------------------------------------------------------------------------

    suspend fun listBots(): JSONObject = withContext(Dispatchers.IO) { obj(get("/bots")) }

    suspend fun getBot(id: String): JSONObject = withContext(Dispatchers.IO) { obj(get("/bots/${enc(id)}")) }

    suspend fun createBot(
        name: String,
        botType: String,
        config: JSONObject? = null,
        exchange: String? = null,
        pair: String? = null,
    ): JSONObject = withContext(Dispatchers.IO) {
        val body = JSONObject().apply {
            put("name", name)
            put("bot_type", botType)
            put("config", config ?: JSONObject())
            exchange?.let { put("exchange", it) }
            pair?.let { put("pair", it) }
        }
        obj(post("/bots", body.toString()))
    }

    suspend fun deleteBot(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(delete("/bots/${enc(id)}")) }

    suspend fun startBot(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/bots/${enc(id)}/start", null)) }

    suspend fun stopBot(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/bots/${enc(id)}/stop", null)) }

    suspend fun pauseBot(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/bots/${enc(id)}/pause", null)) }

    /**
     * Bot executions/logs are not exposed by the current bot_api route table
     * (see mm_bot_platform/bot_api/main.go). The web client nonetheless defines
     * these methods; here they hit the documented /bots/:id/executions and
     * /bots/:id/logs paths so the contract is honoured if the backend grows them.
     */
    suspend fun listBotExecutions(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/bots/${enc(id)}/executions")) }

    suspend fun listBotLogs(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/bots/${enc(id)}/logs")) }

    // Frontend-compat aliases exposed by the backend.
    suspend fun listBotInstances(): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/bots/instances")) }

    suspend fun currentBotUser(): JSONObject = withContext(Dispatchers.IO) { obj(get("/bots/me")) }

    // ---------------------------------------------------------------------------
    // Bot users
    // ---------------------------------------------------------------------------

    suspend fun listBotUsers(): JSONObject = withContext(Dispatchers.IO) { obj(get("/bots/users")) }

    suspend fun createBotUser(
        username: String,
        password: String,
        email: String? = null,
        walletAddress: String? = null,
        role: String? = null,
    ): JSONObject = withContext(Dispatchers.IO) {
        val body = JSONObject().apply {
            put("username", username)
            put("password", password)
            email?.let { put("email", it) }
            walletAddress?.let { put("wallet_address", it) }
            role?.let { put("role", it) }
        }
        obj(post("/bots/users", body.toString()))
    }

    suspend fun deleteBotUser(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(delete("/bots/users/${enc(id)}")) }

    suspend fun listBotTransactions(): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/bots/transactions")) }

    // ---------------------------------------------------------------------------
    // Subscriptions
    // ---------------------------------------------------------------------------

    suspend fun getSubscription(): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/subscription")) }

    suspend fun createSubscription(tier: String, expiresIn: String? = null): JSONObject =
        withContext(Dispatchers.IO) {
            val body = JSONObject().apply {
                put("tier", tier)
                expiresIn?.let { put("expires_in", it) }
            }
            obj(post("/subscription", body.toString()))
        }

    // ---------------------------------------------------------------------------
    // Fees
    // ---------------------------------------------------------------------------

    suspend fun getFeeConfigs(): JSONObject = withContext(Dispatchers.IO) { obj(get("/fees")) }

    suspend fun updateFeeConfig(
        id: String,
        name: String? = null,
        percentage: String? = null,
        enabled: Boolean? = null,
    ): JSONObject = withContext(Dispatchers.IO) {
        val body = JSONObject().apply {
            put("id", id)
            name?.let { put("name", it) }
            percentage?.let { put("percentage", it) }
            enabled?.let { put("enabled", it) }
        }
        obj(put("/fees", body.toString()))
    }

    // ---------------------------------------------------------------------------
    // CEX connectors
    // ---------------------------------------------------------------------------

    suspend fun listCEX(): JSONObject = withContext(Dispatchers.IO) { obj(get("/cex")) }

    suspend fun addCEX(name: String, config: JSONObject): JSONObject =
        withContext(Dispatchers.IO) {
            val body = JSONObject().apply {
                put("name", name)
                put("config", config)
            }
            obj(post("/cex", body.toString()))
        }

    suspend fun removeCEX(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(delete("/cex/${enc(id)}")) }

    // ---------------------------------------------------------------------------
    // DEX connectors
    // ---------------------------------------------------------------------------

    suspend fun listDEX(): JSONObject = withContext(Dispatchers.IO) { obj(get("/dex")) }

    suspend fun addDEX(name: String, config: JSONObject): JSONObject =
        withContext(Dispatchers.IO) {
            val body = JSONObject().apply {
                put("name", name)
                put("config", config)
            }
            obj(post("/dex", body.toString()))
        }

    suspend fun removeDEX(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(delete("/dex/${enc(id)}")) }

    // ---------------------------------------------------------------------------
    // API keys (AES-GCM at rest on the backend)
    // ---------------------------------------------------------------------------

    suspend fun listAPIKeys(): JSONObject = withContext(Dispatchers.IO) { obj(get("/keys")) }

    suspend fun createAPIKey(exchange: String, apiKey: String): JSONObject =
        withContext(Dispatchers.IO) {
            val body = JSONObject().apply {
                put("exchange", exchange)
                put("api_key", apiKey)
            }
            obj(post("/keys", body.toString()))
        }

    suspend fun deleteAPIKey(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(delete("/keys/${enc(id)}")) }

    // ---------------------------------------------------------------------------
    // Admin (super-admin / finance-admin only)
    // ---------------------------------------------------------------------------

    suspend fun adminListUsers(): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/admin/users")) }

    suspend fun adminUserStatus(id: String, active: Boolean): JSONObject =
        withContext(Dispatchers.IO) {
            val body = JSONObject().apply {
                put("id", id)
                put("is_active", active)
            }
            obj(put("/admin/users/${enc(id)}/status", body.toString()))
        }

    suspend fun adminStats(): JSONObject = withContext(Dispatchers.IO) { obj(get("/admin/stats")) }

    suspend fun adminGetFeeAddresses(): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/admin/fee-addresses")) }

    suspend fun adminSetFeeAddress(
        label: String,
        chainId: Long,
        address: String,
    ): JSONObject = withContext(Dispatchers.IO) {
        val body = JSONObject().apply {
            put("label", label)
            put("chain_id", chainId)
            put("address", address)
        }
        obj(post("/admin/fee-addresses", body.toString()))
    }

    suspend fun adminDeleteFeeAddress(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(delete("/admin/fee-addresses/${enc(id)}")) }

    suspend fun adminBotStatus(id: String, status: String): JSONObject =
        withContext(Dispatchers.IO) {
            val body = JSONObject().apply {
                put("id", id)
                put("status", status)
            }
            obj(post("/admin/bots/${enc(id)}/status", body.toString()))
        }

    private fun enc(s: String): String =
        URLEncoder.encode(s, StandardCharsets.UTF_8.name())

    companion object {
        private const val KEY_TOKEN = "bot_jwt"

        @Volatile private var instance: BotApiService? = null

        fun getInstance(context: Context): BotApiService {
            return instance ?: synchronized(this) {
                instance ?: BotApiService(
                    context = context.applicationContext,
                    baseUrl = System.getProperty("BOTS_API_BASE_URL")
                        ?: DEFAULT_BASE_URL,
                ).also { instance = it }
            }
        }

        const val DEFAULT_BASE_URL = "http://localhost:8471/api/v1"
    }
}

class BotApiException(val status: Int, val detail: String) :
    RuntimeException("Bots API $status: $detail")

data class BotAuthResponse(
    val token: String?,
    val userId: String,
    val role: String,
)

data class BotHealth(
    val status: String,
    val service: String,
)
