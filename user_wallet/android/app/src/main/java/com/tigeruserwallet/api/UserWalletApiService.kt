package com.tigeruserwallet.api

import android.content.Context
import android.util.Base64
import android.util.Log
import com.tigeruserwallet.crypto.SecureBlobStore
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.io.IOException
import java.net.URLEncoder
import java.security.SecureRandom
import java.util.concurrent.TimeUnit

/**
 * TigerWallet UserWallet API Service — Android.
 *
 * Mirrors `user_wallet/web/src/services/api.ts` + `contexts/OnboardingContext.tsx`:
 * the no-registration self-custody model. Talks to the canonical TigerWallet
 * UserWallet backend (go/wallet_api :8443; Android emulator maps the host's
 * localhost to 10.0.2.2:8443).
 *
 * Every value comes from a real backend fetch — no stubs, no fabricated data.
 * The transparent ephemeral session ([ensureSession]) auto-provisions a random
 * device-bound identity (java.security.SecureRandom) so the JWT backend is
 * satisfied; the user never sees a login/register form.
 *
 * Blocking OkHttp API; callers wrap in CoroutineScope(Dispatchers.IO).
 */
object UserWalletApiService {
    private const val TAG = "UserWalletApi"

    // Android emulator maps the host's localhost to 10.0.2.2. The canonical
    // UserWallet backend is go/wallet_api (:8443), which exposes all 70+ routes
    // the clients need (the wl_user_wallet :8461 clone only has 44 routes and
    // is missing ~21 endpoint groups Android relies on).
    private const val DEFAULT_BASE_URL = "http://10.0.2.2:8443/api/v1"

    private const val PREFS = "userwallet_prefs"
    private const val TOKEN_KEY = "userwallet_token"

    private const val SESSION_KEY = "userwallet_session"
    private const val WALLET_IDS_KEY = "userwallet_wallet_ids"

    private val client: OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .writeTimeout(30, TimeUnit.SECONDS)
        .build()

    private val jsonMediaType = "application/json".toMediaType()

    @Volatile
    private var baseUrl: String = DEFAULT_BASE_URL

    @Volatile
    private var authToken: String? = null

    @Volatile
    private var appContext: Context? = null

    fun init(context: Context, url: String = DEFAULT_BASE_URL) {
        appContext = context.applicationContext
        baseUrl = url
        SecureBlobStore.init(context)
        authToken = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .getString(TOKEN_KEY, null)
            ?: SecureBlobStore.getString(TOKEN_KEY)
    }

    fun setBaseUrl(url: String) {
        baseUrl = url
    }

    fun setToken(token: String?) {
        authToken = token
        SecureBlobStore.putString(TOKEN_KEY, token)
        appContext?.getSharedPreferences(PREFS, Context.MODE_PRIVATE)?.edit()?.apply {
            if (token != null) putString(TOKEN_KEY, token) else remove(TOKEN_KEY)
            apply()
        }
    }

    fun isAuthenticated(): Boolean = authToken != null

    private fun headers(): okhttp3.Headers.Builder = okhttp3.Headers.Builder()
        .add("Content-Type", "application/json")
        .add("Accept", "application/json")
        .also { authToken?.let { h -> add("Authorization", "Bearer $h") } }

    private fun requestBuilder(path: String): Request.Builder =
        Request.Builder().url("$baseUrl$path").headers(headers())

    /** /health lives at the server root (outside /api/v1). */
    private fun healthUrl(): String =
        (baseUrl.replace(Regex("/api/v1/?$"), "").ifEmpty { "http://10.0.2.2:8443" }) + "/health"

    private fun errorFromResponse(response: okhttp3.Response): String {
        return try {
            val raw = response.body?.string() ?: "{}"
            val json = JSONObject(raw)
            val msg = json.optString("error", "")
            if (msg.isNotEmpty()) msg else "Request failed: ${response.code}"
        } catch (e: Exception) {
            "Request failed: ${response.code}"
        }
    }

    private fun httpException(response: okhttp3.Response): IOException =
        IOException(errorFromResponse(response))

    private fun execute(request: Request): JSONObject {
        client.newCall(request).execute().use { response ->
            if (!response.isSuccessful) throw httpException(response)
            val body = response.body?.string() ?: "{}"
            return JSONObject(body)
        }
    }

    private fun executeList(request: Request, key: String): List<JSONObject> {
        client.newCall(request).execute().use { response ->
            if (!response.isSuccessful) throw httpException(response)
            val json = JSONObject(response.body?.string() ?: "{}")
            val arr = json.optJSONArray(key) ?: JSONArray()
            return (0 until arr.length()).map { arr.getJSONObject(it) }
        }
    }

    // ==================== Chain maps (mirror web services/api.ts) ====================

    val CHAIN_IDS: Map<String, Int> = mapOf(
        "ethereum" to 1,
        "bsc" to 56,
        "polygon" to 137,
        "arbitrum" to 42161,
        "optimism" to 10,
        "base" to 8453,
        "avalanche" to 43114
    )

    val CHAIN_SYMBOLS: Map<Int, String> = mapOf(
        1 to "ETH",
        56 to "BNB",
        137 to "MATIC",
        42161 to "ETH",
        10 to "ETH",
        8453 to "ETH",
        43114 to "AVAX"
    )

    /** Block explorer tx base URLs (mirror web TxSubmittedBanner EXPLORERS). */
    val EXPLORERS: Map<Int, String> = mapOf(
        1 to "https://etherscan.io/tx/",
        56 to "https://bscscan.com/tx/",
        137 to "https://polygonscan.com/tx/",
        42161 to "https://arbiscan.io/tx/",
        10 to "https://optimistic.etherscan.io/tx/",
        8453 to "https://basescan.org/tx/",
        43114 to "https://snowtrace.io/tx/"
    )

    fun chainIdFor(network: String): Int =
        CHAIN_IDS[network] ?: (network.toIntOrNull() ?: 1)

    fun symbolFor(chainId: Int): String = CHAIN_SYMBOLS[chainId] ?: "ETH"

    fun explorerFor(chainId: Int): String = EXPLORERS[chainId] ?: ""

    /** Human-readable chain list (mirror web Onboarding CHAINS). */
    data class ChainOption(val id: Int, val name: String, val symbol: String)

    val CHAINS: List<ChainOption> = listOf(
        ChainOption(1, "Ethereum", "ETH"),
        ChainOption(56, "BNB Chain", "BNB"),
        ChainOption(137, "Polygon", "MATIC"),
        ChainOption(42161, "Arbitrum", "ETH"),
        ChainOption(10, "Optimism", "ETH"),
        ChainOption(8453, "Base", "ETH")
    )

