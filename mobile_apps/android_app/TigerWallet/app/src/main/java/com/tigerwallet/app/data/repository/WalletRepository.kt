package com.tigerwallet.app.data.repository

import android.content.Context
import android.content.SharedPreferences
import com.tigerwallet.app.data.models.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.withContext

class WalletRepository(private val context: Context) {
    
    private val prefs: SharedPreferences = context.getSharedPreferences("tigerwallet", Context.MODE_PRIVATE)
    
    private val _wallets = MutableStateFlow<List<Wallet>>(emptyList())
    val wallets: StateFlow<List<Wallet>> = _wallets.asStateFlow()
    
    private val _currentWallet = MutableStateFlow<Wallet?>(null)
    val currentWallet: StateFlow<Wallet?> = _currentWallet.asStateFlow()
    
    private val _tokens = MutableStateFlow<List<TokenBalance>>(emptyList())
    val tokens: StateFlow<List<TokenBalance>> = _tokens.asStateFlow()
    
    private val _transactions = MutableStateFlow<List<Transaction>>(emptyList())
    val transactions: StateFlow<List<Transaction>> = _transactions.asStateFlow()
    
    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()
    
    // Real token data from API
    private var cachedTokenPrices: List<TokenData> = emptyList()
    
    init {
        loadWallets()
        // Load real token prices from API
        kotlinx.coroutines.GlobalScope.launch {
            cachedTokenPrices = fetchTokenPrices()
        }
    }
    
    private suspend fun fetchTokenPrices(): List<TokenData> {
        return try {
            val client = okhttp3.OkHttpClient()
            val request = okhttp3.Request.Builder()
                .url("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=500&page=1&sparkline=false")
                .get()
                .build()
            
            val response = client.newCall(request).execute()
            if (response.isSuccessful) {
                val body = response.body?.string() ?: return emptyList()
                val jsonArray = org.json.JSONArray(body)
                val tokens = mutableListOf<TokenData>()
                for (i in 0 until jsonArray.length()) {
                    val coin = jsonArray.getJSONObject(i)
                    tokens.add(
                        TokenData(
                            id = coin.getString("id"),
                            symbol = coin.getString("symbol").uppercase(),
                            name = coin.getString("name"),
                            image = coin.optString("image", ""),
                            currentPrice = coin.optDouble("current_price", 0.0),
                            marketCap = coin.optLong("market_cap", 0L),
                            marketCapRank = coin.optInt("market_cap_rank", 0),
                            totalVolume = coin.optLong("total_volume", 0L),
                            priceChange24h = coin.optDouble("price_change_24h", 0.0),
                            priceChangePercentage24h = coin.optDouble("price_change_percentage_24h", 0.0),
                            circulatingSupply = coin.optDouble("circulating_supply", 0.0),
                            totalSupply = coin.optDouble("total_supply", 0.0),
                            ath = coin.optDouble("ath", 0.0),
                            athChangePercentage = coin.optDouble("ath_change_percentage", 0.0),
                            atl = coin.optDouble("atl", 0.0),
                            atlChangePercentage = coin.optDouble("atl_change_percentage", 0.0)
                        )
                    )
                }
                tokens
            } else {
                emptyList()
            }
        } catch (e: Exception) {
            e.printStackTrace()
            emptyList()
        }
    }
    
    suspend fun createWallet(name: String, password: String, mnemonic: List<String>? = null): Wallet {
        val walletService = ServiceLocator.walletService
        val words = mnemonic ?: walletService.generateMnemonic(password)
        val (address, publicKey) = walletService.deriveWalletAddress(words)
        
        val wallet = Wallet(
            name = name,
            address = address,
            publicKey = publicKey
        )
        
        // Generate addresses for all supported chains
        for (network in NetworkRepository.getDefaultNetworks()) {
            if (network.isEVM) {
                wallet.chainAddresses[network.chainId.toString()] = address
            }
        }
        
        _wallets.value = _wallets.value + wallet
        _currentWallet.value = wallet
        saveWallets()
        
        return wallet
    }
    
    suspend fun importWallet(mnemonic: List<String>, name: String, password: String): Wallet? {
        if (!validateMnemonic(mnemonic)) return null
        return createWallet(name, password, mnemonic)
    }
    
    fun deleteWallet(wallet: Wallet) {
        _wallets.value = _wallets.value.filter { it.id != wallet.id }
        if (_currentWallet.value?.id == wallet.id) {
            _currentWallet.value = _wallets.value.firstOrNull()
        }
        saveWallets()
    }
    
