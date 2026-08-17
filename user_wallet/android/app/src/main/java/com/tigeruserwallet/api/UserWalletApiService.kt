package com.tigeruserwallet.api

import android.content.Context
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import android.util.Base64
import org.json.JSONArray
import org.json.JSONObject
import java.io.IOException
import java.net.URLEncoder
import java.util.concurrent.TimeUnit

/**
 * TigerWallet UserWallet API Service — Android.
 *
 * Talks to the canonical TigerWallet Go wallet-api backend (go/wallet_api,
 * port 8443): REAL on-chain RPC, REAL BIP-39/32/44 HD derivation, REAL
 * secp256k1 signing + broadcast, AES-256-GCM encrypted-seed persistence
 * (PostgreSQL + Redis). No stubs, no fabricated data.
 *
 * Kotlin suspend-function API consumed by the fragments' CoroutineScope.
 */
object UserWalletApiService {

    private const val DEFAULT_BASE_URL = "http://localhost:8443/api/v1"
    private const val PREFS = "userwallet_prefs"
    private const val TOKEN_KEY = "userwallet_token"

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
        authToken = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .getString(TOKEN_KEY, null)
    }

    fun setToken(token: String?) {
        authToken = token
        appContext?.getSharedPreferences(PREFS, Context.MODE_PRIVATE)?.edit()?.apply {
            if (token != null) putString(TOKEN_KEY, token) else remove(TOKEN_KEY)
            apply()
        }
    }

    fun isAuthenticated(): Boolean = authToken != null

    private fun headers(): okhttp3.Headers.Builder = okhttp3.Headers.Builder()
        .add("Content-Type", "application/json")
        .add("Accept", "application/json")
        .also { authToken?.let { h -> add("Authorization", "Bearer $it") } }

    private fun requestBuilder(path: String): Request.Builder =
        Request.Builder().url("$baseUrl$path").headers(headers())

    private fun errorFromResponse(response: okhttp3.Response): String {
        return try {
            val json = JSONObject(response.body?.string() ?: "{}")
            json.optString("error", "Request failed: ${response.code}")
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

    // ==================== Auth ====================

    data class AuthResult(val token: String, val userId: String?)

    fun login(email: String, password: String): AuthResult {
        val body = JSONObject().put("email", email).put("password", password).toString()
        val req = requestBuilder("/auth/login").post(body.toRequestBody(jsonMediaType)).build()
        val json = execute(req)
        val token = json.getString("token")
        setToken(token)
        return AuthResult(token, json.optString("user_id", null))
    }

    fun register(email: String, password: String): AuthResult {
        // Canonical /auth/register accepts {email, password} only (see route table).
        val body = JSONObject().put("email", email).put("password", password).toString()
        val req = requestBuilder("/auth/register").post(body.toRequestBody(jsonMediaType)).build()
        val json = execute(req)
        val token = json.getString("token")
        setToken(token)
        return AuthResult(token, json.optString("user_id", null))
    }

    // POST /auth/guest { device_id } -> { user_id, token, guest: true }. Public
    // (no auth required). Provisions an anonymous guest account so the user can
    // Create/Import a wallet without registering. The token is persisted exactly
    // like login (setToken -> SharedPreferences TOKEN_KEY).
    data class GuestAuthResult(val token: String, val userId: String?, val guest: Boolean)

    fun guestAuth(deviceId: String): GuestAuthResult {
        val body = JSONObject().put("device_id", deviceId).toString()
        val req = requestBuilder("/auth/guest").post(body.toRequestBody(jsonMediaType)).build()
        val json = execute(req)
        val token = json.getString("token")
        setToken(token)
        return GuestAuthResult(
            token = token,
            userId = json.optString("user_id", null),
            guest = if (json.has("guest")) json.optBoolean("guest", true) else true
        )
    }

    fun logout() {
        setToken(null)
    }

    // ==================== Wallets ====================

    data class Wallet(
        val id: String,
        val label: String,
        val chainId: Int,
        val address: String,
        val derivationPath: String,
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
                derivationPath = it.optString("derivation_path"),
                mnemonic = it.optString("mnemonic", null)
            )
        }
    }

    fun createWallet(label: String, password: String, chainId: Int, mnemonic: String? = null): Wallet {
        val body = JSONObject().apply {
            put("label", label)
            put("password", password)
            put("chain_id", chainId)
            if (mnemonic != null) put("mnemonic", mnemonic) else put("entropy_bits", 256)
        }.toString()
        val req = requestBuilder("/wallets").post(body.toRequestBody(jsonMediaType)).build()
        val json = execute(req)
        return Wallet(
            id = json.optString("id"),
            label = json.optString("label"),
            chainId = json.optInt("chain_id"),
            address = json.optString("address"),
            derivationPath = json.optString("derivation_path"),
            mnemonic = json.optString("mnemonic", null)
        )
    }

    // ==================== Balances (real eth_getBalance via backend) ====================

    data class Balance(
        val chainId: Int,
        val symbol: String,
        val address: String,
        val balance: String,
        val balanceF: Double,
        val usdValue: Double
    )

    fun getBalances(): List<Balance> {
        val wallets = getWallets()
        return wallets.mapNotNull { w ->
            try {
                fetchBalance(w.address, w.chainId)
            } catch (e: Exception) {
                null
            }
        }
    }

    fun fetchBalance(address: String, chainId: Int): Balance {
        val path = "/balance?address=$address&chain_id=$chainId"
        val req = requestBuilder(path).get().build()
        client.newCall(req).execute().use { response ->
            if (!response.isSuccessful) throw httpException(response)
            val json = JSONObject(response.body?.string() ?: "{}")
            return Balance(
                chainId = json.optInt("chain_id"),
                symbol = json.optString("symbol"),
                address = json.optString("address"),
                balance = json.optString("balance"),
                balanceF = json.optDouble("balance_f"),
                usdValue = json.optDouble("usd_value")
            )
        }
    }

    // ==================== Transactions (real Etherscan via backend) ====================

    data class Transaction(
        val hash: String,
        val from: String,
        val to: String,
        val value: String,
        val timeStamp: String,
        val isError: String
    )

    fun getTransactions(address: String? = null, chainId: Int = 1): List<Transaction> {
        val addr = address ?: getWallets().firstOrNull()?.address ?: ""
        if (addr.isEmpty()) return emptyList()
        val path = "/transactions?address=$addr&chain_id=$chainId"
        val req = requestBuilder(path).get().build()
        return executeList(req, "transactions").map {
            Transaction(
                hash = it.optString("hash"),
                from = it.optString("from"),
                to = it.optString("to"),
                value = it.optString("value"),
                timeStamp = it.optString("timeStamp"),
                isError = it.optString("isError", "0")
            )
        }
    }

    // GET /transactions/:txHash?chain_id=N -> { status, block_number?, confirmations? }.
    // Transaction-status proxy (explorer receipt lookup).
    data class TransactionStatus(
        val status: String,
        val blockNumber: Long?,
        val confirmations: Long?
    )

    fun getTransactionStatus(txHash: String, chainId: Int = 1): TransactionStatus {
        val path = "/transactions/${txHash}?chain_id=$chainId"
        val req = requestBuilder(path).get().build()
        val json = execute(req)
        return TransactionStatus(
            status = json.optString("status"),
            blockNumber = if (json.has("block_number")) json.optLong("block_number") else null,
            confirmations = if (json.has("confirmations")) json.optLong("confirmations") else null
        )
    }

    // ==================== Send / Sign (real on-chain) ====================

    data class SendResult(val txHash: String, val rawTx: String, val nonce: Long)

    fun sendTransaction(walletId: String, password: String, to: String, value: String, chainId: Int = 1): SendResult {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("to", to)
            put("value", value)
            put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/send").post(body.toRequestBody(jsonMediaType)).build()
        val json = execute(req)
        return SendResult(
            txHash = json.optString("tx_hash"),
            rawTx = json.optString("raw_tx"),
            nonce = json.optLong("nonce")
        )
    }

    // POST /auto-send with the SAME body as /send, plus optional
    // ?master_wallet_id=<id> query. Same Bearer JWT auth as /send. Returns the
    // existing send response PLUS { auto_approved, auto_approval_reason }.
    data class AutoSendResult(
        val txHash: String,
        val rawTx: String,
        val nonce: Long,
        val autoApproved: Boolean,
        val autoApprovalReason: String
    )

    fun autoSendTransaction(
        walletId: String,
        password: String,
        to: String,
        value: String,
        chainId: Int = 1,
        masterWalletId: String? = null
    ): AutoSendResult {
        val path = if (masterWalletId != null) {
            "/auto-send?master_wallet_id=${masterWalletId}"
        } else {
            "/auto-send"
        }
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("to", to)
            put("value", value)
            put("chain_id", chainId)
        }.toString()
        val req = requestBuilder(path).post(body.toRequestBody(jsonMediaType)).build()
        val json = execute(req)
        return AutoSendResult(
            txHash = json.optString("tx_hash"),
            rawTx = json.optString("raw_tx"),
            nonce = json.optLong("nonce"),
            autoApproved = json.optBoolean("auto_approved", false),
            autoApprovalReason = json.optString("auto_approval_reason", "")
        )
    }

    fun signMessage(walletId: String, password: String, message: String): String {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("message", message)
        }.toString()
        val req = requestBuilder("/sign").post(body.toRequestBody(jsonMediaType)).build()
        val json = execute(req)
        return json.optString("signature")
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
        // Mirrors the web client: derive connected/chain-id from /chains (no
        // dedicated status route on wallet_api). block_number is not exposed by
        // the chains list endpoint, so we report 0 honestly rather than fabricate.
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
            priceImpact = json.optDouble("price_impact"),
            route = json.optString("route")
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
            priceImpact = json.optDouble("price_impact"),
            route = json.optString("route"),
        )
    }

    // ==================== Profile (local JWT decode) ====================

    data class Profile(val id: String, val email: String, val username: String)

    fun getProfile(): Profile {
        val token = authToken ?: throw Exception("Not authenticated")
        val parts = token.split(".")
        if (parts.size < 2) throw Exception("Not authenticated")
        val payload = String(Base64.decode(parts[1], Base64.DEFAULT))
        val json = JSONObject(payload)
        return Profile(
            id = json.optString("user_id", json.optString("sub", "")),
            email = json.optString("email", ""),
            username = json.optString("username", json.optString("name", ""))
        )
    }

    // ==================== Health (outside /api/v1) ====================

    fun health(): JSONObject {
        val base = baseUrl.removeSuffix("/api/v1")
        val req = Request.Builder().url("$base/health").headers(headers()).get().build()
        return execute(req)
    }

    // ==================== Import wallet (POST /wallets) ====================

    fun importWallet(
        label: String,
        password: String,
        mnemonic: String,
        chainId: Int,
        passphrase: String? = null
    ): Wallet {
        val body = JSONObject().apply {
            put("label", label)
            put("password", password)
            put("chain_id", chainId)
            put("mnemonic", mnemonic)
            if (passphrase != null) put("passphrase", passphrase)
        }.toString()
        val req = requestBuilder("/wallets").post(body.toRequestBody(jsonMediaType)).build()
        val json = execute(req)
        return Wallet(
            id = json.optString("id"),
            label = json.optString("label"),
            chainId = json.optInt("chain_id"),
            address = json.optString("address"),
            derivationPath = json.optString("derivation_path"),
            mnemonic = json.optString("mnemonic", null)
        )
    }

    // ==================== NFT transfer (POST /nft/transfer) ====================

    fun transferNFT(
        walletId: String,
        password: String,
        to: String,
        tokenId: String,
        contractAddress: String,
        chainId: Int
    ): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("to", to)
            put("token_id", tokenId)
            put("contract_address", contractAddress)
            put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/nft/transfer").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    // ==================== Transaction receipt (GET /transactions/{txHash}) ====================

    fun getTransactionReceipt(txHash: String, chainId: Int): JSONObject {
        val path = "/transactions/${txHash}?chain_id=$chainId"
        val req = requestBuilder(path).get().build()
        return execute(req)
    }

    // ==================== Gas estimate (POST /gas/estimate) ====================

    fun estimateGas(
        from: String,
        to: String,
        value: String? = null,
        data: String? = null,
        chainId: Int
    ): JSONObject {
        val body = JSONObject().apply {
            put("from", from)
            put("to", to)
            if (value != null) put("value", value)
            if (data != null) put("data", data)
            put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/gas/estimate").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    // ==================== Swap execute (POST /swap/execute) ====================

    fun executeSwap(
        walletId: String,
        password: String,
        fromToken: String,
        toToken: String,
        fromAmount: String,
        chainId: Int
    ): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("from_token", fromToken)
            put("to_token", toToken)
            put("from_amount", fromAmount)
            put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/swap/execute").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    // ==================== AMM (GET /amm/quote, POST /amm/swap) ====================

    fun getAmmQuote(fromToken: String, toToken: String, fromAmount: String, chainId: Int): SwapQuote {
        val path = "/amm/quote?from_token=$fromToken&to_token=$toToken&from_amount=$fromAmount&chain_id=$chainId"
        val req = requestBuilder(path).get().build()
        val json = execute(req)
        return SwapQuote(
            fromToken = json.optString("from_token"),
            toToken = json.optString("to_token"),
            fromAmount = json.optString("from_amount"),
            toAmount = json.optString("to_amount"),
            priceImpact = json.optDouble("price_impact"),
            route = json.optString("route")
        )
    }

    fun ammSwap(
        walletId: String,
        password: String,
        fromToken: String,
        toToken: String,
        fromAmount: String,
        chainId: Int
    ): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("from_token", fromToken)
            put("to_token", toToken)
            put("from_amount", fromAmount)
            put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/amm/swap").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    // ==================== Staking actions (POST /staking/{stake,unstake,claim}) ====================

    fun stake(walletId: String, password: String, asset: String, amount: String, chainId: Int): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("asset", asset)
            put("amount", amount)
            put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/staking/stake").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun unstake(walletId: String, password: String, asset: String, amount: String, chainId: Int): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("asset", asset)
            put("amount", amount)
            put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/staking/unstake").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun claim(walletId: String, password: String, asset: String, chainId: Int): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("asset", asset)
            put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/staking/claim").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    // ==================== Networks (alias of getChains) ====================

    fun getNetworks(): List<ChainInfo> = getChains()

    // ==================== Non-EVM (POST /non_evm/{address,sign,send}) ====================

    fun nonEvmAddress(seed: String, chainType: String, chainId: String, path: String? = null): JSONObject {
        val body = JSONObject().apply {
            put("seed", seed)
            put("chain_type", chainType)
            put("chain_id", chainId)
            if (path != null) put("path", path)
        }.toString()
        val req = requestBuilder("/non_evm/address").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun nonEvmSign(seed: String, chainType: String, chainId: String, messageHash: String, path: String? = null): JSONObject {
        val body = JSONObject().apply {
            put("seed", seed)
            put("chain_type", chainType)
            put("chain_id", chainId)
            put("message_hash", messageHash)
            if (path != null) put("path", path)
        }.toString()
        val req = requestBuilder("/non_evm/sign").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun nonEvmSend(
        seed: String,
        chainType: String,
        chainId: String,
        to: String,
        value: String,
        path: String? = null
    ): JSONObject {
        val body = JSONObject().apply {
            put("seed", seed)
            put("chain_type", chainType)
            put("chain_id", chainId)
            put("to", to)
            put("value", value)
            if (path != null) put("path", path)
        }.toString()
        val req = requestBuilder("/non_evm/send").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    // ==================== Address book (GET/POST/PUT/DELETE /address-book/contacts) ====================

    fun getAddressBookContacts(): List<JSONObject> =
        executeList(requestBuilder("/address-book/contacts").get().build(), "contacts")

    fun addContact(name: String, address: String, chainId: Int? = null): JSONObject {
        val body = JSONObject().apply {
            put("name", name)
            put("address", address)
            if (chainId != null) put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/address-book/contacts").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun updateContact(id: String, name: String? = null, address: String? = null, chainId: Int? = null): JSONObject {
        val body = JSONObject().apply {
            if (name != null) put("name", name)
            if (address != null) put("address", address)
            if (chainId != null) put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/address-book/contacts/${id}").put(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun deleteContact(id: String): JSONObject {
        val req = requestBuilder("/address-book/contacts/${id}").delete().build()
        return execute(req)
    }

    // ==================== Devices (GET/POST /devices, sync + delete) ====================

    fun getDevices(): List<JSONObject> =
        executeList(requestBuilder("/devices").get().build(), "devices")

    fun registerDevice(name: String, deviceType: String): JSONObject {
        val body = JSONObject().apply {
            put("name", name)
            put("device_type", deviceType)
        }.toString()
        val req = requestBuilder("/devices").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun syncDevice(deviceId: String): JSONObject {
        val req = requestBuilder("/devices/${deviceId}/sync").post("".toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun deleteDevice(deviceId: String): JSONObject {
        val req = requestBuilder("/devices/${deviceId}").delete().build()
        return execute(req)
    }

    // ==================== Approvals (GET /approvals, DELETE /approvals/{id}) ====================

    fun getApprovals(address: String, chainId: Int): List<JSONObject> {
        val path = "/approvals?address=${URLEncoder.encode(address, "UTF-8")}&chain_id=$chainId"
        val req = requestBuilder(path).get().build()
        return executeList(req, "approvals")
    }

    fun revokeApproval(approvalId: String): JSONObject {
        val req = requestBuilder("/approvals/${approvalId}").delete().build()
        return execute(req)
    }

    // ==================== Keystore (POST /keystore/{export,import}) ====================

    fun exportKeystore(walletId: String, password: String): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
        }.toString()
        val req = requestBuilder("/keystore/export").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun importKeystore(keystore: String, password: String, label: String? = null): Wallet {
        val body = JSONObject().apply {
            put("keystore", keystore)
            put("password", password)
            if (label != null) put("label", label)
        }.toString()
        val req = requestBuilder("/keystore/import").post(body.toRequestBody(jsonMediaType)).build()
        val json = execute(req)
        return Wallet(
            id = json.optString("id"),
            label = json.optString("label"),
            chainId = json.optInt("chain_id"),
            address = json.optString("address"),
            derivationPath = json.optString("derivation_path"),
            mnemonic = json.optString("mnemonic", null)
        )
    }

    // ==================== Encrypted seed export/import (POST /wallets...) ====================

    fun exportEncryptedSeed(walletId: String, password: String): JSONObject {
        val body = JSONObject().put("password", password).toString()
        val req = requestBuilder("/wallets/${walletId}/export-encrypted-seed")
            .post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun importEncryptedSeed(encryptedSeed: String, password: String, label: String? = null): Wallet {
        val body = JSONObject().apply {
            put("encrypted_seed", encryptedSeed)
            put("password", password)
            if (label != null) put("label", label)
        }.toString()
        val req = requestBuilder("/wallets/import-encrypted-seed")
            .post(body.toRequestBody(jsonMediaType)).build()
        val json = execute(req)
        return Wallet(
            id = json.optString("id"),
            label = json.optString("label"),
            chainId = json.optInt("chain_id"),
            address = json.optString("address"),
            derivationPath = json.optString("derivation_path"),
            mnemonic = json.optString("mnemonic", null)
        )
    }

    // ==================== Security (GET /security/check-*, POST /security/scan) ====================

    fun checkUrl(url: String): JSONObject {
        val path = "/security/check-url?url=${URLEncoder.encode(url, "UTF-8")}"
        val req = requestBuilder(path).get().build()
        return execute(req)
    }

    fun checkAddress(address: String): JSONObject {
        val path = "/security/check-address?address=${URLEncoder.encode(address, "UTF-8")}"
        val req = requestBuilder(path).get().build()
        return execute(req)
    }

    fun securityScan(target: String): JSONObject {
        val body = JSONObject().put("target", target).toString()
        val req = requestBuilder("/security/scan").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    // ==================== Lending (GET /lending/markets,positions; POST /lending/{supply,borrow,withdraw,repay}) ====================

    fun getLendingMarkets(): List<JSONObject> =
        executeList(requestBuilder("/lending/markets").get().build(), "markets")

    fun getLendingPositions(): List<JSONObject> =
        executeList(requestBuilder("/lending/positions").get().build(), "positions")

    fun lendingSupply(walletId: String, password: String, asset: String, amount: String, chainId: Int): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("asset", asset)
            put("amount", amount)
            put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/lending/supply").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun lendingBorrow(walletId: String, password: String, asset: String, amount: String, chainId: Int): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("asset", asset)
            put("amount", amount)
            put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/lending/borrow").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun lendingWithdraw(walletId: String, password: String, asset: String, amount: String, chainId: Int): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("asset", asset)
            put("amount", amount)
            put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/lending/withdraw").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun lendingRepay(walletId: String, password: String, asset: String, amount: String, chainId: Int): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("asset", asset)
            put("amount", amount)
            put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/lending/repay").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    // ==================== Copy trading (GET /copytrading/{traders,signals}; follow + stop) ====================

    fun getCopyTraders(): List<JSONObject> =
        executeList(requestBuilder("/copytrading/traders").get().build(), "traders")

    fun followTrader(traderId: String, allocation: String? = null): JSONObject {
        val body = JSONObject().apply {
            put("trader_id", traderId)
            if (allocation != null) put("allocation", allocation)
        }.toString()
        val req = requestBuilder("/copytrading/follow").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun stopCopyTrader(copierId: String): JSONObject {
        val req = requestBuilder("/copytrading/copiers/${copierId}/stop")
            .post("".toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun getCopySignals(): List<JSONObject> =
        executeList(requestBuilder("/copytrading/signals").get().build(), "signals")

    // ==================== DAO (GET /dao/{proposals,delegates}; create + vote) ====================

    fun getDaoProposals(): List<JSONObject> =
        executeList(requestBuilder("/dao/proposals").get().build(), "proposals")

    fun createDaoProposal(title: String, description: String): JSONObject {
        val body = JSONObject().apply {
            put("title", title)
            put("description", description)
        }.toString()
        val req = requestBuilder("/dao/proposals").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun voteDaoProposal(proposalId: String, support: Boolean): JSONObject {
        val body = JSONObject().put("support", support).toString()
        val req = requestBuilder("/dao/proposals/${proposalId}/vote")
            .post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun getDaoDelegates(): List<JSONObject> =
        executeList(requestBuilder("/dao/delegates").get().build(), "delegates")

    // ==================== Perpetual (GET /perpetual/positions; create + close) ====================

    fun getPerpetualPositions(): List<JSONObject> =
        executeList(requestBuilder("/perpetual/positions").get().build(), "positions")

    fun createPerpetualPosition(pair: String, side: String, size: String, leverage: Int, chainId: Int): JSONObject {
        val body = JSONObject().apply {
            put("pair", pair)
            put("side", side)
            put("size", size)
            put("leverage", leverage)
            put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/perpetual/positions").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun closePerpetualPosition(positionId: String): JSONObject {
        val req = requestBuilder("/perpetual/positions/${positionId}/close")
            .post("".toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    // ==================== Margin (GET /margin/positions; create + close) ====================

    fun getMarginPositions(): List<JSONObject> =
        executeList(requestBuilder("/margin/positions").get().build(), "positions")

    fun createMarginPosition(pair: String, side: String, size: String, leverage: Int, chainId: Int): JSONObject {
        val body = JSONObject().apply {
            put("pair", pair)
            put("side", side)
            put("size", size)
            put("leverage", leverage)
            put("chain_id", chainId)
        }.toString()
        val req = requestBuilder("/margin/positions").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun closeMarginPosition(positionId: String): JSONObject {
        val req = requestBuilder("/margin/positions/${positionId}/close")
            .post("".toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    // ==================== Prediction (GET /prediction/markets; bet) ====================

    fun getPredictionMarkets(): List<JSONObject> =
        executeList(requestBuilder("/prediction/markets").get().build(), "markets")

    fun placePredictionBet(marketId: String, side: String, amount: String): JSONObject {
        val body = JSONObject().apply {
            put("side", side)
            put("amount", amount)
        }.toString()
        val req = requestBuilder("/prediction/markets/${marketId}/bet")
            .post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    // ==================== Launchpool (GET /launchpool, /launchpool/stakes; stake + unstake) ====================

    fun getLaunchpool(): JSONObject =
        execute(requestBuilder("/launchpool").get().build())

    fun getLaunchpoolStakes(): List<JSONObject> =
        executeList(requestBuilder("/launchpool/stakes").get().build(), "stakes")

    fun launchpoolStake(walletId: String, password: String, amount: String): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("amount", amount)
        }.toString()
        val req = requestBuilder("/launchpool/stake").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    fun launchpoolUnstake(walletId: String, password: String, amount: String): JSONObject {
        val body = JSONObject().apply {
            put("wallet_id", walletId)
            put("password", password)
            put("amount", amount)
        }.toString()
        val req = requestBuilder("/launchpool/unstake").post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    // ==================== Token sales (GET /token-sales; participate) ====================

    fun getTokenSales(): List<JSONObject> =
        executeList(requestBuilder("/token-sales").get().build(), "sales")

    fun participateTokenSale(saleId: String, amount: String): JSONObject {
        val body = JSONObject().put("amount", amount).toString()
        val req = requestBuilder("/token-sales/${saleId}/participate")
            .post(body.toRequestBody(jsonMediaType)).build()
        return execute(req)
    }

    // ==================== Dapps (GET /dapps, /dapps/categories) ====================

    fun getDapps(): List<JSONObject> =
        executeList(requestBuilder("/dapps").get().build(), "dapps")

    fun getDappCategories(): List<JSONObject> =
        executeList(requestBuilder("/dapps/categories").get().build(), "categories")

    // ==================== Chart / DeFi (GET /chart/history, /defi/protocols) ====================

    fun getChartHistory(token: String, days: Int? = null): JSONObject {
        val path = if (days != null) {
            "/chart/history?token=${URLEncoder.encode(token, "UTF-8")}&days=$days"
        } else {
            "/chart/history?token=${URLEncoder.encode(token, "UTF-8")}"
        }
        val req = requestBuilder(path).get().build()
        return execute(req)
    }

    fun getDefiProtocols(): List<JSONObject> =
        executeList(requestBuilder("/defi/protocols").get().build(), "protocols")

    // ==================== Payment URI parser (bare 0x, ethereum:, EIP-681) ====================

    data class PaymentUri(val address: String, val amount: String?, val chainId: Int?)

    fun parsePaymentUri(input: String): PaymentUri? {
        val trimmed = input.trim()
        if (trimmed.isEmpty()) return null
        // Bare 0x address (and possibly ?value=... appended).
        if (trimmed.startsWith("0x", ignoreCase = true)) {
            val (addr, params) = splitUri(trimmed)
            if (!isValidAddress(addr)) return null
            val amount = params["value"] ?: params["amount"]
            return PaymentUri(addr, amount, params["chainId"]?.toIntOrNull())
        }
        // ethereum:<address> or ethereum:<address>?... (EIP-681) or ethereum:/<address>
        if (trimmed.startsWith("ethereum:", ignoreCase = true)) {
            val rest = trimmed.substring("ethereum:".length).trim()
            val cleaned = if (rest.startsWith("/")) rest.removePrefix("/") else rest
            if (cleaned.startsWith("@")) {
                // EIP-681 chain-tagged form: ethereum:<chainId>@<address>?...
                val atIdx = cleaned.indexOf('@')
                val chainPart = cleaned.substring(1, atIdx)
                val remainder = cleaned.substring(atIdx + 1)
                val (addr, params) = splitUri(remainder)
                if (!isValidAddress(addr)) return null
                val amount = params["value"] ?: params["amount"]
                return PaymentUri(addr, amount, chainPart.toIntOrNull() ?: params["chainId"]?.toIntOrNull())
            }
            val (addr, params) = splitUri(cleaned)
            if (!isValidAddress(addr)) return null
            val amount = params["value"] ?: params["amount"]
            return PaymentUri(addr, amount, params["chainId"]?.toIntOrNull())
        }
        return null
    }

    private fun splitUri(s: String): Pair<String, Map<String, String>> {
        val qIdx = s.indexOf('?')
        if (qIdx < 0) return s to emptyMap()
        val addr = s.substring(0, qIdx)
        val query = s.substring(qIdx + 1)
        val params = mutableMapOf<String, String>()
        for (pair in query.split('&')) {
            val eq = pair.indexOf('=')
            if (eq > 0) {
                val k = pair.substring(0, eq)
                val v = pair.substring(eq + 1)
                params[k] = v
            }
        }
        return addr to params
    }

    private fun isValidAddress(s: String): Boolean {
        val a = s.trim()
        if (!a.startsWith("0x", ignoreCase = true)) return false
        val hex = a.substring(2)
        return hex.length == 40 && hex.all { it.isLetterOrDigit() }
    }
}