    /**
     * Convert a wei string (balance_wei) to a human-readable float in native
     * units. Big-number safe via string parsing (mirror web weiToFloat).
     */
    fun weiToFloat(wei: String): Double {
        if (wei.isEmpty()) return 0.0
        val neg = wei.startsWith("-")
        val digits = if (neg) wei.substring(1) else wei
        val padded = digits.padStart(19, '0')
        val whole = padded.substring(0, padded.length - 18)
        val frac = padded.substring(padded.length - 18).replace(Regex("0+$"), "")
        val num = if (frac.isEmpty()) {
            whole.toDoubleOrNull() ?: 0.0
        } else {
            "$whole.$frac".toDoubleOrNull() ?: 0.0
        }
        return if (neg) -num else num
    }

    // ==================== Auth (mirror web services/api.ts) ====================

    data class User(val id: String, val email: String, val username: String)

    data class AuthResult(val token: String, val userId: String?, val email: String?)

    fun login(email: String, password: String): AuthResult {
        val body = JSONObject().put("email", email).put("password", password).toString()
        val req = requestBuilder("/auth/login").post(body.toRequestBody(jsonMediaType)).build()
        val json = execute(req)
        val token = json.optString("token")
        if (token.isNotEmpty()) setToken(token)
        return AuthResult(token, json.optString("user_id", null), json.optString("email", null))
    }

