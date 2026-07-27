package com.tigerwallet.app.data.models

import java.util.UUID

// Wallet model
data class Wallet(
    val id: String = UUID.randomUUID().toString(),
    var name: String,
    var address: String = "",
    var publicKey: String = "",
    val chainAddresses: MutableMap<String, String> = mutableMapOf(),
    val createdAt: Long = System.currentTimeMillis(),
    var isBackedUp: Boolean = false
)

// Token balance model
data class TokenBalance(
    val id: String,
    val symbol: String,
    val name: String,
    val address: String?,
    val decimals: Int,
    var balance: Double,
    var price: Double,
    val chainId: Long,
    val logoUrl: String? = null
) {
    val usdValue: Double
        get() = balance * price
}

// NFT model
data class NFT(
    val id: String,
    val tokenId: String,
    val contractAddress: String,
    val name: String,
    val description: String?,
    val imageUrl: String?,
    val chainId: Long,
    val collectionName: String?
)

// Transaction model
data class Transaction(
    val id: String = UUID.randomUUID().toString(),
    val hash: String,
    val from: String,
    val to: String,
    val amount: Double,
    val symbol: String,
    val decimals: Int,
    val chainId: Long,
    val status: TransactionStatus,
    val timestamp: Long,
    val type: TransactionType,
    val gasUsed: Double? = null,
    val gasPrice: Double? = null
)

enum class TransactionStatus {
    PENDING, CONFIRMED, FAILED
}

enum class TransactionType {
    SEND, RECEIVE, SWAP, STAKE, UNSTAKE, APPROVE, CONTRACT_INTERACTION
}

// Blockchain network model
data class BlockchainNetwork(
    val id: String,
    val name: String,
    val symbol: String,
    val chainId: Long,
    val isEVM: Boolean,
    val rpcUrl: String = "",
    val explorerUrl: String = ""
)

// Swap quote model
data class SwapQuote(
    val fromToken: String,
    val toToken: String,
    val fromAmount: Double,
    val toAmount: Double,
    val priceImpact: Double,
    val route: List<String>,
    val gasEstimate: Double
)

// Staking position model
data class StakingPosition(
    val id: String,
    val validator: String,
    val amount: Double,
    val rewards: Double,
    val unlockTime: Long?
)

// Wallet state
sealed class WalletState {
    object Empty : WalletState()
    object Loading : WalletState()
    data class Loaded(val wallet: Wallet, val tokens: List<TokenBalance>) : WalletState()
    data class Error(val message: String) : WalletState()
}
