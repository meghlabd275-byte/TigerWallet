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
    
    init {
        loadWallets()
    }
    
    fun createWallet(name: String, mnemonic: List<String>? = null): Wallet {
        val walletService = ServiceLocator.walletService
        val words = mnemonic ?: walletService.generateMnemonic()
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
    
    fun importWallet(mnemonic: List<String>, name: String): Wallet? {
        if (!validateMnemonic(mnemonic)) return null
        return createWallet(name, mnemonic)
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
    
    private fun getTokenPrice(symbol: String): Double = when (symbol) {
        "ETH" -> 3500.0
        "BNB" -> 600.0
        "MATIC" -> 0.8
        "SOL" -> 100.0
        "TRX" -> 0.1
        else -> 0.0
    }
}

// Network Repository
object NetworkRepository {
    fun getDefaultNetworks(): List<BlockchainNetwork> = listOf(
        BlockchainNetwork("ethereum", "Ethereum", "ETH", 1, true, "https://eth.llamarpc.com", "https://etherscan.io"),
        BlockchainNetwork("bsc", "BNB Chain", "BNB", 56, true, "https://bsc-dataseed.binance.org", "https://bscscan.com"),
        BlockchainNetwork("polygon", "Polygon", "MATIC", 137, true, "https://polygon-rpc.com", "https://polygonscan.com"),
        BlockchainNetwork("arbitrum", "Arbitrum", "ETH", 42161, true, "https://arb1.arbitrum.io/rpc", "https://arbiscan.io"),
        BlockchainNetwork("optimism", "Optimism", "ETH", 10, true, "https://mainnet.optimism.io", "https://optimistic.etherscan.io"),
        BlockchainNetwork("avalanche", "Avalanche", "AVAX", 43114, true, "https://api.avax.network/ext/bc/C/rpc", "https://snowtrace.io"),
        BlockchainNetwork("solana", "Solana", "SOL", 0, false, "https://api.mainnet-beta.solana.com", "https://solscan.io"),
        BlockchainNetwork("tron", "Tron", "TRX", 195, false, "https://api.trongrid.io", "https://tronscan.org"),
        BlockchainNetwork("bitcoin", "Bitcoin", "BTC", 0, false, "https://blockstream.info/api", "https://blockstream.info")
    )
}