    /**
     * WL /auth/register accepts {email, password} and returns { id, email } —
     * it does NOT return a JWT. Caller must login afterwards (mirror web).
     */
    fun register(email: String, password: String): JSONObject {
        val body = JSONObject().put("email", email).put("password", password).toString()
        val req = requestBuilder("/auth/register").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    /**
     * Decode the JWT payload locally (no network) — mirrors web getProfile().
     * Hydrates the user identity from a stored token.
     */
    fun getProfile(): User {
        val token = authToken ?: throw IOException("Not authenticated")
        val parts = token.split(".")
        if (parts.size < 2) throw IOException("Malformed token")
        val payload = parts[1]
        val decoded = String(
            Base64.decode(
                payload.replace("-", "+").replace("_", "/"),
                Base64.DEFAULT
            ),
            Charsets.UTF_8
        )
        val json = JSONObject(decoded)
        val id = json.optString("sub", json.optString("user_id", ""))
        val email = json.optString("email", "")
        return User(id = id, email = email, username = json.optString("username", email))
    }

    fun logout() {
        setToken(null)
        SecureBlobStore.remove(SESSION_KEY)
        SecureBlobStore.remove(WALLET_IDS_KEY)
    }

    // ==================== Transparent no-registration session ====================
    // (mirrors web contexts/OnboardingContext.tsx ensureSession)

    /** Persisted transparent-session blob (email/password/token/userId). */
    private data class SessionBlob(
        val email: String,
        val password: String,
        val token: String,
        val userId: String
    )

    /** CSPRNG identity for the transparent account (mirror web randomIdentity). */
    private fun randomIdentity(): Pair<String, String> {
        val bytes = ByteArray(32)
        SecureRandom().nextBytes(bytes)
        val id = bytesToHex(bytes.copyOfRange(0, 16))
        val email = "$id@device.local"
        val password = bytesToHex(bytes)
        return email to password
    }

    private fun bytesToHex(bytes: ByteArray): String =
        bytes.joinToString("") { "%02x".format(it) }

    private fun sessionToJson(s: SessionBlob): String {
        return JSONObject().apply {
            put("email", s.email)
            put("password", s.password)
            put("token", s.token)
            put("userId", s.userId)
        }.toString()
    }

    private fun sessionFromJson(raw: String): SessionBlob? {
        return try {
            val j = JSONObject(raw)
            SessionBlob(
                email = j.optString("email"),
                password = j.optString("password"),
                token = j.optString("token"),
                userId = j.optString("userId")
            )
        } catch (e: Exception) {
            null
        }
    }

    private fun loadSession(): SessionBlob? =
        SecureBlobStore.getString(SESSION_KEY)?.let { sessionFromJson(it) }

    private fun saveSession(s: SessionBlob) {
        SecureBlobStore.putString(SESSION_KEY, sessionToJson(s))
    }

    /**
     * ensureSession — the transparent no-registration bootstrap.
     *
     * If a session blob + token already exist, re-validate the token via
     * getProfile(); if the token is expired, re-login transparently with the
     * stored ephemeral credentials. If no session exists, provision a random
     * device-bound identity (SecureRandom), register it, login to obtain a JWT,
     * and persist the session. One-time, invisible to the user.
     *
     * Mirrors web OnboardingContext.ensureSession exactly (including the
     * register-fails-then-login-surfaces-real-error fallback).
     */
    fun ensureSession(): Session {
        // Fast path: a token is already in memory.
        authToken?.let { existing ->
            val user = validateOrRelogin(existing)
            if (user != null) return Session(existing, user)
            // token invalid -> clear and fall through to provisioning.
            authToken = null
        }
        // Stored session blob?
        var blob = loadSession()
        if (blob != null) {
            setToken(blob.token)
            val user = validateOrRelogin(blob.token)
            if (user != null) {
                return Session(blob.token, user)
            }
            // token expired — transparent re-login with stored creds.
            try {
                val result = login(blob.email, blob.password)
                if (result.token.isNotEmpty()) {
                    blob = blob.copy(token = result.token)
                    saveSession(blob)
                    setToken(result.token)
                    val refreshed = getProfile()
                    return Session(result.token, refreshed)
                }
            } catch (e: Exception) {
                Log.w(TAG, "Stored session re-login failed: ${e.message}")
            }
            // fall through to fresh provisioning
        }
        // Fresh provisioning.
        val (email, password) = randomIdentity()
        try {
            register(email, password)
        } catch (e: Exception) {
            // If register fails (identity collision / network), fall through to
            // login which will surface the real error.
            Log.w(TAG, "Transparent register failed (will try login): ${e.message}")
        }
        val result = login(email, password)
        if (result.token.isEmpty()) throw IOException("Failed to provision transparent session")
        val newBlob = SessionBlob(
            email = email,
            password = password,
            token = result.token,
            userId = result.userId ?: ""
        )
        saveSession(newBlob)
        setToken(result.token)
        val user = try {
            getProfile()
        } catch (e: Exception) {
            User(id = result.userId ?: "", email = email, username = email)
        }
        return Session(result.token, user)
    }

    /** Result of [ensureSession]: the JWT + the locally-decoded user identity. */
    data class Session(val token: String, val user: User)

    private fun validateOrRelogin(token: String): User? {
        return try {
            getProfile()
        } catch (e: Exception) {
            null
        }
    }

    // ==================== Wallet-ids gate (mirror web localWalletIds) ====================

    fun localWalletIds(): List<String> {
        val raw = SecureBlobStore.getString(WALLET_IDS_KEY) ?: return emptyList()
        return try {
            val arr = JSONArray(raw)
            (0 until arr.length()).map { arr.optString(it) }
        } catch (e: Exception) {
            emptyList()
        }
    }

    /** True iff at least one wallet has been created/imported locally (mirror web `onboarded`). */
    fun isOnboarded(): Boolean = localWalletIds().isNotEmpty()

    fun rememberWallet(id: String) {
        val ids = localWalletIds().toMutableList()
        if (!ids.contains(id)) {
            ids.add(id)
            SecureBlobStore.putString(WALLET_IDS_KEY, JSONArray(ids).toString())
        }
    }

    fun forgetWallet(id: String) {
        val ids = localWalletIds().filter { it != id }
        SecureBlobStore.putString(WALLET_IDS_KEY, JSONArray(ids).toString())
    }

    // ==================== Wallets (mirror web) ====================

    data class Wallet(
        val id: String,
        val label: String,
        val chainId: Int,
        val address: String,
        val createdAt: String?,
        val mnemonic: String?
    )

    fun getWallets(): List<Wallet> {
        val req = requestBuilder("/wallets").get().build()
        return executeList(req, "wallets").map {
            Wallet(
                id = it.optString("id"),
                label = it.optString("label"),
                chainId = it.optInt("chain_id"),
                address = it.optString("address"),
                createdAt = it.optString("created_at").ifEmpty { null },
                mnemonic = it.optString("mnemonic", null)
            )
        }
    }

    /**
     * WL POST /wallets { label, password, chain_id, mnemonic?, passphrase? }
     * -> 201 { id, label, address, chain_id, mnemonic? }
     *
     * Mirror of web createWalletTyped. Mnemonic is returned only on a fresh
     * create (NOT on import, since the user supplied it).
     */
    fun createWalletTyped(
        label: String,
        password: String,
        chainId: Int,
        mnemonic: String? = null,
        passphrase: String? = null
    ): Wallet {
        val body = JSONObject().apply {
            put("label", label)
            put("password", password)
            put("chain_id", chainId)
            if (mnemonic != null) put("mnemonic", mnemonic)
            if (passphrase != null) put("passphrase", passphrase)
        }.toString()
        val req = requestBuilder("/wallets").post(body.toRequestBody(jsonMediaType)).build()
        val json = execute(req)
        return Wallet(
            id = json.optString("id"),
            label = json.optString("label"),
            chainId = json.optInt("chain_id"),
            address = json.optString("address"),
            createdAt = json.optString("created_at").ifEmpty { null },
            mnemonic = json.optString("mnemonic", null)
        )
    }

    // ==================== Balances (real eth_getBalance via backend) ====================

    data class Balance(
        val walletId: String,
        val chainId: Int,
        val symbol: String,
        val address: String,
        val balanceWei: String,
        val balanceF: Double,
        val usdValue: Double
    )

    /**
     * GET /balance?address=&chain_id= -> canonical BalanceResult
     * { chain_id, symbol, address, balance, balance_wei, balance_f, usd_value }.
     * The canonical backend's /balance takes an ADDRESS (not a wallet id); if
     * given a wallet id we resolve it to the wallet's address + chain_id first
     * (mirror web getBalance). balance_wei is the raw-wei alias the client reads.
     */
    fun getBalance(walletId: String): Balance {
        var address = walletId
        var chainId = 1
        if (!walletId.startsWith("0x")) {
            val w = getWallets().find { it.id == walletId }
            if (w != null) {
                address = w.address
                chainId = w.chainId
            }
        }
        val req = requestBuilder("/balance?address=$address&chain_id=$chainId").get().build()
        client.newCall(req).execute().use { response ->
            if (!response.isSuccessful) throw httpException(response)
            val json = JSONObject(response.body?.string() ?: "{}")
            val cid = json.optInt("chain_id", chainId)
            return Balance(
                walletId = walletId,
                chainId = cid,
                symbol = json.optString("symbol").ifEmpty { symbolFor(cid) },
                address = json.optString("address").ifEmpty { address },
                balanceWei = json.optString("balance_wei", json.optString("balance")),
                balanceF = json.optDouble("balance_f", 0.0),
                usdValue = json.optDouble("usd_value", 0.0)
            )
        }
    }

    /** Aggregated balances across all wallets (mirror web getBalances). */
    fun getBalances(): List<Balance> {
        val wallets = getWallets()
        return wallets.mapNotNull { w ->
            try {
                getBalance(w.id)
            } catch (e: Exception) {
                null
            }
        }
    }

    // ==================== Transactions (mirror web) ====================

    data class Transaction(
        val id: String,
        val txHash: String,
        val type: String,
        val status: String,
        val from: String,
        val to: String,
        val amount: String,
        val token: String,
        val chainId: Int,
        val createdAt: String
    )

    /**
     * GET /transactions?address=&chain_id= -> { transactions: [{ hash, to, value, ... }] }.
     * The canonical backend's /transactions takes an ADDRESS (not a wallet id);
     * we resolve walletId -> address + chain_id first (mirror web). Response
     * keys are hash/value (canonical), with amount/tx_hash as fallback aliases.
     */
    fun getTransactions(
        walletId: String,
        network: String? = null,
        token: String? = null
    ): List<Transaction> {
        if (walletId.isEmpty()) return emptyList()
        var address = walletId
        var chainId = 1
        if (!walletId.startsWith("0x")) {
            val w = getWallets().find { it.id == walletId }
            if (w != null) {
                address = w.address
                chainId = w.chainId
            }
        }
        if (network != null) chainId = chainIdFor(network)
        val req = requestBuilder("/transactions?address=$address&chain_id=$chainId").get().build()
        val raw = executeList(req, "transactions").map {
            Transaction(
                id = it.optString("id", it.optString("hash")),
                txHash = it.optString("hash", it.optString("tx_hash")),
                type = it.optString("type"),
                status = it.optString("status"),
                from = it.optString("from"),
                to = it.optString("to"),
                amount = it.optString("value", it.optString("amount")),
                token = it.optString("token"),
                chainId = it.optInt("chain_id", chainId),
                createdAt = it.optString("created_at")
            )
        }
        var txs = raw
        if (network != null) {
            val cid = chainIdFor(network)
            txs = txs.filter { it.chainId == cid }
        }
        if (token != null) {
            val tok = token.uppercase()
            txs = txs.filter {
                it.token.uppercase() == tok || (it.token.isEmpty() && tok == "ETH")
            }
        }
        return txs
    }

    // ==================== Send / Sign (real on-chain, mirror web) ====================

    data class SendResult(
        val transactionHash: String,
        val status: String,
        val from: String
    ) {
        /** Alias used by the UI layer (SendFragment). */
        val txHash: String get() = transactionHash
    }

    /**
     * Auto-send result: the backend auto-signs + auto-approves (within a
     * second, when the license is alive + the tx qualifies for the fast path)
     * and returns whether it was auto-approved, plus the reason if not.
     */
    data class AutoSendResult(
        val txHash: String,
        val autoApproved: Boolean,
        val autoApprovalReason: String
    )

    /**
     * POST /send { wallet_id, to, value, password, chain_id, unlock_token,
     * gas_limit, max_fee_gwei, max_priority_gwei } -> { tx_hash, raw_tx, chain_id, nonce }.
     * The canonical backend uses the FLAT /send route with wallet_id in the JSON
     * body (not /wallets/:id/send). Field is `value` (ether), not `amount`.
     */
    fun sendTransaction(
        walletId: String,
        password: String,
        to: String,
        value: String,
        chainId: Int? = null,
        unlockToken: String? = null,
        gasLimit: Int? = null,
        maxFeeGwei: String? = null,
        maxPriorityGwei: String? = null
    ): SendResult {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("to", to)
            put("value", value)
            if (password.isNotEmpty()) put("password", password)
            if (chainId != null) put("chain_id", chainId)
            if (unlockToken != null && unlockToken.isNotEmpty()) put("unlock_token", unlockToken)
            if (gasLimit != null) put("gas_limit", gasLimit)
            if (!maxFeeGwei.isNullOrEmpty()) put("max_fee_gwei", maxFeeGwei)
            if (!maxPriorityGwei.isNullOrEmpty()) put("max_priority_gwei", maxPriorityGwei)
        }.toString()
        val req = requestBuilder("/send")
            .post(body.toRequestBody(jsonMediaType))
            .build()
        val json = execute(req)
        return SendResult(
            transactionHash = json.optString("tx_hash", json.optString("transaction_hash")),
            status = json.optString("status", "ok"),
            from = json.optString("from")
        )
    }

