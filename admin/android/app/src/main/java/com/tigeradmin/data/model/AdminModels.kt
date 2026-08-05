package com.tigeradmin.data.model

import com.google.gson.annotations.SerializedName

/**
 * Admin User Model
 * Represents an admin user in the system
 */
data class AdminUser(
    @SerializedName("id")
    val id: Long,
    
    @SerializedName("username")
    val username: String,
    
    @SerializedName("email")
    val email: String,
    
    @SerializedName("first_name")
    val firstName: String?,
    
    @SerializedName("last_name")
    val lastName: String?,
    
    @SerializedName("role")
    val role: AdminRole,
    
    @SerializedName("permissions")
    val permissions: List<String>,
    
    @SerializedName("status")
    val status: AdminStatus,
    
    @SerializedName("two_factor_enabled")
    val twoFactorEnabled: Boolean,
    
    @SerializedName("last_login_at")
    val lastLoginAt: String?,
    
    @SerializedName("created_at")
    val createdAt: String,
    
    @SerializedName("updated_at")
    val updatedAt: String,
    
    @SerializedName("avatar_url")
    val avatarUrl: String?,
    
    @SerializedName("phone")
    val phone: String?,
    
    @SerializedName("department")
    val department: String?
)

/**
 * Admin Role Enum
 */
enum class AdminRole {
    @SerializedName("super_admin")
    SUPER_ADMIN,
    
    @SerializedName("admin")
    ADMIN,
    
    @SerializedName("support")
    SUPPORT,
    
    @SerializedName("analyst")
    ANALYST,
    
    @SerializedName("moderator")
    MODERATOR
}

/**
 * Admin Status Enum
 */
enum class AdminStatus {
    @SerializedName("active")
    ACTIVE,
    
    @SerializedName("suspended")
    SUSPENDED,
    
    @SerializedName("inactive")
    INACTIVE
}

/**
 * Platform User Model
 * Represents a user in the TigerWallet platform
 */
data class PlatformUser(
    @SerializedName("id")
    val id: Long,
    
    @SerializedName("email")
    val email: String,
    
    @SerializedName("username")
    val username: String?,
    
    @SerializedName("wallet_address")
    val walletAddress: String?,
    
    @SerializedName("status")
    val status: UserStatus,
    
    @SerializedName("kyc_status")
    val kycStatus: KYCStatus,
    
    @SerializedName("kyc_level")
    val kycLevel: Int,
    
    @SerializedName("risk_score")
    val riskScore: Int,
    
    @SerializedName("created_at")
    val createdAt: String,
    
    @SerializedName("last_login_at")
    val lastLoginAt: String?,
    
    @SerializedName("registration_ip")
    val registrationIp: String?,
    
    @SerializedName("tags")
    val tags: List<String>,
    
    @SerializedName("referred_by")
    val referredBy: String?,
    
    @SerializedName("white_label_id")
    val whiteLabelId: Long?
)

/**
 * User Status Enum
 */
enum class UserStatus {
    @SerializedName("active")
    ACTIVE,
    
    @SerializedName("pending")
    PENDING,
    
    @SerializedName("suspended")
    SUSPENDED,
    
    @SerializedName("banned")
    BANNED
}

/**
 * KYC Status Enum
 */
enum class KYCStatus {
    @SerializedName("none")
    NONE,
    
    @SerializedName("pending")
    PENDING,
    
    @SerializedName("level1")
    LEVEL1,
    
    @SerializedName("level2")
    LEVEL2,
    
    @SerializedName("level3")
    LEVEL3,
    
    @SerializedName("rejected")
    REJECTED
}

/**
 * Transaction Model
 */
data class Transaction(
    @SerializedName("id")
    val id: Long,
    
    @SerializedName("hash")
    val hash: String,
    
    @SerializedName("type")
    val type: TransactionType,
    
    @SerializedName("chain")
    val chain: String,
    
    @SerializedName("from_address")
    val fromAddress: String,
    
    @SerializedName("to_address")
    val toAddress: String,
    
    @SerializedName("amount")
    val amount: String,
    
    @SerializedName("token")
    val token: String,
    
    @SerializedName("token_amount")
    val tokenAmount: String?,
    
    @SerializedName("status")
    val status: TransactionStatus,
    
    @SerializedName("block_number")
    val blockNumber: Long?,
    
    @SerializedName("gas_used")
    val gasUsed: String?,
    
    @SerializedName("gas_price")
    val gasPrice: String?,
    
    @SerializedName("timestamp")
    val timestamp: String,
    
    @SerializedName("flagged")
    val flagged: Boolean,
    
    @SerializedName("flag_reason")
    val flagReason: String?,
    
    @SerializedName("user_id")
    val userId: Long
)

enum class TransactionType {
    @SerializedName("transfer") TRANSFER,
    @SerializedName("swap") SWAP,
    @SerializedName("stake") STAKE,
    @SerializedName("unstake") UNSTAKE,
    @SerializedName("bridge") BRIDGE,
    @SerializedName("withdraw") WITHDRAW,
    @SerializedName("deposit") DEPOSIT,
    @SerializedName("mint") MINT,
    @SerializedName("burn") BURN
}

