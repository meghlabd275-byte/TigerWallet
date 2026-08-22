package com.tigerparty.api

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
 * ProjectParty Android API client.
 *
 * Targets the standalone ProjectParty backend (project_party/go/cmd/main.go,
 * port 8106, path prefix /api/v1, JWT auth + RBAC). The token is persisted in
 * an EncryptedSharedPreferences file ("tigerparty_prefs"). Base URL is
 * configurable via the `PROJECT_PARTY_API_BASE_URL` system property and
 * defaults to the local dev endpoint.
 *
 * Every method issues a real HttpURLConnection against the backend — no stubs,
 * fakes, or mock data. On any non-2xx response the method throws
 * [ProjectPartyApiException] (fail-closed); it never returns fabricated data.
 *
 * Method set (matches project_party/web/src/services/api.ts + the discovery,
 * pricing, analytics, compliance routes the task requires):
 *   auth: register, login
 *   discovery: coins, search, featured, trending, market
 *   tokens: listTokens, getToken, createToken, updateToken, deleteToken,
 *         submitToken, approveToken, rejectToken
 *   listings: listListings, getListing, createListing, updateListingStatus,
 *         featureListing
 *   launchpad: listLaunchpads, getLaunchpad, createLaunchpad, contribute,
 *         claimTokens, cancelLaunchpad
 *   market-making: getMakerOrders, getMarketMakerStatus, createMakerOrders,
 *         updateOrderStatus, addLiquidity, removeLiquidity
 *   pricing: getTokenPrice, getPriceHistory, setTokenPrice, updatePrice
 *   analytics: getTradingVolume, getLiquidity, getHolderCount,
 *         getTransactionCount
 *   compliance: getAuditStatus, getKYCStatus, requestAudit, submitKYC
 *   fees: getListingFees, calculateFees, payFees
 *   favorites: listFavorites, addFavorite, removeFavorite
 *   health: getHealth
 *
 * NOTE: route paths and auth payload fields are aligned to the standalone
 * backend (username/password; /launchpad/create; /market-making/orders;
 * /launchpad/:id/contribute), NOT the WL :8464 web client, because these
 * clients target :8106.
 */