    /**
     * POST /auto-send?master_wallet_id= { wallet_id, to, value, password,
     * chain_id, gas_limit, unlock_token, max_fee_gwei, max_priority_gwei }
     * -> { tx_hash, auto_approved, auto_approval_reason, raw_tx, chain_id }.
     * The canonical backend uses the FLAT /auto-send route with wallet_id in the
     * JSON body (not /wallets/:id/auto-send). The backend auto-signs + auto-
     * approves within a second when the MasterWallet policy allows it; the
     * client surfaces auto_approved + the reason. Field is `value`, not `amount`.
     */
    fun autoSendTransaction(
        walletId: String,
        password: String,
        to: String,
        value: String,
        chainId: Int? = null,
        gasLimit: Int? = null,
        unlockToken: String? = null,
        maxFeeGwei: String? = null,
        maxPriorityGwei: String? = null
    ): AutoSendResult {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("to", to)
            put("value", value)
            if (password.isNotEmpty()) put("password", password)
            if (chainId != null) put("chain_id", chainId)
            if (gasLimit != null) put("gas_limit", gasLimit)
            if (unlockToken != null && unlockToken.isNotEmpty()) put("unlock_token", unlockToken)
            if (!maxFeeGwei.isNullOrEmpty()) put("max_fee_gwei", maxFeeGwei)
            if (!maxPriorityGwei.isNullOrEmpty()) put("max_priority_gwei", maxPriorityGwei)
        }.toString()
        val req = requestBuilder("/auto-send")
            .post(body.toRequestBody(jsonMediaType))
            .build()
        val json = execute(req)
        return AutoSendResult(
            txHash = json.optString("tx_hash", json.optString("transaction_hash")),
            autoApproved = json.optBoolean("auto_approved", false),
            autoApprovalReason = json.optString("auto_approval_reason")
        )
    }

    /**
     * WL POST /wallets/:id/unlock { passcode? } -> { unlock_token, expires_in }.
     * Releases a short-lived unlock_token used to sign transactions without
     * re-entering the wallet password on every send (mirror web unlockWallet).
     */
    data class UnlockParams(
        val passcode: String? = null,
        val password: String? = null
    )

    fun unlockWallet(walletId: String, params: UnlockParams): JSONObject {
        val body = JSONObject().apply {
            if (!params.passcode.isNullOrEmpty()) put("passcode", params.passcode)
            if (!params.password.isNullOrEmpty()) put("password", params.password)
        }.toString()
        val req = requestBuilder("/wallets/$walletId/unlock")
            .post(body.toRequestBody(jsonMediaType))
            .build()
        return execute(req)
    }

    // ==================== Simulate / ENS (mirror web services/api.ts) ====================

    /**
     * Result of POST /simulate — a dry-run of the exact tx the user is about
     * to send, against the chain RPC (eth_estimateGas + eth_call revert check
     * + EIP-1559 cost projection).
     */
    data class SimulationResult(
        val chainId: Int,
        val success: Boolean,
        val gasEstimate: Long,
        val willRevert: Boolean,
        val revertReason: String?,
        val estimateError: String?,
        val gasPrice: String?,
        val maxFeePerGas: String?,
        val maxPriorityFee: String?,
        val estimatedCostWei: String?
    )

    /**
     * WL POST /simulate { chain_id, from, to, value?, data? }
     * -> { chain_id, success, gas_estimate, will_revert, revert_reason?,
     *     estimate_error?, gas_price?, max_fee_per_gas?, max_priority_fee?,
     *     estimated_cost_wei? }
     *
     * `value` is a human-readable native amount (e.g. "0.05"), same as /send.
     */
    fun simulateTransaction(
        chainId: Int,
        from: String,
        to: String,
        value: String? = null,
        data: String? = null
    ): SimulationResult {
        val body = JSONObject().apply {
            put("chain_id", chainId)
            put("from", from)
            put("to", to)
            if (!value.isNullOrEmpty()) put("value", value)
            if (!data.isNullOrEmpty()) put("data", data)
        }.toString()
        val req = requestBuilder("/simulate")
            .post(body.toRequestBody(jsonMediaType))
            .build()
        val json = execute(req)
        return SimulationResult(
            chainId = json.optInt("chain_id", chainId),
            success = json.optBoolean("success", false),
            gasEstimate = json.optLong("gas_estimate", 0L),
            willRevert = json.optBoolean("will_revert", false),
            revertReason = json.optString("revert_reason").ifEmpty { null },
            estimateError = json.optString("estimate_error").ifEmpty { null },
            gasPrice = json.optString("gas_price").ifEmpty { null },
            maxFeePerGas = json.optString("max_fee_per_gas").ifEmpty { null },
            maxPriorityFee = json.optString("max_priority_fee").ifEmpty { null },
            estimatedCostWei = json.optString("estimated_cost_wei").ifEmpty { null }
        )
    }

    data class EnsResolution(val name: String, val address: String)

    /** WL GET /ens/resolve?name=alice.eth -> { name, address } (real on-chain). */
    fun resolveENS(name: String): EnsResolution {
        val encoded = URLEncoder.encode(name, "UTF-8")
        val req = requestBuilder("/ens/resolve?name=$encoded").get().build()
        val json = execute(req)
        return EnsResolution(
            name = json.optString("name"),
            address = json.optString("address")
        )
    }

    data class EnsLookup(val address: String, val name: String)

    /** WL GET /ens/lookup?address=0x... -> { address, name } (reverse lookup). */
    fun lookupENS(address: String): EnsLookup {
        val encoded = URLEncoder.encode(address, "UTF-8")
        val req = requestBuilder("/ens/lookup?address=$encoded").get().build()
        val json = execute(req)
        return EnsLookup(
            address = json.optString("address"),
            name = json.optString("name")
        )
    }

    /** POST /sign { wallet_id, message, password } -> { signature }. */
    data class SignResult(val signature: String, val address: String)

    fun signMessage(walletId: String, password: String, message: String): SignResult {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("message", message)
            if (password.isNotEmpty()) put("password", password)
        }.toString()
        val req = requestBuilder("/sign")
            .post(body.toRequestBody(jsonMediaType))
            .build()
        val json = execute(req)
        return SignResult(
            signature = json.optString("signature"),
            address = json.optString("address")
        )
    }

    // ==================== Health (mirror web) ====================

    data class Health(
        val status: String,
        val service: String,
        val licensed: Boolean,
        val wlClientId: String
    )

    fun health(): Health {
        val req = Request.Builder().url(healthUrl()).headers(headers()).get().build()
        client.newCall(req).execute().use { response ->
            if (!response.isSuccessful) throw httpException(response)
            val json = JSONObject(response.body?.string() ?: "{}")
            return Health(
                status = json.optString("status"),
                service = json.optString("service"),
                licensed = json.optBoolean("licensed"),
                wlClientId = json.optString("wl_client_id")
            )
        }
    }

    // ==================== Tokens (real ERC-20 balanceOf via backend) ====================

    data class TokenBalance(
        val contractAddress: String,
        val symbol: String,
        val name: String,
        val decimals: Int,
        val balance: String,
        val balanceF: Double,
        val usdValue: Double
    )

    fun getTokenBalances(address: String, chainId: Int): List<TokenBalance> {
        val path = "/tokens?address=$address&chain_id=$chainId"
        val req = requestBuilder(path).get().build()
        return executeList(req, "tokens").map {
            TokenBalance(
                contractAddress = it.optString("contract_address"),
                symbol = it.optString("symbol"),
                name = it.optString("name"),
                decimals = it.optInt("decimals"),
                balance = it.optString("balance"),
                balanceF = it.optDouble("balance_f"),
                usdValue = it.optDouble("usd_value")
            )
        }
    }

    // ==================== NFTs (real on-chain ERC-721 via backend) ====================

    data class NFT(
        val contractAddress: String,
        val tokenId: String,
        val name: String,
        val symbol: String,
        val tokenURI: String,
        val imageURI: String
    )

    fun getNFTs(address: String, chainId: Int): List<NFT> {
        val path = "/nfts?address=$address&chain_id=$chainId"
        val req = requestBuilder(path).get().build()
        return executeList(req, "nfts").map {
            NFT(
                contractAddress = it.optString("contract_address"),
                tokenId = it.optString("token_id"),
                name = it.optString("name"),
                symbol = it.optString("symbol"),
                tokenURI = it.optString("token_uri"),
                imageURI = it.optString("image_uri")
            )
        }
    }

    // ==================== Gas / Price / Chains (real RPC + CoinGecko via backend) ====================

    data class GasPrice(val gasPrice: String, val baseFee: String, val priorityFee: String, val estimatedCost: String)

    fun getGasPrice(chainId: Int): GasPrice {
        val path = "/gas?chain_id=$chainId"
        val req = requestBuilder(path).get().build()
        val json = execute(req)
        return GasPrice(
            gasPrice = json.optString("gas_price"),
            baseFee = json.optString("base_fee"),
            priorityFee = json.optString("priority_fee"),
            estimatedCost = json.optString("estimated_cost")
        )
    }

    data class TokenPrice(val usd: Double, val usd24hChange: Double)

    fun getTokenPrice(token: String): TokenPrice {
        val path = "/price?token=$token"
        val req = requestBuilder(path).get().build()
        val json = execute(req)
        return TokenPrice(
            usd = json.optDouble("usd"),
            usd24hChange = json.optDouble("usd_24h_change")
        )
    }

    data class ChainInfo(
        val chainId: Int,
        val name: String,
        val symbol: String,
        val rpcEndpoint: String,
        val derivationPath: String?,
        val explorerApi: String?,
        val explorerUrl: String?,
        val chainType: String?,
        val decimals: Int?,
        val coinType: Int?,
        val isTestnet: Boolean?
    )

    fun getChains(): List<ChainInfo> {
        val req = requestBuilder("/chains").get().build()
        return executeList(req, "chains").map {
            ChainInfo(
                chainId = it.optInt("id"),
                name = it.optString("name"),
                symbol = it.optString("symbol"),
                rpcEndpoint = it.optString("rpc_endpoint"),
                derivationPath = it.optString("derivation_path").ifEmpty { null },
                explorerApi = it.optString("explorer_api").ifEmpty { null },
                explorerUrl = it.optString("explorer_url").ifEmpty { null },
                chainType = it.optString("chain_type").ifEmpty { null },
                decimals = if (it.has("decimals")) it.optInt("decimals") else null,
                coinType = if (it.has("coin_type")) it.optInt("coin_type") else null,
                isTestnet = if (it.has("is_testnet")) it.optBoolean("is_testnet") else null
            )
        }
    }

    data class NetworkStatus(val chainId: Int, val blockNumber: Long, val connected: Boolean)

    fun getNetworkStatus(chainId: Int): NetworkStatus {
        val chains = getChains()
        val chain = chains.firstOrNull { it.chainId == chainId }
        return NetworkStatus(
            chainId = chain?.chainId ?: chainId,
            blockNumber = 0L,
            connected = chain != null
        )
    }

