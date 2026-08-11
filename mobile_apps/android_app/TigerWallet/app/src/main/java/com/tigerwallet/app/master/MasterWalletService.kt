/**
 * TigerWallet Android - Master Wallet Service
 * Master wallet OWNS and CONTROLS all user wallets
 *
 * Fail-closed: wallet `address` and `publicKey` are NEVER fabricated. Wallet
 * creation delegates to the REAL backend (go/wallet_api at
 * BACKEND_BASE_URL, POST /api/v1/wallets), which derives a real secp256k1
 * address from a real BIP-39 mnemonic. The previous implementation returned
 * `"0x" + UUID.randomUUID().replace("-","").take(40)` as the address and
 * `take(130)` as the public key — that fabrication is removed. If the backend
 * is unreachable or rejects the request, or no JWT is configured, creation
 * throws fail-closed (no wallet with a fake address/key is ever persisted).
 */

package com.tigerwallet.app.master

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.util.concurrent.TimeUnit

/**
 * Master Wallet Service - Owner of All User Wallets
 */
class MasterWalletService private constructor() {

    companion object {
        val instance: MasterWalletService by lazy { MasterWalletService() }

        /** Real backend base URL (go/wallet_api). Wallet creation is delegated
         *  here so the returned address is a real secp256k1 address, never a
         *  fabricated `"0x" + UUID`. */
        const val BACKEND_BASE_URL = "http://localhost:8443"

        private val JSON_MEDIA_TYPE = "application/json".toMediaType()
    }

    /** JWT used to authenticate backend wallet-creation. Must be set by the
     *  host app before createMasterWallet/createUserWallet can create a wallet.
     *  When empty, creation fails closed. */
    @Volatile
    var jwt: String = ""

