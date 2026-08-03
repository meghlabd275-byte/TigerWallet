package com.tigerwallet.admin.data.api.model

import com.google.gson.annotations.SerializedName

// Auth Models
data class LoginRequest(val email: String, val password: String)
data class RefreshTokenRequest(@SerializedName("refresh_token") val refreshToken: String)
data class LoginResponse(val token: String, @SerializedName("refresh_token") val refreshToken: String, @SerializedName("expires_in") val expiresIn: Long, val admin: AdminResponse)

// Admin
data class AdminResponse(val id: String, val username: String, val email: String, val role: String, val status: String, @SerializedName("two_factor_enabled") val twoFactorEnabled: Boolean, @SerializedName("created_at") val createdAt: String)
data class AdminsResponse(val data: List<AdminResponse>, val meta: PaginationMeta)
data class CreateAdminRequest(val username: String, val email: String, val password: String, val role: String, val permissions: List<String>)

// User
data class UsersResponse(val data: List<UserResponse>, val meta: PaginationMeta)
data class UserResponse(val id: String, val username: String, val email: String, val status: String, @SerializedName("kyc_status") val kycStatus: String, @SerializedName("created_at") val createdAt: String)
data class SuspendRequest(val reason: String)
data class BanRequest(val reason: String)

// KYC
data class KYCListResponse(val data: List<KYCResponse>, val meta: PaginationMeta)
data class KYCResponse(val id: String, @SerializedName("user_id") val userId: String, val level: Int, val status: String, @SerializedName("created_at") val createdAt: String)
data class ApproveKYCRequest(val notes: String? = null)
data class RejectKYCRequest(val reason: String)

// Token
data class TokensResponse(val data: List<TokenResponse>, val meta: PaginationMeta)
data class TokenResponse(val id: String, val name: String, val symbol: String, @SerializedName("chain_id") val chainId: String, @SerializedName("is_active") val isActive: Boolean, @SerializedName("is_verified") val isVerified: Boolean)

// Pair
data class PairsResponse(val data: List<PairResponse>, val meta: PaginationMeta)
data class PairResponse(val id: String, @SerializedName("base_symbol") val baseSymbol: String, @SerializedName("quote_symbol") val quoteSymbol: String, val status: String, @SerializedName("current_price") val currentPrice: Double?)

// Transaction
data class TransactionsResponse(val data: List<TransactionResponse>, val meta: PaginationMeta)
data class TransactionResponse(val id: String, @SerializedName("tx_hash") val txHash: String?, val type: String, val status: String, val amount: String, @SerializedName("created_at") val createdAt: String)

// Withdrawal
data class WithdrawalsResponse(val data: List<WithdrawalResponse>, val meta: PaginationMeta)
data class WithdrawalResponse(val id: String, @SerializedName("user_id") val userId: String, val token: String, val amount: String, val status: String, @SerializedName("created_at") val createdAt: String)
data class RejectWithdrawalRequest(val reason: String)

// Chain
data class ChainsResponse(val data: List<ChainResponse>)
data class ChainResponse(val id: String, val name: String, val symbol: String, val type: String, @SerializedName("is_active") val isActive: Boolean)
data class CreateChainRequest(@SerializedName("chain_id") val chainId: Long, val name: String, val symbol: String, val type: String, @SerializedName("rpc_urls") val rpcUrls: List<String>)

// Fee
data class FeesResponse(val data: List<FeeResponse>)
data class FeeResponse(val id: String, @SerializedName("fee_type") val feeType: String, @SerializedName("fee_percent") val feePercent: Double)

// White Label
data class WhiteLabelsResponse(val data: List<WhiteLabelResponse>, val meta: PaginationMeta)
data class WhiteLabelResponse(val id: String, val name: String, val domain: String, val status: String, @SerializedName("platform_fee_percent") val platformFeePercent: Double)
data class CreateWhiteLabelRequest(val name: String, val domain: String)

// Dashboard
data class DashboardResponse(@SerializedName("total_users") val totalUsers: Long, @SerializedName("active_users") val activeUsers: Long, @SerializedName("total_transactions") val totalTransactions: Long, @SerializedName("volume_24h") val volume24h: Double)
data class AnalyticsResponse(val labels: List<String>, val values: List<Double>)

// Audit
data class AuditLogsResponse(val data: List<AuditLogResponse>, val meta: PaginationMeta)
data class AuditLogResponse(val id: String, val action: String, @SerializedName("admin_id") val adminId: String?, @SerializedName("created_at") val createdAt: String)

// Common
data class PaginationMeta(val page: Int, val limit: Int, val total: Int, @SerializedName("total_pages") val totalPages: Int)