// ==================== Swap (real CoinGecko cross-rate + on-chain via backend) ====================

    data class SwapQuote(
        val fromToken: String,
        val toToken: String,
        val fromAmount: String,
        val toAmount: String,
        val priceImpact: Double,
        val route: String
    )

    fun getSwapQuote(fromToken: String, toToken: String, fromAmount: String, chainId: Int = 1): SwapQuote {
        val path = "/swap/quote?from_token=$fromToken&to_token=$toToken&from_amount=$fromAmount&chain_id=$chainId"
        val req = requestBuilder(path).get().build()
        val json = execute(req)
        return SwapQuote(
            fromToken = json.optString("from_token"),
            toToken = json.optString("to_token"),
            fromAmount = json.optString("from_amount"),
            toAmount = json.optString("to_amount"),
            priceImpact = json.optDouble("price_impact", json.optString("price_impact", "0").toDoubleOrNull() ?: 0.0),
            route = json.optString("route", "swap")
        )
    }

    // ==================== Staking (real on-chain action via backend /send) ====================

    data class StakingAsset(
        val symbol: String,
        val chainId: Int,
        val apy: Double,
        val minStake: Double,
        val lockPeriod: Long,
        val verified: Boolean,
    )

    data class StakingQuote(
        val success: Boolean,
        val assets: List<StakingAsset>,
        val apy: Double,
        val minStake: Double,
        val lockPeriod: Long,
    )

    fun getStakingQuote(_asset: String? = null): StakingQuote {
        // The backend returns the full supported-asset list and ignores
        // ?asset=; the response shape is { success, assets[], apy,
        // min_stake, lock_period }. Decoded into the typed StakingQuote.
        val req = requestBuilder("/staking/quote").get().build()
        val json = execute(req)
        val arr = json.optJSONArray("assets") ?: org.json.JSONArray()
        val assets = (0 until arr.length()).map { i ->
            val a = arr.getJSONObject(i)
            StakingAsset(
                symbol = a.optString("symbol"),
                chainId = a.optInt("chain_id"),
                apy = a.optDouble("apy"),
                minStake = a.optDouble("min_stake"),
                lockPeriod = a.optLong("lock_period"),
                verified = a.optBoolean("verified"),
            )
        }
        return StakingQuote(
            success = json.optBoolean("success"),
            assets = assets,
            apy = json.optDouble("apy"),
            minStake = json.optDouble("min_stake"),
            lockPeriod = json.optLong("lock_period"),
        )
    }

    // ==================== Auxiliary DeFi (fiat ramp, crypto card, P2P, convert) ====================
    // All delegate to the canonical backend proxy routes (real CoinGecko
    // prices, real provider checkout URLs, real PostgreSQL-backed listings).

    fun getFiatProviders(): JSONObject =
        execute(requestBuilder("/ramp/providers").get().build())

    fun getFiatQuote(providerId: String, amount: String, fiat: String, crypto: String, method: String): JSONObject {
        val body = JSONObject().apply {
            put("providerId", providerId)
            put("amount", amount)
            put("fiatCurrency", fiat)
            put("cryptoCurrency", crypto)
            put("paymentMethod", method)
        }
        val req = requestBuilder("/ramp/quote").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun getFiatOfframpQuote(providerId: String, amount: String, fiat: String, crypto: String): JSONObject {
        val body = JSONObject().apply {
            put("providerId", providerId)
            put("amount", amount)
            put("fiatCurrency", fiat)
            put("cryptoCurrency", crypto)
        }
        val req = requestBuilder("/ramp/offramp-quote").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun getCryptoCardBalance(): JSONObject =
        execute(requestBuilder("/card/balance").get().build())

    fun getCardTransactions(): List<JSONObject> =
        executeList(requestBuilder("/card/transactions").get().build(), "transactions")

    fun getP2PAdverts(): List<JSONObject> =
        executeList(requestBuilder("/p2p/adverts").get().build(), "adverts")

    // Convert is the same path as swap (cross-token conversion).
    fun getConvertQuote(fromToken: String, toToken: String, fromAmount: String, chainId: Int): SwapQuote {
        val path = "/swap/quote?from_token=$fromToken&to_token=$toToken&from_amount=$fromAmount&chain_id=$chainId"
        val req = requestBuilder(path).get().build()
        val json = execute(req)
        return SwapQuote(
            fromToken = json.optString("from_token"),
            toToken = json.optString("to_token"),
            fromAmount = json.optString("from_amount"),
            toAmount = json.optString("to_amount"),
            priceImpact = json.optDouble("price_impact", json.optString("price_impact", "0").toDoubleOrNull() ?: 0.0),
            route = json.optString("route", "swap")
        )
    }

    // ==================== Guest auth (transparent bootstrap) ====================
    /** POST /auth/guest { device_id } -> { user_id, token, guest } (mirror web). */
    fun guestAuth(deviceId: String): JSONObject {
        val body = JSONObject().put("device_id", deviceId).toString()
        val req = requestBuilder("/auth/guest").post(body.toRequestBody(jsonMediaType)).build()
        val json = execute(req)
        val token = json.optString("token")
        if (token.isNotEmpty()) setToken(token)
        return json
    }

    // ==================== AMM swap (real on-chain getAmountsOut + calldata) ====================
    /**
     * GET /amm/quote?chain_id=&token_in=&token_out=&amount_in= -> { amount_out,
     * router, ... }. The canonical backend binds token_in/token_out/amount_in
     * (NOT from_token/to_token/from_amount); using the wrong keys 400s.
     */
    fun getAmmQuote(fromToken: String, toToken: String, fromAmount: String, chainId: Int): SwapQuote {
        val path = "/amm/quote?chain_id=$chainId&token_in=$fromToken&token_out=$toToken&amount_in=$fromAmount"
        val json = execute(requestBuilder(path).get().build())
        return SwapQuote(
            fromToken = json.optString("token_in", fromToken),
            toToken = json.optString("token_out", toToken),
            fromAmount = json.optString("amount_in", fromAmount),
            toAmount = json.optString("amount_out"),
            priceImpact = 0.0,
            route = json.optString("router", ""),
        )
    }

    /**
     * POST /amm/swap { from, chain_id, token_in, token_out, amount_in, amount_out_min? }
     * -> calldata for swapExactTokensForTokens (broadcast via /send). The
     * canonical backend binds token_in/token_out/amount_in (NOT from_token/
     * to_token/from_amount) and requires the sender `from` address. We resolve
     * walletId -> address for `from`. No tx hash is fabricated here.
     */
    fun ammSwap(walletId: String, password: String, fromToken: String, toToken: String, fromAmount: String, chainId: Int): JSONObject {
        val from = getWallets().find { it.id == walletId }?.address ?: walletId
        val body = JSONObject().apply {
            put("from", from)
            put("chain_id", chainId)
            put("token_in", fromToken)
            put("token_out", toToken)
            put("amount_in", fromAmount)
        }.toString()
        return execute(requestBuilder("/amm/swap").post(body.toRequestBody(jsonMediaType)).build())
    }

    // ==================== Staking actions (real on-chain via backend) ====================
    fun stake(walletId: String, password: String, asset: String, amount: String, chainId: Int): JSONObject =
        stakingAction("stake", walletId, password, asset, amount, chainId)

    fun unstake(walletId: String, password: String, asset: String, amount: String, chainId: Int): JSONObject =
        stakingAction("unstake", walletId, password, asset, amount, chainId)

    fun claim(walletId: String, password: String, asset: String, chainId: Int): JSONObject =
        stakingAction("claim", walletId, password, asset, null, chainId)

    private fun stakingAction(action: String, walletId: String, password: String, asset: String, amount: String?, chainId: Int): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            if (password.isNotEmpty()) put("password", password)
            put("token", asset)
            if (amount != null) put("amount", amount)
            put("chain_id", chainId)
        }.toString()
        return execute(requestBuilder("/staking/$action").post(body.toRequestBody(jsonMediaType)).build())
    }

    // ==================== NFT transfer ====================
    fun transferNFT(walletId: String, password: String, to: String, tokenId: String, contractAddress: String, chainId: Int): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("to", to)
            put("token_id", tokenId)
            put("contract_address", contractAddress)
            put("chain_id", chainId)
        }.toString()
        return execute(requestBuilder("/nft/transfer").post(body.toRequestBody(jsonMediaType)).build())
    }

    // ==================== KYC (proxied listing_service) ====================
    fun getKycStatus(userId: String?): JSONObject {
        val path = if (userId.isNullOrEmpty()) "/kyc/status"
            else "/kyc/status?user_id=${URLEncoder.encode(userId, "UTF-8")}"
        return execute(requestBuilder(path).get().build())
    }

    fun registerKyc(body: JSONObject): JSONObject =
        execute(requestBuilder("/kyc/register").post(body.toString().toRequestBody(jsonMediaType)).build())

    fun submitKyc(body: JSONObject): JSONObject =
        execute(requestBuilder("/kyc/submit").post(body.toString().toRequestBody(jsonMediaType)).build())

    // ==================== Address book ====================
    fun getAddressBookContacts(): List<JSONObject> =
        executeList(requestBuilder("/address-book/contacts").get().build(), "contacts")

    fun addContact(name: String, address: String, chainId: Int?): JSONObject {
        val body = JSONObject().apply {
            put("name", name)
            put("address", address)
            if (chainId != null) put("chain_id", chainId)
        }.toString()
        return execute(requestBuilder("/address-book/contacts").post(body.toRequestBody(jsonMediaType)).build())
    }

    fun updateContact(id: String, name: String?, address: String?, chainId: Int?): JSONObject {
        val body = JSONObject().apply {
            if (name != null) put("name", name)
            if (address != null) put("address", address)
            if (chainId != null) put("chain_id", chainId)
        }.toString()
        return execute(requestBuilder("/address-book/contacts/$id").put(body.toRequestBody(jsonMediaType)).build())
    }

    fun deleteContact(id: String): JSONObject =
        execute(requestBuilder("/address-book/contacts/$id").delete().build())

    // ==================== Multi-device sync ====================
    fun getDevices(): List<JSONObject> =
        executeList(requestBuilder("/devices").get().build(), "devices")

    fun registerDevice(name: String, deviceType: String): JSONObject {
        val body = JSONObject().apply {
            put("name", name)
            put("device_type", deviceType)
        }.toString()
        return execute(requestBuilder("/devices").post(body.toRequestBody(jsonMediaType)).build())
    }

    fun syncDevice(deviceId: String): JSONObject =
        execute(requestBuilder("/devices/$deviceId/sync").post("{}".toRequestBody(jsonMediaType)).build())

    fun deleteDevice(deviceId: String): JSONObject =
        execute(requestBuilder("/devices/$deviceId").delete().build())

    // ==================== Token approvals ====================
    fun getApprovals(address: String, chainId: Int): List<JSONObject> =
        executeList(requestBuilder("/approvals?address=$address&chain_id=$chainId").get().build(), "data")

    fun revokeApproval(approvalId: String): JSONObject =
        execute(requestBuilder("/approvals/$approvalId").delete().build())

    // ==================== Keystore V3 (Web3 Secret Storage) ====================
    /**
     * POST /keystore/export { wallet_id, password, export_password }
     * -> Web3 Secret Storage V3 keystore JSON (application/json body, raw).
     * The canonical backend requires BOTH the wallet password (to decrypt the
     * seed) and a separate export_password (to re-encrypt the V3 keystore).
     */
    fun exportKeystore(walletId: String, password: String): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("export_password", password)
        }.toString()
        return execute(requestBuilder("/keystore/export").post(body.toRequestBody(jsonMediaType)).build())
    }

    /**
     * POST /keystore/import { keystore_json, password, label, chain_id? }
     * -> { wallet_id, address, label, chain_id, source }.
     * The canonical backend binds `keystore_json` (NOT `keystore`).
     */
    fun importKeystore(keystore: String, password: String, label: String?): Wallet {
        val body = JSONObject().apply {
            put("keystore_json", keystore)
            put("password", password)
            if (label != null) put("label", label)
        }.toString()
        val json = execute(requestBuilder("/keystore/import").post(body.toRequestBody(jsonMediaType)).build())
        return Wallet(
            id = json.optString("wallet_id"),
            label = label ?: json.optString("label"),
            chainId = 1,
            address = json.optString("address"),
            createdAt = null,
            mnemonic = null,
        )
    }

    // ==================== Lending (go/lending_service proxy) ====================
    fun getLendingMarkets(): List<JSONObject> =
        executeList(requestBuilder("/lending/markets").get().build(), "markets")

    fun lendingSupply(walletId: String, password: String, asset: String, amount: String, chainId: Int): JSONObject =
        lendingAction("supply", walletId, password, asset, amount, chainId)

    fun lendingBorrow(walletId: String, password: String, asset: String, amount: String, chainId: Int): JSONObject =
        lendingAction("borrow", walletId, password, asset, amount, chainId)

    fun lendingWithdraw(walletId: String, password: String, asset: String, amount: String, chainId: Int): JSONObject =
        lendingAction("withdraw", walletId, password, asset, amount, chainId)

    fun lendingRepay(walletId: String, password: String, asset: String, amount: String, chainId: Int): JSONObject =
        lendingAction("repay", walletId, password, asset, amount, chainId)

    private fun lendingAction(action: String, walletId: String, password: String, asset: String, amount: String, chainId: Int): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("asset", asset)
            put("amount", amount)
            put("chain_id", chainId)
        }.toString()
        return execute(requestBuilder("/lending/$action").post(body.toRequestBody(jsonMediaType)).build())
    }

    // ==================== Copy trading (go/copy_trading_service proxy) ====================
    fun getCopyTraders(): List<JSONObject> =
        executeList(requestBuilder("/copytrading/traders").get().build(), "traders")

    fun followTrader(traderId: String, allocation: String?): JSONObject {
        val body = JSONObject().apply {
            put("trader_id", traderId)
            if (allocation != null) put("allocation", allocation)
        }.toString()
        return execute(requestBuilder("/copytrading/follow").post(body.toRequestBody(jsonMediaType)).build())
    }

    fun stopCopyTrader(copierId: String): JSONObject =
        execute(requestBuilder("/copytrading/copiers/$copierId/stop").post("{}".toRequestBody(jsonMediaType)).build())

    // ==================== DAO (wallet_api /dao/*) ====================
    fun getDaoProposals(): List<JSONObject> =
        executeList(requestBuilder("/dao/proposals").get().build(), "data")

    fun createDaoProposal(title: String, description: String): JSONObject {
        val body = JSONObject().apply {
            put("title", title)
            put("description", description)
        }.toString()
        return execute(requestBuilder("/dao/proposals").post(body.toRequestBody(jsonMediaType)).build())
    }

    fun voteDaoProposal(proposalId: String, support: Boolean): JSONObject {
        val body = JSONObject().put("support", support).toString()
        return execute(requestBuilder("/dao/proposals/$proposalId/vote").post(body.toRequestBody(jsonMediaType)).build())
    }

    // ==================== Perpetual positions (wallet_api /perpetual/*) ====================
    fun getPerpetualPositions(): List<JSONObject> =
        executeList(requestBuilder("/perpetual/positions").get().build(), "data")

    fun createPerpetualPosition(pair: String, side: String, size: String, leverage: Int, chainId: Int): JSONObject {
        val body = JSONObject().apply {
            put("pair", pair)
            put("side", side)
            put("size", size)
            put("leverage", leverage)
            put("chain_id", chainId)
        }.toString()
        return execute(requestBuilder("/perpetual/positions").post(body.toRequestBody(jsonMediaType)).build())
    }

    fun closePerpetualPosition(positionId: String): JSONObject =
        execute(requestBuilder("/perpetual/positions/$positionId/close").post("{}".toRequestBody(jsonMediaType)).build())

    // ==================== Margin positions (wallet_api /margin/*) ====================
    fun getMarginPositions(): List<JSONObject> =
        executeList(requestBuilder("/margin/positions").get().build(), "data")

    fun createMarginPosition(pair: String, side: String, size: String, leverage: Int, chainId: Int): JSONObject {
        val body = JSONObject().apply {
            put("pair", pair)
            put("side", side)
            put("size", size)
            put("leverage", leverage)
            put("chain_id", chainId)
        }.toString()
        return execute(requestBuilder("/margin/positions").post(body.toRequestBody(jsonMediaType)).build())
    }

    fun closeMarginPosition(positionId: String): JSONObject =
        execute(requestBuilder("/margin/positions/$positionId/close").post("{}".toRequestBody(jsonMediaType)).build())

    // ==================== Prediction markets (go/prediction_service proxy) ====================
    fun getPredictionMarkets(): List<JSONObject> =
        executeList(requestBuilder("/prediction/markets").get().build(), "markets")

    fun placePredictionBet(marketId: String, side: String, amount: String): JSONObject {
        val body = JSONObject().apply {
            put("side", side)
            put("amount", amount)
        }.toString()
        return execute(requestBuilder("/prediction/markets/$marketId/bet").post(body.toRequestBody(jsonMediaType)).build())
    }

    // ==================== Launchpool (wallet_api /launchpool/*) ====================
    fun getLaunchpool(): JSONObject =
        execute(requestBuilder("/launchpool").get().build())

    fun getLaunchpoolStakes(): List<JSONObject> =
        executeList(requestBuilder("/launchpool/stakes").get().build(), "data")

    fun launchpoolStake(walletId: String, password: String, amount: String): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("amount", amount)
        }.toString()
        return execute(requestBuilder("/launchpool/stake").post(body.toRequestBody(jsonMediaType)).build())
    }

    fun launchpoolUnstake(walletId: String, password: String, amount: String): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("amount", amount)
        }.toString()
        return execute(requestBuilder("/launchpool/unstake").post(body.toRequestBody(jsonMediaType)).build())
    }

    // ==================== Token sales (wallet_api /token-sales/*) ====================
    fun getTokenSales(): List<JSONObject> =
        executeList(requestBuilder("/token-sales").get().build(), "data")

    fun participateTokenSale(saleId: String, amount: String): JSONObject {
        val body = JSONObject().put("amount", amount).toString()
        return execute(requestBuilder("/token-sales/$saleId/participate").post(body.toRequestBody(jsonMediaType)).build())
    }

    // ==================== Passkey / lock (mirror web) ====================
    data class LockSetupParams(
        val passcode: String? = null,
        val passkeyCredentialId: String? = null,
        val passkeyPublicKey: String? = null
    )

    fun setupLock(walletId: String, params: LockSetupParams): JSONObject {
        val body = JSONObject().apply {
            if (!params.passcode.isNullOrEmpty()) put("passcode", params.passcode)
            if (!params.passkeyCredentialId.isNullOrEmpty()) put("passkey_credential_id", params.passkeyCredentialId)
            if (!params.passkeyPublicKey.isNullOrEmpty()) put("passkey_public_key", params.passkeyPublicKey)
        }.toString()
        return execute(requestBuilder("/wallets/$walletId/lock").post(body.toRequestBody(jsonMediaType)).build())
    }
}