enum class TransactionStatus {
    @SerializedName("pending") PENDING,
    @SerializedName("confirmed") CONFIRMED,
    @SerializedName("failed") FAILED
}

/**
 * KYC Application Model
 */
data class KYCApplication(
    @SerializedName("id") val id: Long,
    @SerializedName("user_id") val userId: Long,
    @SerializedName("user_email") val userEmail: String,
    @SerializedName("level") val level: Int,
    @SerializedName("status") val status: KYCApplicationStatus,
    @SerializedName("submitted_at") val submittedAt: String,
    @SerializedName("reviewed_at") val reviewedAt: String?,
    @SerializedName("reviewed_by") val reviewedBy: String?,
    @SerializedName("rejection_reason") val rejectionReason: String?,
    @SerializedName("documents") val documents: List<KYCDocument>,
    @SerializedName("ip_address") val ipAddress: String?,
    @SerializedName("notes") val notes: String?
)

enum class KYCApplicationStatus {
    @SerializedName("pending") PENDING,
    @SerializedName("approved") APPROVED,
    @SerializedName("rejected") REJECTED
}

data class KYCDocument(
    @SerializedName("type") val type: String,
    @SerializedName("url") val url: String,
    @SerializedName("status") val status: String,
    @SerializedName("verified_at") val verifiedAt: String?
)

/**
 * Token Model
 */
data class Token(
    @SerializedName("id") val id: Long,
    @SerializedName("name") val name: String,
    @SerializedName("symbol") val symbol: String,
    @SerializedName("contract_address") val contractAddress: String,
    @SerializedName("chain") val chain: String,
    @SerializedName("decimals") val decimals: Int,
    @SerializedName("total_supply") val totalSupply: String,
    @SerializedName("logo_url") val logoUrl: String?,
    @SerializedName("website") val website: String?,
    @SerializedName("description") val description: String?,
    @SerializedName("price") val price: String?,
    @SerializedName("market_cap") val marketCap: String?,
    @SerializedName("volume_24h") val volume24h: String?,
    @SerializedName("price_change_24h") val priceChange24h: String?,
    @SerializedName("is_active") val isActive: Boolean,
    @SerializedName("is_verified") val isVerified: Boolean,
    @SerializedName("listing_fee") val listingFee: String?,
    @SerializedName("listed_at") val listedAt: String?
)

/**
 * Token Listing Request Model
 */
data class TokenListingRequest(
    @SerializedName("id") val id: Long,
    @SerializedName("token_symbol") val tokenSymbol: String,
    @SerializedName("token_name") val tokenName: String,
    @SerializedName("contract_address") val contractAddress: String,
    @SerializedName("chain_id") val chainId: Long,
    @SerializedName("tier") val tier: ListingTier,
    @SerializedName("status") val status: ListingStatus,
    @SerializedName("requester_address") val requesterAddress: String,
    @SerializedName("requester_email") val requesterEmail: String,
    @SerializedName("one_time_fee") val oneTimeFee: String,
    @SerializedName("monthly_fee") val monthlyFee: String,
    @SerializedName("requested_at") val requestedAt: String
)

enum class ListingTier {
    @SerializedName("basic") BASIC,
    @SerializedName("standard") STANDARD,
    @SerializedName("premium") PREMIUM,
    @SerializedName("premium_plus") PREMIUM_PLUS
}

enum class ListingStatus {
    @SerializedName("pending") PENDING,
    @SerializedName("approved") APPROVED,
    @SerializedName("rejected") REJECTED
}

/**
 * Withdrawal Request Model
 */
data class WithdrawalRequest(
    @SerializedName("id") val id: Long,
    @SerializedName("user_id") val userId: Long,
    @SerializedName("user_email") val userEmail: String,
    @SerializedName("amount") val amount: String,
    @SerializedName("token") val token: String,
    @SerializedName("chain") val chain: String,
    @SerializedName("to_address") val toAddress: String,
    @SerializedName("status") val status: WithdrawalStatus,
    @SerializedName("approved_at") val approvedAt: String?,
    @SerializedName("approved_by") val approvedBy: String?,
    @SerializedName("rejected_at") val rejectedAt: String?,
    @SerializedName("rejection_reason") val rejectionReason: String?,
    @SerializedName("processed_at") val processedAt: String?,
    @SerializedName("tx_hash") val txHash: String?,
    @SerializedName("fee") val fee: String?,
    @SerializedName("created_at") val createdAt: String
)

enum class WithdrawalStatus {
    @SerializedName("pending") PENDING,
    @SerializedName("approved") APPROVED,
    @SerializedName("rejected") REJECTED,
    @SerializedName("processing") PROCESSING,
    @SerializedName("completed") COMPLETED,
    @SerializedName("failed") FAILED
}

/**
 * White Label Model
 */