    private val client = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS).build()

    // Master wallet is the OWNER of all user wallets
    private val _masterWalletsFlow = MutableStateFlow<List<MasterWallet>>(emptyList())
    val masterWalletsFlow: StateFlow<List<MasterWallet>> = _masterWalletsFlow.asStateFlow()

    // All user wallets owned by master wallet
    private val _userWalletsFlow = MutableStateFlow<List<UserWallet>>(emptyList())
    val userWalletsFlow: StateFlow<List<UserWallet>> = _userWalletsFlow.asStateFlow()

    private val _networksFlow = MutableStateFlow<List<BlockchainNetwork>>(emptyList())
    val networksFlow: StateFlow<List<BlockchainNetwork>> = _networksFlow.asStateFlow()

    private val _tokensFlow = MutableStateFlow<List<CryptoToken>>(emptyList())
    val tokensFlow: StateFlow<List<CryptoToken>> = _tokensFlow.asStateFlow()

    fun initialize() {
        loadMasterWallets()
        loadUserWallets()
        loadNetworks()
        loadTokens()
    }

    // ============================================================================
    // MASTER WALLET - The Owner
    // ============================================================================

    private fun loadMasterWallets() {
        val data = KeychainManager.load("master_wallets")
        if (data != null) {
            try {
                val wallets = SerializationUtils.deserialize<List<MasterWallet>>(data)
                _masterWalletsFlow.value = wallets
            } catch (e: Exception) { _masterWalletsFlow.value = emptyList() }
        }
    }

    /**
     * Create a master wallet. The address is derived by the REAL backend
     * (POST /api/v1/wallets, real secp256k1 from a real BIP-39 mnemonic) and is
     * never fabricated. `publicKey` is left empty (the backend wallet record
     * exposes only the address; a full public key must come from a real
     * secp256k1 derivation path, never a UUID). Throws fail-closed if the
     * backend is unreachable/rejects the request or no JWT is configured.
     */
    suspend fun createMasterWallet(name: String, type: WalletType, blockchain: String): MasterWallet =
        withContext(Dispatchers.IO) {
            val backend = createWalletViaBackend(name, blockchain)
            val wallet = MasterWallet(
                id = backend.id,
                name = name,
                type = type,
                blockchain = blockchain,
                address = backend.address,
                publicKey = "",
                balance = 0.0,
                isActive = true,
                createdAt = System.currentTimeMillis()
            )
            saveMasterWallet(wallet)
            _masterWalletsFlow.value = _masterWalletsFlow.value + wallet
            wallet
        }

    private fun saveMasterWallet(wallet: MasterWallet) {
        val wallets = _masterWalletsFlow.value.toMutableList()
        wallets.removeAll { it.id == wallet.id }
        wallets.add(wallet)
        val data = SerializationUtils.serialize(wallets)
        KeychainManager.save("master_wallets", data)
    }

    fun getMasterWallets(): List<MasterWallet> = _masterWalletsFlow.value

    // ============================================================================
    // USER WALLETS - Owned by Master Wallet
    // ============================================================================

    private fun loadUserWallets() {
        val data = KeychainManager.load("user_wallets")
        if (data != null) {
            try {
                val wallets = SerializationUtils.deserialize<List<UserWallet>>(data)
                _userWalletsFlow.value = wallets
            } catch (e: Exception) { _userWalletsFlow.value = emptyList() }
        }
    }

    // MASTER WALLET creates/owns user wallets
    suspend fun createUserWallet(masterWalletId: String, userId: String, blockchain: String): UserWallet =
        withContext(Dispatchers.IO) {
            val masterWallet = _masterWalletsFlow.value.find { it.id == masterWalletId }
                ?: throw Exception("Master wallet not found")

            val backend = createWalletViaBackend(masterWallet.name, blockchain)
            val userWallet = UserWallet(
                id = backend.id,
                userId = userId,
                ownerMasterWalletId = masterWalletId,  // OWNERSHIP
                ownerAddress = masterWallet.address,     // Master wallet address owns this
                blockchain = blockchain,
                address = backend.address,
                publicKey = "",
                balance = 0.0,
                isActive = true,
                createdAt = System.currentTimeMillis()
            )

            saveUserWallet(userWallet)
            _userWalletsFlow.value = _userWalletsFlow.value + userWallet
            userWallet
        }

    // MASTER WALLET can control any user wallet
    fun controlUserWallet(masterWalletId: String, userWalletId: String): UserWallet? {
        return _userWalletsFlow.value.find { 
            it.id == userWalletId && it.ownerMasterWalletId == masterWalletId 
        }
    }

    // MASTER WALLET approves all transactions from user wallets
    fun approveTransaction(masterWalletId: String, userWalletId: String, txHash: String): Boolean {
        val userWallet = controlUserWallet(masterWalletId, userWalletId)
        return userWallet != null  // Master wallet approves
    }

    private fun saveUserWallet(wallet: UserWallet) {
        val wallets = _userWalletsFlow.value.toMutableList()
        wallets.removeAll { it.id == wallet.id }
        wallets.add(wallet)
        val data = SerializationUtils.serialize(wallets)
        KeychainManager.save("user_wallets", data)
        _userWalletsFlow.value = wallets
    }

    // Get all user wallets owned by a master wallet
    fun getUserWallets(masterWalletId: String): List<UserWallet> {
        return _userWalletsFlow.value.filter { it.ownerMasterWalletId == masterWalletId }
    }

    // Get all user wallets for a user
    fun getUserWalletsByUser(userId: String): List<UserWallet> {
        return _userWalletsFlow.value.filter { it.userId == userId }
    }

    // ============================================================================
    // Networks & Tokens (same as before)
    // ============================================================================

    private fun loadNetworks() {
        _networksFlow.value = getDefaultNetworks()
    }

    private fun loadTokens() {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val tokens = fetchTokensFromAPI()
                if (tokens.isNotEmpty()) _tokensFlow.value = tokens
            } catch (e: Exception) { _tokensFlow.value = emptyList() }
        }
    }

    private suspend fun fetchTokensFromAPI(): List<CryptoToken> = withContext(Dispatchers.IO) {
        try {
            val request = Request.Builder()
                .url("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=500&page=1&sparkline=false")
                .get().build()
            val response = client.newCall(request).execute()
            if (response.isSuccessful) {
                val body = response.body?.string() ?: return@withContext emptyList()
                val jsonArray = JSONArray(body)
                val tokens = mutableListOf<CryptoToken>()
                for (i in 0 until jsonArray.length()) {
                    val coin = jsonArray.getJSONObject(i)
                    tokens.add(CryptoToken(
                        id = coin.getString("id"),
                        symbol = coin.getString("symbol").uppercase(),
                        name = coin.getString("name"),
                        image = coin.optString("image", ""),
                        currentPrice = coin.optDouble("current_price", 0.0),
                        marketCap = coin.optLong("market_cap", 0L),
                        rank = coin.optInt("market_cap_rank", 0),
                        priceChange24h = coin.optDouble("price_change_24h", 0.0)
                    ))
                }
                tokens
            } else emptyList()
        } catch (e: Exception) { emptyList() }
    }

    fun getDefaultNetworks(): List<BlockchainNetwork> = listOf(
        BlockchainNetwork("ethereum", "Ethereum", "ETH", 1, "https://eth.llamarpc.com", true),
        BlockchainNetwork("polygon", "Polygon", "MATIC", 137, "https://polygon-rpc.com", true),
        BlockchainNetwork("bsc", "BNB Chain", "BNB", 56, "https://bsc-dataseed.binance.org", true),
        BlockchainNetwork("arbitrum", "Arbitrum", "ETH", 42161, "https://arb1.arbitrum.io/rpc", true),
        BlockchainNetwork("optimism", "Optimism", "ETH", 10, "https://mainnet.optimism.io", true),
        BlockchainNetwork("avalanche", "Avalanche", "AVAX", 43114, "https://api.avax.network/ext/bc/C/rpc", true),
        BlockchainNetwork("base", "Base", "ETH", 8453, "https://mainnet.base.org", true),
        BlockchainNetwork("solana", "Solana", "SOL", 0, "https://api.mainnet-beta.solana.com", false),
        BlockchainNetwork("tron", "Tron", "TRX", 0, "https://api.trongrid.io", false),
        BlockchainNetwork("bitcoin", "Bitcoin", "BTC", 0, "https://blockstream.info/api", false)
    )

    fun addNetwork(network: BlockchainNetwork) {
        val updated = _networksFlow.value.toMutableList()
        if (updated.none { it.id == network.id }) { updated.add(network); _networksFlow.value = updated }
    }

    fun addToken(token: CryptoToken) {
        val updated = _tokensFlow.value.toMutableList()
        if (updated.none { it.id == token.id }) { updated.add(token); _tokensFlow.value = updated }
    }

    fun getNetworks(): List<BlockchainNetwork> = _networksFlow.value
    fun getTokens(): List<CryptoToken> = _tokensFlow.value

    /**
     * Password used to encrypt the backend wallet seed (backend requires
     * `password` min 8 chars). Must be set by the host app from user input
     * before createMasterWallet/createUserWallet can create a wallet. When
     * empty, creation fails closed.
     */
    @Volatile
    var creationPassword: String = ""

    /** Result of a real backend wallet creation. `address` is a real
     *  secp256k1 address derived by go/wallet_api from a real BIP-39 mnemonic. */
    private data class BackendWallet(val id: String, val address: String)

    private fun chainIdFor(blockchain: String): Long =
        getDefaultNetworks().firstOrNull { it.id == blockchain }?.chainId
            ?: getDefaultNetworks().firstOrNull { it.id == "ethereum" }?.chainId
            ?: 1L

    /**
     * Creates a wallet on the REAL backend (POST /api/v1/wallets) and returns
     * the real secp256k1 address + wallet id. Never fabricates an address or
     * public key. Throws fail-closed if no JWT/password is configured, the
     * backend is unreachable, or it rejects the request.
     */
    private fun createWalletViaBackend(label: String, blockchain: String): BackendWallet {
        if (jwt.isEmpty()) {
            throw IllegalStateException(
                "No auth JWT configured; cannot create wallet on backend (fail-closed)."
            )
        }
        if (creationPassword.length < 8) {
            throw IllegalStateException(
                "Wallet creation requires a user password (>=8 chars) for backend seed encryption (fail-closed)."
            )
        }
        val body = JSONObject().apply {
            put("label", label)
            put("chain_id", chainIdFor(blockchain))
            put("password", creationPassword)
            put("entropy_bits", 256)
        }
        val request = Request.Builder()
            .url("$BACKEND_BASE_URL/api/v1/wallets")
            .header("Authorization", "Bearer $jwt")
            .header("Content-Type", "application/json")
            .post(body.toString().toRequestBody(JSON_MEDIA_TYPE))
            .build()
        val respData: String
        val statusCode: Int
        try {
            client.newCall(request).execute().use { resp ->
                statusCode = resp.code
                respData = resp.body?.string() ?: ""
            }
        } catch (e: Exception) {
            throw IllegalStateException("Backend unreachable: ${e.message ?: e.toString()}")
        }
        if (statusCode != 201) {
            throw IllegalStateException("Backend rejected wallet creation (HTTP $statusCode): $respData")
        }
        val json: JSONObject
        try {
            json = JSONObject(respData)
        } catch (e: Exception) {
            throw IllegalStateException("Malformed backend response: $respData")
        }
        val id = json.optString("id", "")
        val address = json.optString("address", "")
        if (id.isEmpty() || !address.startsWith("0x") || address.length != 42) {
            throw IllegalStateException("Backend did not return a valid wallet address: $respData")
        }
        return BackendWallet(id, address)
    }
}