    fun selectWallet(wallet: Wallet) {
        _currentWallet.value = wallet
    }
    
    suspend fun fetchBalances() = withContext(Dispatchers.IO) {
        val wallet = _currentWallet.value ?: return@withContext
        _isLoading.value = true
        
        val blockchainService = ServiceLocator.blockchainService
        val allTokens = mutableListOf<TokenBalance>()
        
        for ((chainIdStr, address) in wallet.chainAddresses) {
            val chainId = chainIdStr.toLongOrNull() ?: continue
            
            // Get native token balance
            val balance = blockchainService.getBalance(address, chainId)
            val token = TokenBalance(
                id = "${chainId}_native",
                symbol = getChainSymbol(chainId),
                name = getChainName(chainId),
                address = null,
                decimals = getChainDecimals(chainId),
                balance = balance,
                price = getTokenPrice(getChainSymbol(chainId)),
                chainId = chainId
            )
            allTokens.add(token)
        }
        
        _tokens.value = allTokens
        _isLoading.value = false
    }
    
    suspend fun sendTransaction(
        to: String,
        amount: Double,
        chainId: Long,
        tokenAddress: String? = null
    ): String {
        val wallet = _currentWallet.value ?: throw Exception("No wallet selected")
        val fromAddress = wallet.chainAddresses[chainId.toString()] 
            ?: throw Exception("No address for chain")
        
        val walletService = ServiceLocator.walletService
        val signedTx = walletService.buildAndSignTransaction(fromAddress, to, amount, chainId, tokenAddress)
        
        val blockchainService = ServiceLocator.blockchainService
        val txHash = blockchainService.broadcastTransaction(signedTx, chainId)
        
        val transaction = Transaction(
            hash = txHash,
            from = fromAddress,
            to = to,
            amount = amount,
            symbol = getChainSymbol(chainId),
            decimals = getChainDecimals(chainId),
            chainId = chainId,
            status = TransactionStatus.PENDING,
            timestamp = System.currentTimeMillis(),
            type = TransactionType.SEND
        )
        
        _transactions.value = listOf(transaction) + _transactions.value
        
        return txHash
    }
    
    private fun validateMnemonic(mnemonic: List<String>): Boolean {
        return mnemonic.size == 12 || mnemonic.size == 24
    }
    
    private fun saveWallets() {
        // Simplified - in production use proper serialization
        val walletIds = _wallets.value.map { it.id }
        prefs.edit().putStringSet("wallet_ids", walletIds.toSet()).apply()
    }
    
    private fun loadWallets() {
        // Simplified - in production load from database
    }
    
    private fun getChainSymbol(chainId: Long): String = when (chainId) {
        1L -> "ETH"
        56L -> "BNB"
        137L -> "MATIC"
        42161L -> "ETH"
        10L -> "ETH"
        43114L -> "AVAX"
        0L -> "SOL"
        195L -> "TRX"
        else -> "UNKNOWN"
    }
    
    private fun getChainName(chainId: Long): String = when (chainId) {
        1L -> "Ethereum"
        56L -> "BNB Smart Chain"
        137L -> "Polygon"
        42161L -> "Arbitrum"
        10L -> "Optimism"
        43114L -> "Avalanche"
        0L -> "Solana"
        195L -> "Tron"
        else -> "Unknown"
    }
    
    private fun getChainDecimals(chainId: Long): Int = when (chainId) {
        0L -> 9  // Solana
        195L -> 6 // Tron
        else -> 18
    }
    
    private fun getTokenPrice(symbol: String): Double {
        // Use real API data from CoinGecko
        return cachedTokenPrices.find { it.symbol == symbol.uppercase() }?.currentPrice ?: 0.0
    }
    
    // Get all 500+ real tokens
    fun getAllTokens(): List<TokenData> = cachedTokenPrices
    
    // Get token by symbol
    fun getTokenBySymbol(symbol: String): TokenData? {
        return cachedTokenPrices.find { it.symbol == symbol.uppercase() }
    }
    
    // Get tokens by market cap (top tokens)
    fun getTopTokens(limit: Int = 100): List<TokenData> {
        return cachedTokenPrices.sortedByDescending { it.marketCap }.take(limit)
    }
}

// Network Repository - Uses Real 103+ Networks
object NetworkRepository {
    fun getDefaultNetworks(): List<BlockchainNetwork> {
        return RealBlockchainNetworks.getAllNetworks()
    }
    
    // Fetch real token prices from API
    suspend fun fetchTokenPrices(): List<TokenData> {
        return RealBlockchainNetworks.fetchTokenListFromAPI()
    }
}