data class WhiteLabel(
    @SerializedName("id") val id: Long,
    @SerializedName("name") val name: String,
    @SerializedName("slug") val slug: String,
    @SerializedName("domain") val domain: String?,
    @SerializedName("logo_url") val logoUrl: String?,
    @SerializedName("favicon_url") val faviconUrl: String?,
    @SerializedName("primary_color") val primaryColor: String,
    @SerializedName("secondary_color") val secondaryColor: String?,
    @SerializedName("status") val status: WhiteLabelStatus,
    @SerializedName("contact_email") val contactEmail: String?,
    @SerializedName("contact_phone") val contactPhone: String?,
    @SerializedName("address") val address: String?,
    @SerializedName("description") val description: String?,
    @SerializedName("features") val features: List<String>,
    @SerializedName("fee_structure") val feeStructure: FeeStructure?,
    @SerializedName("created_at") val createdAt: String,
    @SerializedName("expires_at") val expiresAt: String?
)

enum class WhiteLabelStatus {
    @SerializedName("active") ACTIVE,
    @SerializedName("suspended") SUSPENDED,
    @SerializedName("pending") PENDING
}

data class FeeStructure(
    @SerializedName("trading_fee") val tradingFee: String,
    @SerializedName("withdrawal_fee") val withdrawalFee: String,
    @SerializedName("deposit_fee") val depositFee: String,
    @SerializedName("listing_fee") val listingFee: String
)

/**
 * Fee Configuration Model
 */
data class FeeConfig(
    @SerializedName("id") val id: Long,
    @SerializedName("fee_type") val feeType: String,
    @SerializedName("chain_id") val chainId: Long?,
    @SerializedName("token_symbol") val tokenSymbol: String?,
    @SerializedName("fee_amount_usd") val feeAmountUsd: String,
    @SerializedName("fee_percentage") val feePercentage: String,
    @SerializedName("min_fee_usd") val minFeeUsd: String,
    @SerializedName("max_fee_usd") val maxFeeUsd: String?,
    @SerializedName("is_active") val isActive: Boolean
)

/**
 * System Status Model
 */
data class SystemStatus(
    @SerializedName("service_name") val serviceName: String,
    @SerializedName("status") val status: String,
    @SerializedName("uptime") val uptime: String,
    @SerializedName("latency") val latency: String,
    @SerializedName("last_check") val lastCheck: String
)

/**
 * Analytics Data Model
 */
data class AnalyticsData(
    @SerializedName("total_users") val totalUsers: Long,
    @SerializedName("active_users") val activeUsers: Long,
    @SerializedName("total_volume") val totalVolume: String,
    @SerializedName("daily_transactions") val dailyTransactions: Long,
    @SerializedName("total_fees") val totalFees: String,
    @SerializedName("pending_kyc") val pendingKyc: Long,
    @SerializedName("system_health") val systemHealth: String,
    @SerializedName("timestamp") val timestamp: String
)

/**
 * Bot Instance Model
 */
data class BotInstance(
    @SerializedName("id") val id: Long,
    @SerializedName("user_id") val userId: Long,
    @SerializedName("user_email") val userEmail: String,
    @SerializedName("bot_type") val botType: String,
    @SerializedName("name") val name: String,
    @SerializedName("status") val status: BotStatus,
    @SerializedName("connected_dexs") val connectedDexs: Int,
    @SerializedName("connected_cexs") val connectedCexs: Int,
    @SerializedName("total_pnl") val totalPnl: String,
    @SerializedName("total_volume") val totalVolume: String,
    @SerializedName("total_orders") val totalOrders: Long,
    @SerializedName("avg_latency_us") val avgLatencyUs: Long,
    @SerializedName("created_at") val createdAt: String,
    @SerializedName("last_trade_at") val lastTradeAt: String?
)

enum class BotStatus {
    @SerializedName("running") RUNNING,
    @SerializedName("stopped") STOPPED,
    @SerializedName("error") ERROR,
    @SerializedName("paused") PAUSED
}

/**
 * API Key Model
 */
data class APIKey(
    @SerializedName("id") val id: Long,
    @SerializedName("name") val name: String,
    @SerializedName("key") val key: String,
    @SerializedName("admin_id") val adminId: Long,
    @SerializedName("permissions") val permissions: APIKeyPermissions,
    @SerializedName("rate_limit") val rateLimit: Int,
    @SerializedName("last_used_at") val lastUsedAt: String?,
    @SerializedName("expires_at") val expiresAt: String?,
    @SerializedName("status") val status: String,
    @SerializedName("created_at") val createdAt: String
)

data class APIKeyPermissions(
    @SerializedName("trading") val trading: Boolean,
    @SerializedName("reading") val reading: Boolean,
    @SerializedName("withdrawal") val withdrawal: Boolean
)

/**
 * Blockchain Model
 */
data class Blockchain(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String,
    @SerializedName("symbol") val symbol: String,
    @SerializedName("chain_id") val chainId: Long,
    @SerializedName("chain_id_hex") val chainIdHex: String?,
    @SerializedName("is_evm") val isEvm: Boolean,
    @SerializedName("is_active") val isActive: Boolean,
    @SerializedName("explorer_url") val explorerUrl: String?,
    @SerializedName("rpc_url") val rpcUrl: String?,
    @SerializedName("native_token_symbol") val nativeTokenSymbol: String,
    @SerializedName("avg_gas_price_gwei") val avgGasPriceGwei: Double
)