class ProjectPartyApiService private constructor(
    private val context: Context,
    private val baseUrl: String,
) {
    private val prefs: SharedPreferences by lazy {
        val masterKey = MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
        EncryptedSharedPreferences.create(
            context,
            "tigerparty_prefs",
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
        absolutePath: Boolean = false,
    ): String {
        val urlStr = if (absolutePath) baseUrl.trimEnd('/') + path else baseUrl + path
        val conn = (URL(urlStr).openConnection() as HttpURLConnection).apply {
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
                throw ProjectPartyApiException(code, msg)
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

    private fun get(path: String, authenticated: Boolean = true, absolute: Boolean = false): String =
        http(path, "GET", null, authenticated, absolute)

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

    private fun enc(s: String): String =
        URLEncoder.encode(s, StandardCharsets.UTF_8.name())

    // ---------------------------------------------------------------------------
    // Health (lives at /health, outside /api/v1)
    // ---------------------------------------------------------------------------

    suspend fun getHealth(): JSONObject = withContext(Dispatchers.IO) {
        obj(get("/health", authenticated = false, absolute = true))
    }

    // ---------------------------------------------------------------------------
    // Auth
    // ---------------------------------------------------------------------------

    suspend fun register(username: String, password: String): PartyAuthResponse =
        withContext(Dispatchers.IO) {
            val body = JSONObject().apply {
                put("username", username)
                put("password", password)
            }
            val res = obj(post("/auth/register", body.toString(), authenticated = false))
            val tok = res.optString("token").takeIf { it.isNotEmpty() }
            if (tok != null) setToken(tok)
            PartyAuthResponse(
                token = tok,
                username = res.optString("username"),
                role = res.optString("role"),
            )
        }

    suspend fun login(username: String, password: String): PartyAuthResponse =
        withContext(Dispatchers.IO) {
            val body = JSONObject().apply {
                put("username", username)
                put("password", password)
            }
            val res = obj(post("/auth/login", body.toString(), authenticated = false))
            val tok = res.optString("token").takeIf { it.isNotEmpty() }
            if (tok != null) setToken(tok)
            PartyAuthResponse(
                token = tok,
                username = res.optString("username"),
                role = res.optString("role"),
            )
        }

    // ---------------------------------------------------------------------------
    // Discovery (public)
    // ---------------------------------------------------------------------------

    suspend fun getCoins(): JSONArray = withContext(Dispatchers.IO) {
        arr(get("/coins", authenticated = false))
    }

    suspend fun searchTokens(q: String): JSONObject = withContext(Dispatchers.IO) {
        obj(get("/search?q=${enc(q)}", authenticated = false))
    }

    suspend fun getFeatured(): JSONObject = withContext(Dispatchers.IO) {
        obj(get("/featured", authenticated = false))
    }

    suspend fun getTrending(): JSONObject = withContext(Dispatchers.IO) {
        obj(get("/trending", authenticated = false))
    }

    suspend fun getMarket(): JSONObject = withContext(Dispatchers.IO) {
        obj(get("/market", authenticated = false))
    }

    // ---------------------------------------------------------------------------
    // Tokens
    // ---------------------------------------------------------------------------

    suspend fun listTokens(status: String? = null): JSONObject = withContext(Dispatchers.IO) {
        val path = if (status.isNullOrEmpty()) "/tokens" else "/tokens?status=${enc(status)}"
        obj(get(path))
    }

    suspend fun getToken(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/tokens/${enc(id)}")) }

    suspend fun createToken(token: JSONObject): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/tokens", token.toString())) }

    suspend fun updateToken(id: String, token: JSONObject): JSONObject =
        withContext(Dispatchers.IO) { obj(put("/tokens/${enc(id)}", token.toString())) }

    suspend fun deleteToken(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(delete("/tokens/${enc(id)}")) }

    suspend fun submitToken(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/tokens/${enc(id)}/submit", null)) }

    suspend fun approveToken(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/tokens/${enc(id)}/approve", null)) }

    suspend fun rejectToken(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/tokens/${enc(id)}/reject", null)) }

    // ---------------------------------------------------------------------------
    // Listings
    // ---------------------------------------------------------------------------

    suspend fun listListings(status: String? = null): JSONObject =
        withContext(Dispatchers.IO) {
            val path = if (status.isNullOrEmpty()) "/listings" else "/listings?status=${enc(status)}"
            obj(get(path))
        }

    suspend fun getListing(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/listings/${enc(id)}")) }

    suspend fun createListing(listing: JSONObject): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/listings", listing.toString())) }

    suspend fun updateListingStatus(id: String, status: String): JSONObject =
        withContext(Dispatchers.IO) {
            val body = JSONObject().put("status", status)
            obj(put("/listings/${enc(id)}/status", body.toString()))
        }

    suspend fun featureListing(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/listings/${enc(id)}/featured", null)) }

    // ---------------------------------------------------------------------------
    // Launchpad
    // ---------------------------------------------------------------------------

    suspend fun listLaunchpads(status: String? = null): JSONObject =
        withContext(Dispatchers.IO) {
            val path = if (status.isNullOrEmpty()) "/launchpad" else "/launchpad?status=${enc(status)}"
            obj(get(path))
        }

    suspend fun getLaunchpad(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/launchpad/${enc(id)}")) }

    suspend fun createLaunchpad(launchpad: JSONObject): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/launchpad/create", launchpad.toString())) }

    suspend fun contribute(id: String, amount: String): JSONObject =
        withContext(Dispatchers.IO) {
            val body = JSONObject().put("amount", amount)
            obj(post("/launchpad/${enc(id)}/contribute", body.toString()))
        }

    suspend fun claimTokens(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/launchpad/${enc(id)}/claim", null)) }

    suspend fun cancelLaunchpad(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/launchpad/${enc(id)}/cancel", null)) }

    // ---------------------------------------------------------------------------
    // Market-making
    // ---------------------------------------------------------------------------

    suspend fun getMakerOrders(tokenId: String? = null): JSONObject =
        withContext(Dispatchers.IO) {
            val path = if (tokenId.isNullOrEmpty()) "/market-making/orders"
                else "/market-making/orders?token_id=${enc(tokenId)}"
            obj(get(path))
        }

    suspend fun getMarketMakerStatus(tokenId: String): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/market-making/status/${enc(tokenId)}")) }

    suspend fun createMakerOrders(orders: JSONObject): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/market-making/orders", orders.toString())) }

    suspend fun updateOrderStatus(id: String, status: String): JSONObject =
        withContext(Dispatchers.IO) {
            val body = JSONObject().put("status", status)
            obj(put("/market-making/orders/${enc(id)}/status", body.toString()))
        }

    suspend fun addLiquidity(liquidity: JSONObject): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/market-making/liquidity/add", liquidity.toString())) }

    suspend fun removeLiquidity(liquidity: JSONObject): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/market-making/liquidity/remove", liquidity.toString())) }

    // ---------------------------------------------------------------------------
    // Pricing
    // ---------------------------------------------------------------------------

    suspend fun getTokenPrice(tokenId: String): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/pricing/${enc(tokenId)}")) }

    suspend fun getPriceHistory(tokenId: String): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/pricing/history/${enc(tokenId)}")) }

    suspend fun setTokenPrice(tokenId: String, price: String): JSONObject =
        withContext(Dispatchers.IO) {
            val body = JSONObject().apply {
                put("token_id", tokenId)
                put("price", price)
            }
            obj(post("/pricing/set", body.toString()))
        }

    suspend fun updatePrice(tokenId: String, price: String): JSONObject =
        withContext(Dispatchers.IO) {
            val body = JSONObject().apply {
                put("token_id", tokenId)
                put("price", price)
            }
            obj(post("/pricing/update", body.toString()))
        }

    // ---------------------------------------------------------------------------
    // Analytics (public read)
    // ---------------------------------------------------------------------------

    suspend fun getTradingVolume(): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/analytics/volume", authenticated = false)) }

    suspend fun getLiquidity(): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/analytics/liquidity", authenticated = false)) }

    suspend fun getHolderCount(): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/analytics/holders", authenticated = false)) }

    suspend fun getTransactionCount(): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/analytics/transactions", authenticated = false)) }

    // ---------------------------------------------------------------------------
    // Compliance
    // ---------------------------------------------------------------------------

    suspend fun getAuditStatus(tokenId: String): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/compliance/audit/${enc(tokenId)}")) }

    suspend fun getKYCStatus(tokenId: String): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/compliance/kyc/${enc(tokenId)}")) }

    suspend fun requestAudit(audit: JSONObject): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/compliance/audit", audit.toString())) }

    suspend fun submitKYC(kyc: JSONObject): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/compliance/kyc/submit", kyc.toString())) }

    // ---------------------------------------------------------------------------
    // Fees
    // ---------------------------------------------------------------------------

    suspend fun getListingFees(): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/fees", authenticated = false)) }

    suspend fun calculateFees(fee: JSONObject): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/fees/calculate", fee.toString())) }

    suspend fun payFees(payment: JSONObject): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/fees/pay", payment.toString())) }

    // ---------------------------------------------------------------------------
    // Favorites (auth)
    // ---------------------------------------------------------------------------

    suspend fun listFavorites(): JSONObject =
        withContext(Dispatchers.IO) { obj(get("/favorites")) }

    suspend fun addFavorite(favorite: JSONObject): JSONObject =
        withContext(Dispatchers.IO) { obj(post("/favorites", favorite.toString())) }

    suspend fun removeFavorite(id: String): JSONObject =
        withContext(Dispatchers.IO) { obj(delete("/favorites/${enc(id)}")) }

    companion object {
        private const val KEY_TOKEN = "party_jwt"

        @Volatile private var instance: ProjectPartyApiService? = null

        fun getInstance(context: Context): ProjectPartyApiService {
            return instance ?: synchronized(this) {
                instance ?: ProjectPartyApiService(
                    context = context.applicationContext,
                    baseUrl = System.getProperty("PROJECT_PARTY_API_BASE_URL")
                        ?: DEFAULT_BASE_URL,
                ).also { instance = it }
            }
        }

        const val DEFAULT_BASE_URL = "http://localhost:8106/api/v1"
    }
}

class ProjectPartyApiException(val status: Int, val detail: String) :
    RuntimeException("ProjectParty API $status: $detail")

data class PartyAuthResponse(
    val token: String?,
    val username: String,
    val role: String,
)