// ============================================================================
// Data Models
// ============================================================================

enum class WalletType { HOT, COLD, OPERATIONS }

data class MasterWallet(
    val id: String,
    val name: String,
    val type: WalletType,
    val blockchain: String,
    val address: String,
    val publicKey: String,
    val balance: Double,
    val isActive: Boolean,
    val createdAt: Long
)

// User wallet OWNED by master wallet
data class UserWallet(
    val id: String,
    val userId: String,
    val ownerMasterWalletId: String,   // WHO OWNS THIS WALLET
    val ownerAddress: String,          // Master wallet address
    val blockchain: String,
    val address: String,
    val publicKey: String,
    val balance: Double,
    val isActive: Boolean,
    val createdAt: Long
)

data class BlockchainNetwork(val id: String, val name: String, val symbol: String, val chainId: Long, val rpcUrl: String, val isEVM: Boolean)
data class CryptoToken(val id: String, val symbol: String, val name: String, val image: String, val currentPrice: Double, val marketCap: Long, val rank: Int, val priceChange24h: Double)

// Utilities
object KeychainManager {
    private val prefs = android.content.Context.getSharedPreferences("tigerwallet", android.content.Context.MODE_PRIVATE)
    fun save(key: String, data: ByteArray) { prefs.edit().putString(key, android.util.Base64.encodeToString(data, android.util.Base64.NO_WRAP)).apply() }
    fun load(key: String): ByteArray? { val encoded = prefs.getString(key, null) ?: return null; return android.util.Base64.decode(encoded, android.util.Base64.NO_WRAP) }
}

object SerializationUtils {
    private val gson = com.google.gson.Gson()
    fun serialize(obj: Any): ByteArray = gson.toJson(obj).toByteArray()
    inline fun <reified T> deserialize(data: ByteArray): T = gson.fromJson(String(data), T::class.java)
}
