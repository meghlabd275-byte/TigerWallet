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

    /**
     * PUT /api/v1/master-wallet/:id — update wallet metadata/limits. Only non-null
     * fields are sent so the backend leaves the others untouched. Returns the backend
     * id and whether anything actually changed.
     */
    suspend fun updateMasterWallet(
        masterId: String,
        name: String? = null,
        isActive: Boolean? = null,
        dailyLimit: java.math.BigDecimal? = null,
        perTransactionLimit: java.math.BigDecimal? = null,
        metadata: Map<String, String>? = null
    ): UpdateResult = withContext(Dispatchers.IO) {
        try {
            val body = JSONObject()
            name?.let { body.put("name", it) }
            isActive?.let { body.put("is_active", it) }
            dailyLimit?.let { body.put("daily_limit", it.toPlainString()) }
            perTransactionLimit?.let { body.put("per_transaction_limit", it.toPlainString()) }
            metadata?.let {
                val meta = JSONObject()
                it.forEach { (k, v) -> meta.put(k, v) }
                body.put("metadata", meta)
            }
            val resp = apiPut("/api/v1/master-wallet/$masterId", body.toString())
                ?: return@withContext UpdateResult(success = false, error = "Update request failed")
            val json = JSONObject(resp)
            UpdateResult(
                success = true,
                id = json.optString("id", masterId),
                updated = json.optBoolean("updated", false)
            )
        } catch (e: Exception) {
            UpdateResult(success = false, error = e.message)
        }
    }

    /**
     * GET /api/v1/master-wallet/:id/transactions/:tid — fetch a single transaction
     * by id. Returns the raw transaction object the backend produced (real on-chain
     * data, never fabricated).
     */
    suspend fun getTransaction(masterId: String, txId: String): TransactionDetailResult =
        withContext(Dispatchers.IO) {
            try {
                val resp = apiGet("/api/v1/master-wallet/$masterId/transactions/$txId")
                    ?: return@withContext TransactionDetailResult(success = false, error = "Transaction fetch failed")
                val json = JSONObject(resp)
                val tx = json.optJSONObject("transaction") ?: json
                TransactionDetailResult(
                    success = true,
                    transaction = tx.toString()
                )
            } catch (e: Exception) {
                TransactionDetailResult(success = false, error = e.message)
            }
        }

    /**
     * GET /api/v1/master-wallet/:id/multisig/wallets/:wid — fetch a multisig wallet
     * (owners, threshold, chain, address, optional pending transactions).
     */
    suspend fun getMultisigWalletDetail(masterId: String, walletId: String): MultisigWalletDetailResult =
        withContext(Dispatchers.IO) {
            try {
                val resp = apiGet("/api/v1/master-wallet/$masterId/multisig/wallets/$walletId")
                    ?: return@withContext MultisigWalletDetailResult(success = false, error = "Multisig fetch failed")
                val json = JSONObject(resp)
                val mw = json.optJSONObject("multisig_wallet") ?: json
                val owners = mutableListOf<String>()
                mw.optJSONArray("owners")?.let { arr ->
                    for (i in 0 until arr.length()) owners.add(arr.optString(i))
                }
                val pending = mutableListOf<String>()
                mw.optJSONArray("pending_transactions")?.let { arr ->
                    for (i in 0 until arr.length()) pending.add(arr.optString(i))
                }
                MultisigWalletDetailResult(
                    success = true,
                    wallet = MultisigWalletDetail(
                        id = mw.optString("id", walletId),
                        name = mw.optString("name", ""),
                        owners = owners,
                        threshold = mw.optInt("threshold", 0),
                        chainId = mw.optLong("chain_id", 0L),
                        address = mw.optString("address", ""),
                        pendingTransactions = pending
                    )
                )
            } catch (e: Exception) {
                MultisigWalletDetailResult(success = false, error = e.message)
            }
        }

    /**
     * POST /api/v1/master-wallet/:id/passkey/register — register a WebAuthn
     * credential with the backend. credentialId/publicKey are base64url, publicKey
     * is the X.509/SPKI P-256 public key. Returns the server-assigned passkey id.
     */
    suspend fun registerPasskey(
        masterId: String,
        credentialId: String,
        publicKey: String,
        signCount: Long,
        transports: List<String>,
        label: String
    ): PasskeyRegisterResult = withContext(Dispatchers.IO) {
        try {
            val transportsArr = JSONArray()
            transports.forEach { transportsArr.put(it) }
            val body = JSONObject()
                .put("credential_id", credentialId)
                .put("public_key", publicKey)
                .put("sign_count", signCount)
                .put("transports", transportsArr)
                .put("label", label)
                .toString()
            val resp = apiPost("/api/v1/master-wallet/$masterId/passkey/register", body)
                ?: return@withContext PasskeyRegisterResult(success = false, error = "Passkey register failed")
            val json = JSONObject(resp)
            PasskeyRegisterResult(
                success = true,
                passkeyId = json.optString("passkey_id", ""),
                credentialId = json.optString("credential_id", credentialId),
                registered = json.optBoolean("registered", false)
            )
        } catch (e: Exception) {
            PasskeyRegisterResult(success = false, error = e.message)
        }
    }

    /**
     * GET /api/v1/master-wallet/:id/passkey/credentials — list registered passkeys.
     */
    suspend fun listPasskeys(masterId: String): PasskeyListResult = withContext(Dispatchers.IO) {
        try {
            val resp = apiGet("/api/v1/master-wallet/$masterId/passkey/credentials")
                ?: return@withContext PasskeyListResult(success = false, error = "Passkey list failed")
            val json = JSONObject(resp)
            val arr = json.optJSONArray("passkeys") ?: JSONArray()
            val list = mutableListOf<PasskeyCredential>()
            for (i in 0 until arr.length()) {
                val p = arr.getJSONObject(i)
                val transports = mutableListOf<String>()
                p.optJSONArray("transports")?.let { t ->
                    for (j in 0 until t.length()) transports.add(t.optString(j))
                }
                list.add(
                    PasskeyCredential(
                        id = p.optString("id"),
                        credentialId = p.optString("credential_id"),
                        signCount = p.optLong("sign_count", 0L),
                        transports = transports,
                        label = p.optString("label", ""),
                        createdAt = p.optString("created_at", ""),
                        updatedAt = p.optString("updated_at", "")
                    )
                )
            }
            PasskeyListResult(success = true, passkeys = list)
        } catch (e: Exception) {
            PasskeyListResult(success = false, error = e.message)
        }
    }

    /**
     * DELETE /api/v1/master-wallet/:id/passkey/credentials/:credId — remove a
     * passkey credential from the backend. Backend returns 204 on success.
     */
    suspend fun deletePasskey(masterId: String, credId: String): Boolean = withContext(Dispatchers.IO) {
        apiDelete("/api/v1/master-wallet/$masterId/passkey/credentials/$credId")
    }

    /**
     * POST /api/v1/master-wallet/:id/passkey/verify-assertion — server-side
     * verification of a WebAuthn assertion. All fields are base64url. The backend
     * performs the real P-256 ECDSA verification; this method only reports its
     * verdict (never fabricates success).
     */
    suspend fun verifyPasskeyAssertion(
        masterId: String,
        credentialId: String,
        authData: String,
        clientDataJson: String,
        signature: String
    ): PasskeyVerifyResult = withContext(Dispatchers.IO) {
        try {
            val body = JSONObject()
                .put("credential_id", credentialId)
                .put("authenticator_data", authData)
                .put("client_data_json", clientDataJson)
                .put("signature", signature)
                .toString()
            val resp = apiPost("/api/v1/master-wallet/$masterId/passkey/verify-assertion", body)
                ?: return@withContext PasskeyVerifyResult(success = false, error = "Assertion verification request failed")
            val json = JSONObject(resp)
            PasskeyVerifyResult(
                success = true,
                verified = json.optBoolean("verified", false),
                credentialId = json.optString("credential_id", credentialId)
            )
        } catch (e: Exception) {
            PasskeyVerifyResult(success = false, error = e.message)
        }
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

    private fun apiPut(endpoint: String, body: String): String? {
        val token = try { requireToken() } catch (e: Exception) { return null }
        return try {
            val conn = (URL("$baseUrl$endpoint").openConnection() as HttpURLConnection).apply {
                requestMethod = "PUT"
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

data class UpdateResult(
    val success: Boolean,
    val id: String = "",
    val updated: Boolean = false,
    val error: String? = null
)

data class TransactionDetailResult(
    val success: Boolean,
    val transaction: String = "",
    val error: String? = null
)

data class MultisigWalletDetail(
    val id: String,
    val name: String,
    val owners: List<String>,
    val threshold: Int,
    val chainId: Long,
    val address: String,
    val pendingTransactions: List<String>
)

data class MultisigWalletDetailResult(
    val success: Boolean,
    val wallet: MultisigWalletDetail? = null,
    val error: String? = null
)

/**
 * Passkey credential as returned by the backend
 * (GET /passkey/credentials). credentialId is base64url; signCount is the
 * authenticator counter; createdAt/updatedAt are backend timestamps.
 */
data class PasskeyCredential(
    val id: String,
    val credentialId: String,
    val signCount: Long,
    val transports: List<String>,
    val label: String,
    val createdAt: String,
    val updatedAt: String
)

data class PasskeyRegisterResult(
    val success: Boolean,
    val passkeyId: String = "",
    val credentialId: String = "",
    val registered: Boolean = false,
    val error: String? = null
)

data class PasskeyListResult(
    val success: Boolean,
    val passkeys: List<PasskeyCredential> = emptyList(),
    val error: String? = null
)

data class PasskeyVerifyResult(
    val success: Boolean,
    val verified: Boolean = false,
    val credentialId: String = "",
    val error: String? = null
)
