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

    /**
     * POST /api/v1/master-wallet/:id/withdrawal-request — two-party gate.
     * Body {to_address, amount_wei, currency?, chain_id?} → {withdrawal_id, status}.
     */
    suspend fun requestWithdrawal(
        masterId: String,
        toAddress: String,
        amountWei: String,
        currency: String?,
        chainId: Long?
    ): WithdrawalRequestResult? = withContext(Dispatchers.IO) {
        try {
            val body = JSONObject()
                .put("to_address", toAddress)
                .put("amount_wei", amountWei)
                .apply {
                    currency?.let { put("currency", it) }
                    chainId?.let { put("chain_id", it) }
                }
                .toString()
            val resp = apiPost("/api/v1/master-wallet/$masterId/withdrawal-request", body)
                ?: return@withContext null
            val json = JSONObject(resp)
            WithdrawalRequestResult(
                withdrawalId = json.optString("withdrawal_id", ""),
                status = json.optString("status", "")
            )
        } catch (e: Exception) {
            null
        }
    }

    /**
     * POST /api/v1/master-wallet/:id/revenue-payout — executes a two-party
     * gate payout. Body {to, amount, password, gas_limit?, withdrawal_id}
     * → {transaction_hash, status, withdrawal_id?, from?, chain_id?}.
     */
    suspend fun revenuePayout(
        masterId: String,
        to: String,
        amount: String,
        password: String,
        gasLimit: Long?,
        withdrawalId: String
    ): RevenuePayoutResult? = withContext(Dispatchers.IO) {
        try {
            val body = JSONObject()
                .put("to", to)
                .put("amount", amount)
                .put("password", password)
                .put("withdrawal_id", withdrawalId)
                .apply { gasLimit?.let { put("gas_limit", it) } }
                .toString()
            val resp = apiPost("/api/v1/master-wallet/$masterId/revenue-payout", body)
                ?: return@withContext null
            val json = JSONObject(resp)
            RevenuePayoutResult(
                transactionHash = json.optString("transaction_hash", ""),
                status = json.optString("status", ""),
                withdrawalId = json.optString("withdrawal_id", withdrawalId),
                from = json.optString("from", ""),
                chainId = json.optLong("chain_id", 0L)
            )
        } catch (e: Exception) {
            null
        }
    }

    // MARK: - Sub Wallets (mirror iOS: GET/POST /master-wallet/:id/sub-wallets,
    //   GET .../sub-wallets/:sid/balance, POST .../sub-wallets/:sid/transfer)

    suspend fun getSubWallets(masterWalletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$masterWalletId/sub-wallets") }

    suspend fun createSubWallet(
        masterWalletId: String,
        name: String,
        password: String,
        chainId: Int
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("name", name)
            .put("password", password)
            .put("chain_id", chainId)
            .toString()
        apiPost("/api/v1/master-wallet/$masterWalletId/sub-wallets", body)
    }

    suspend fun getSubWalletBalance(masterWalletId: String, subWalletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$masterWalletId/sub-wallets/$subWalletId/balance") }

    suspend fun transferSubWallet(
        masterWalletId: String,
        subWalletId: String,
        to: String,
        amount: String,
        password: String,
        token: String? = null
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("to", to)
            .put("amount", amount)
            .put("password", password)
            .apply { token?.let { put("token", it) } }
            .toString()
        apiPost("/api/v1/master-wallet/$masterWalletId/sub-wallets/$subWalletId/transfer", body)
    }

    // MARK: - Transactions (getTransaction already exists)

    suspend fun listTransactions(walletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$walletId/transactions") }

    suspend fun createTransaction(
        walletId: String,
        to: String,
        amount: String,
        password: String,
        token: String? = null
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("to", to)
            .put("amount", amount)
            .put("password", password)
            .apply { token?.let { put("token", it) } }
            .toString()
        apiPost("/api/v1/master-wallet/$walletId/transactions", body)
    }

    suspend fun approveTransaction(walletId: String, transactionId: String): String? =
        withContext(Dispatchers.IO) { apiPost("/api/v1/master-wallet/$walletId/transactions/$transactionId/approve", "{}") }

    suspend fun rejectTransaction(walletId: String, transactionId: String): String? =
        withContext(Dispatchers.IO) { apiPost("/api/v1/master-wallet/$walletId/transactions/$transactionId/reject", "{}") }

    // MARK: - Policies (GET/POST /policies, PUT/DELETE /policies/:pid)

    suspend fun getPolicies(walletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$walletId/policies") }

    suspend fun createPolicy(
        walletId: String,
        name: String,
        policyType: String,
        conditions: Map<String, String>? = null,
        actions: Map<String, String>? = null,
        isActive: Boolean? = null,
        priority: Int = 0
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("name", name)
            .put("policy_type", policyType)
            .apply {
                conditions?.let { put("conditions", JSONObject(it)) }
                actions?.let { put("actions", JSONObject(it)) }
                isActive?.let { put("is_active", it) }
            }
            .put("priority", priority)
            .toString()
        apiPost("/api/v1/master-wallet/$walletId/policies", body)
    }

    suspend fun updatePolicy(
        walletId: String,
        policyId: String,
        name: String? = null,
        policyType: String? = null,
        conditions: Map<String, String>? = null,
        actions: Map<String, String>? = null,
        isActive: Boolean? = null,
        priority: Int? = null
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .apply {
                name?.let { put("name", it) }
                policyType?.let { put("policy_type", it) }
                conditions?.let { put("conditions", JSONObject(it)) }
                actions?.let { put("actions", JSONObject(it)) }
                isActive?.let { put("is_active", it) }
                priority?.let { put("priority", it) }
            }
            .toString()
        apiPut("/api/v1/master-wallet/$walletId/policies/$policyId", body)
    }

    suspend fun deletePolicy(walletId: String, policyId: String): Boolean =
        withContext(Dispatchers.IO) { apiDelete("/api/v1/master-wallet/$walletId/policies/$policyId") }

    // MARK: - Fees (GET/POST /fees, DELETE /fees/:fid)

    suspend fun getFees(walletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$walletId/fees") }

    suspend fun createFee(
        walletId: String,
        feeType: String,
        feePercentage: Double? = null,
        feeFixed: String? = null,
        isActive: Boolean? = null
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("fee_type", feeType)
            .apply {
                feePercentage?.let { put("fee_percentage", it) }
                feeFixed?.let { put("fee_fixed", it) }
                isActive?.let { put("is_active", it) }
            }
            .toString()
        apiPost("/api/v1/master-wallet/$walletId/fees", body)
    }

    suspend fun deleteFee(walletId: String, feeId: String): Boolean =
        withContext(Dispatchers.IO) { apiDelete("/api/v1/master-wallet/$walletId/fees/$feeId") }

    // MARK: - Auto-sign (master): GET/POST /auto-sign, DELETE /auto-sign/:rid,
    //   POST /auto-sign-transaction, GET /auto-sign-logs

    suspend fun getAutoSignRules(walletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$walletId/auto-sign") }

    suspend fun createAutoSignRule(
        walletId: String,
        name: String,
        ruleType: String,
        conditions: Map<String, String>? = null,
        maxAmount: String? = null,
        isActive: Boolean? = null
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("name", name)
            .put("rule_type", ruleType)
            .apply {
                conditions?.let { put("conditions", JSONObject(it)) }
                maxAmount?.let { put("max_amount", it) }
                isActive?.let { put("is_active", it) }
            }
            .toString()
        apiPost("/api/v1/master-wallet/$walletId/auto-sign", body)
    }

    suspend fun deleteAutoSignRule(walletId: String, ruleId: String): Boolean =
        withContext(Dispatchers.IO) { apiDelete("/api/v1/master-wallet/$walletId/auto-sign/$ruleId") }

    /**
     * POST /master-wallet/:id/auto-sign-transaction — backend performs the real
     * secp256k1 signing + broadcast. Body mirrors the backend AutoSignRequest.
     */
    suspend fun autoSignTransaction(
        id: String,
        mnemonic: String,
        chainId: Long,
        chainType: String,
        txType: String,
        toAddress: String,
        value: String,
        tokenAddress: String? = null,
        derivationPath: String? = null,
        accountIndex: Long? = null,
        contractAddress: String? = null,
        data: String? = null,
        withdrawalId: String? = null
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("mnemonic", mnemonic)
            .put("chain_id", chainId)
            .put("chain_type", chainType)
            .put("tx_type", txType)
            .put("to_address", toAddress)
            .put("value", value)
            .apply {
                tokenAddress?.let { put("token_address", it) }
                derivationPath?.let { put("derivation_path", it) }
                accountIndex?.let { put("account_index", it) }
                contractAddress?.let { put("contract_address", it) }
                data?.let { put("data", it) }
                withdrawalId?.let { put("withdrawal_id", it) }
            }
            .toString()
        apiPost("/api/v1/master-wallet/$id/auto-sign-transaction", body)
    }

    suspend fun listAutoSignLogs(id: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$id/auto-sign-logs") }

    // MARK: - Users (GET/POST /users, DELETE /users/:uid)

    suspend fun getUsers(walletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$walletId/users") }

    suspend fun createUser(
        walletId: String,
        email: String,
        password: String,
        name: String? = null,
        role: String? = null
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("email", email)
            .put("password", password)
            .apply {
                name?.let { put("name", it) }
                role?.let { put("role", it) }
            }
            .toString()
        apiPost("/api/v1/master-wallet/$walletId/users", body)
    }

    suspend fun deleteUser(walletId: String, userId: String): Boolean =
        withContext(Dispatchers.IO) { apiDelete("/api/v1/master-wallet/$walletId/users/$userId") }

    // MARK: - Audit (GET /audit)

    suspend fun getAudit(walletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$walletId/audit") }

    // MARK: - Analytics (GET /analytics/volume, /transactions, /wallets)

    suspend fun getAnalyticsVolume(walletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$walletId/analytics/volume") }

    suspend fun getAnalyticsTransactions(walletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$walletId/analytics/transactions") }

    suspend fun getAnalyticsWallets(walletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$walletId/analytics/wallets") }

    // MARK: - Notifications (GET/POST /notifications)

    suspend fun getNotifications(walletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$walletId/notifications") }

    suspend fun createNotification(
        walletId: String,
        type: String,
        title: String,
        message: String,
        userId: String? = null,
        category: String? = null,
        priority: String? = null,
        channel: String? = null,
        data: Map<String, String>? = null
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("notification_type", type)
            .put("title", title)
            .put("message", message)
            .apply {
                userId?.let { put("user_id", it) }
                category?.let { put("category", it) }
                priority?.let { put("priority", it) }
                channel?.let { put("channel", it) }
                data?.let { put("data", JSONObject(it)) }
            }
            .toString()
        apiPost("/api/v1/master-wallet/$walletId/notifications", body)
    }

    // MARK: - Webhooks (GET/POST /webhooks, DELETE /webhooks/:wid)

    suspend fun getWebhooks(walletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$walletId/webhooks") }

    suspend fun createWebhook(
        walletId: String,
        name: String,
        url: String,
        events: List<String>,
        retryCount: Int = 0
    ): String? = withContext(Dispatchers.IO) {
        val eventsArr = JSONArray()
        events.forEach { eventsArr.put(it) }
        val body = JSONObject()
            .put("name", name)
            .put("url", url)
            .put("events", eventsArr)
            .put("retry_count", retryCount)
            .toString()
        apiPost("/api/v1/master-wallet/$walletId/webhooks", body)
    }

    suspend fun deleteWebhook(walletId: String, webhookId: String): Boolean =
        withContext(Dispatchers.IO) { apiDelete("/api/v1/master-wallet/$walletId/webhooks/$webhookId") }

    // MARK: - Treasury (GET /treasury, GET /treasury/transactions,
    //   POST /treasury/transfer, POST /treasury/sweep)

    suspend fun getTreasury(walletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$walletId/treasury") }

    suspend fun getTreasuryTransactions(walletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$walletId/treasury/transactions") }

    /** POST .../treasury/transfer — body mirrors iOS TreasuryTransferRequest {to, amount, password}. */
    suspend fun treasuryTransfer(walletId: String, to: String, amount: String, password: String): String? =
        withContext(Dispatchers.IO) {
            val body = JSONObject()
                .put("to", to)
                .put("amount", amount)
                .put("password", password)
                .toString()
            apiPost("/api/v1/master-wallet/$walletId/treasury/transfer", body)
        }

    /** POST .../treasury/sweep — body mirrors iOS TreasurySweepRequest {to, password}. */
    suspend fun treasurySweep(walletId: String, to: String, password: String): String? =
        withContext(Dispatchers.IO) {
            val body = JSONObject()
                .put("to", to)
                .put("password", password)
                .toString()
            apiPost("/api/v1/master-wallet/$walletId/treasury/sweep", body)
        }

    // MARK: - Multisig (getMultisigWalletDetail already exists)

    suspend fun getMultisigWallets(walletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$walletId/multisig/wallets") }

    suspend fun createMultisigWallet(
        walletId: String,
        name: String,
        owners: List<String>,
        threshold: Int
    ): String? = withContext(Dispatchers.IO) {
        val ownersArr = JSONArray()
        owners.forEach { ownersArr.put(it) }
        val body = JSONObject()
            .put("name", name)
            .put("owners", ownersArr)
            .put("threshold", threshold)
            .toString()
        apiPost("/api/v1/master-wallet/$walletId/multisig/wallets", body)
    }

    suspend fun getMultisigTransactions(walletId: String, multisigWalletId: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$walletId/multisig/wallets/$multisigWalletId/transactions") }

    suspend fun createMultisigTransaction(
        walletId: String,
        multisigWalletId: String,
        to: String,
        amount: String,
        data: String? = null
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("to", to)
            .put("amount", amount)
            .apply { data?.let { put("data", it) } }
            .toString()
        apiPost("/api/v1/master-wallet/$walletId/multisig/wallets/$multisigWalletId/transactions", body)
    }

    suspend fun signMultisigTransaction(walletId: String, transactionId: String): String? =
        withContext(Dispatchers.IO) { apiPost("/api/v1/master-wallet/$walletId/multisig/transactions/$transactionId/sign", "{}") }

    suspend fun executeMultisigTransaction(walletId: String, transactionId: String): String? =
        withContext(Dispatchers.IO) { apiPost("/api/v1/master-wallet/$walletId/multisig/transactions/$transactionId/execute", "{}") }

    // MARK: - User-wallet management / governance (EVM + non-EVM chains, tokens,
    //   derive-user-address, user-wallet-addresses)

    suspend fun listUserEVMChains(id: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$id/user-chains/evm") }

    suspend fun addUserEVMChain(
        id: String,
        chainId: Int,
        name: String,
        symbol: String,
        rpcUrl: String,
        explorerUrl: String,
        decimals: Int,
        derivationPath: String
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("chain_id", chainId)
            .put("name", name)
            .put("symbol", symbol)
            .put("rpc_url", rpcUrl)
            .put("explorer_url", explorerUrl)
            .put("decimals", decimals)
            .put("derivation_path", derivationPath)
            .toString()
        apiPost("/api/v1/master-wallet/$id/user-chains/evm", body)
    }

    suspend fun updateUserEVMChain(
        id: String,
        chainId: Int,
        name: String? = null,
        symbol: String? = null,
        rpcUrl: String? = null,
        explorerUrl: String? = null,
        decimals: Int? = null,
        derivationPath: String? = null
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .apply {
                name?.let { put("name", it) }
                symbol?.let { put("symbol", it) }
                rpcUrl?.let { put("rpc_url", it) }
                explorerUrl?.let { put("explorer_url", it) }
                decimals?.let { put("decimals", it) }
                derivationPath?.let { put("derivation_path", it) }
            }
            .toString()
        apiPut("/api/v1/master-wallet/$id/user-chains/evm/$chainId", body)
    }

    suspend fun removeUserEVMChain(id: String, chainId: Int): Boolean =
        withContext(Dispatchers.IO) { apiDelete("/api/v1/master-wallet/$id/user-chains/evm/$chainId") }

    suspend fun listUserNonEVMChains(id: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$id/user-chains/nonevm") }

    suspend fun addUserNonEVMChain(
        id: String,
        chainId: Int,
        name: String,
        symbol: String,
        chainType: String,
        rpcUrl: String,
        derivationPath: String,
        addressPrefix: String
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("chain_id", chainId)
            .put("name", name)
            .put("symbol", symbol)
            .put("chain_type", chainType)
            .put("rpc_url", rpcUrl)
            .put("derivation_path", derivationPath)
            .put("address_prefix", addressPrefix)
            .toString()
        apiPost("/api/v1/master-wallet/$id/user-chains/nonevm", body)
    }

    suspend fun updateUserNonEVMChain(
        id: String,
        chainId: Int,
        name: String? = null,
        symbol: String? = null,
        chainType: String? = null,
        rpcUrl: String? = null,
        derivationPath: String? = null,
        addressPrefix: String? = null
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .apply {
                name?.let { put("name", it) }
                symbol?.let { put("symbol", it) }
                chainType?.let { put("chain_type", it) }
                rpcUrl?.let { put("rpc_url", it) }
                derivationPath?.let { put("derivation_path", it) }
                addressPrefix?.let { put("address_prefix", it) }
            }
            .toString()
        apiPut("/api/v1/master-wallet/$id/user-chains/nonevm/$chainId", body)
    }

    suspend fun removeUserNonEVMChain(id: String, chainId: Int): Boolean =
        withContext(Dispatchers.IO) { apiDelete("/api/v1/master-wallet/$id/user-chains/nonevm/$chainId") }

    suspend fun listUserTokens(id: String, chainId: Int? = null): String? =
        withContext(Dispatchers.IO) {
            val endpoint = if (chainId != null) "/api/v1/master-wallet/$id/user-tokens?chain_id=$chainId"
                else "/api/v1/master-wallet/$id/user-tokens"
            apiGet(endpoint)
        }

    suspend fun addUserToken(
        id: String,
        chainId: Int,
        contractAddress: String,
        symbol: String,
        name: String,
        decimals: Int,
        isNative: Boolean
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("chain_id", chainId)
            .put("contract_address", contractAddress)
            .put("symbol", symbol)
            .put("name", name)
            .put("decimals", decimals)
            .put("is_native", isNative)
            .toString()
        apiPost("/api/v1/master-wallet/$id/user-tokens", body)
    }

    suspend fun updateUserToken(
        id: String,
        tokenId: String,
        symbol: String? = null,
        name: String? = null,
        decimals: Int? = null,
        isNative: Boolean? = null
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .apply {
                symbol?.let { put("symbol", it) }
                name?.let { put("name", it) }
                decimals?.let { put("decimals", it) }
                isNative?.let { put("is_native", it) }
            }
            .toString()
        apiPut("/api/v1/master-wallet/$id/user-tokens/$tokenId", body)
    }

    suspend fun removeUserToken(id: String, tokenId: String): Boolean =
        withContext(Dispatchers.IO) { apiDelete("/api/v1/master-wallet/$id/user-tokens/$tokenId") }

    suspend fun deriveUserAddress(
        id: String,
        mnemonic: String,
        chainId: Long,
        chainType: String,
        derivationPath: String,
        accountIndex: Long
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("mnemonic", mnemonic)
            .put("chain_id", chainId)
            .put("chain_type", chainType)
            .put("derivation_path", derivationPath)
            .put("account_index", accountIndex)
            .toString()
        apiPost("/api/v1/master-wallet/$id/derive-user-address", body)
    }

    suspend fun listUserWalletAddresses(id: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$id/user-wallet-addresses") }

    // MARK: - Feature flags (GET/POST /feature-flags, PUT/DELETE .../feature-flags/:flagId)

    suspend fun listFeatureFlags(id: String): String? =
        withContext(Dispatchers.IO) { apiGet("/api/v1/master-wallet/$id/feature-flags") }

    suspend fun addFeatureFlag(
        id: String,
        flagKey: String,
        flagValue: String,
        description: String? = null,
        isEnabled: Boolean
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("flag_key", flagKey)
            .put("flag_value", flagValue)
            .put("is_enabled", isEnabled)
            .apply { description?.let { put("description", it) } }
            .toString()
        apiPost("/api/v1/master-wallet/$id/feature-flags", body)
    }

    suspend fun updateFeatureFlag(
        id: String,
        flagId: String,
        flagValue: String? = null,
        description: String? = null,
        isEnabled: Boolean? = null
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .apply {
                flagValue?.let { put("flag_value", it) }
                description?.let { put("description", it) }
                isEnabled?.let { put("is_enabled", it) }
            }
            .toString()
        apiPut("/api/v1/master-wallet/$id/feature-flags/$flagId", body)
    }

    suspend fun removeFeatureFlag(id: String, flagId: String): Boolean =
        withContext(Dispatchers.IO) { apiDelete("/api/v1/master-wallet/$id/feature-flags/$flagId") }

    // MARK: - Auto-sign bridge (POST /user-wallet-auto-sign, POST /check-auto-sign-policy)

    /**
     * POST /master-wallet/:id/user-wallet-auto-sign — the user-wallet auto-sign
     * bridge. Body mirrors the backend AutoSignRequest (real signing on the server).
     */
    suspend fun userWalletAutoSign(
        id: String,
        mnemonic: String,
        chainId: Long,
        chainType: String,
        txType: String,
        toAddress: String,
        value: String,
        tokenAddress: String? = null,
        derivationPath: String? = null,
        accountIndex: Long? = null,
        contractAddress: String? = null,
        data: String? = null,
        withdrawalId: String? = null
    ): String? = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("mnemonic", mnemonic)
            .put("chain_id", chainId)
            .put("chain_type", chainType)
            .put("tx_type", txType)
            .put("to_address", toAddress)
            .put("value", value)
            .apply {
                tokenAddress?.let { put("token_address", it) }
                derivationPath?.let { put("derivation_path", it) }
                accountIndex?.let { put("account_index", it) }
                contractAddress?.let { put("contract_address", it) }
                data?.let { put("data", it) }
                withdrawalId?.let { put("withdrawal_id", it) }
            }
            .toString()
        apiPost("/api/v1/master-wallet/$id/user-wallet-auto-sign", body)
    }

    /** POST /master-wallet/:id/check-auto-sign-policy — body {tx_type, value}. */
    suspend fun checkAutoSignPolicy(id: String, txType: String, value: String): String? =
        withContext(Dispatchers.IO) {
            val body = JSONObject()
                .put("tx_type", txType)
                .put("value", value)
                .toString()
            apiPost("/api/v1/master-wallet/$id/check-auto-sign-policy", body)
        }

    // MARK: - Public endpoints (NO auth) — getSupportedChains already exists.
    //   /gas, /price, /transactions/history, /health do not require a JWT.

    suspend fun getGas(chainId: Int): String? =
        withContext(Dispatchers.IO) { apiGetPublic("/api/v1/gas?chain_id=$chainId") }

    suspend fun getPrice(coinId: String): String? =
        withContext(Dispatchers.IO) { apiGetPublic("/api/v1/price?coin_id=$coinId") }

    suspend fun getHealth(): String? =
        withContext(Dispatchers.IO) { apiGetPublic("/api/v1/health") }

    suspend fun getTransactionHistory(address: String, chainId: Int): String? =
        withContext(Dispatchers.IO) { apiGetPublic("/api/v1/transactions/history?address=$address&chain_id=$chainId") }

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

    // Public GET (no JWT) for /api/v1/chains, /gas, /price, /transactions/history, /health.
    private fun apiGetPublic(endpoint: String): String? = try {
        val conn = (URL("$baseUrl$endpoint").openConnection() as HttpURLConnection).apply {
            requestMethod = "GET"
            connectTimeout = 10000
            readTimeout = 10000
        }
        if (conn.responseCode in 200..299) conn.inputStream.bufferedReader().readText() else null
    } catch (e: Exception) { null }
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

data class WithdrawalRequestResult(
    val withdrawalId: String,
    val status: String
)

data class RevenuePayoutResult(
    val transactionHash: String,
    val status: String,
    val withdrawalId: String,
    val from: String,
    val chainId: Long
)
