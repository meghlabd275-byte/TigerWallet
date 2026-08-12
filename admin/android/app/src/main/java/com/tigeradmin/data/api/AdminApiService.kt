package com.tigeradmin.data.api

import com.tigeradmin.data.model.*
import retrofit2.Response
import retrofit2.http.*

/**
 * Admin API Service Interface
 * Defines all API endpoints for the TigerWallet Admin Platform
 */
interface AdminApiService {

    // ========== Authentication ==========
    
    @POST("auth/login")
    suspend fun login(@Body request: LoginRequest): Response<LoginResponse>
    
    @POST("auth/logout")
    suspend fun logout(): Response<Unit>
    
    @POST("auth/refresh")
    suspend fun refreshToken(@Body request: RefreshTokenRequest): Response<LoginResponse>
    
    @POST("auth/2fa/verify")
    suspend fun verify2FA(@Body request: Verify2FARequest): Response<LoginResponse>
    
    @POST("auth/2fa/enable")
    suspend fun enable2FA(): Response<Enable2FAResponse>
    
    @POST("auth/2fa/disable")
    suspend fun disable2FA(@Body request: Disable2FARequest): Response<Unit>
    
    @POST("auth/password/change")
    suspend fun changePassword(@Body request: ChangePasswordRequest): Response<Unit>
    
    @POST("auth/password/reset")
    suspend fun resetPassword(@Body request: ResetPasswordRequest): Response<Unit>

    // ========== Admin Users Management ==========
    
    @GET("admins")
    suspend fun getAdmins(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("role") role: String? = null,
        @Query("status") status: String? = null,
        @Query("search") search: String? = null
    ): Response<AdminListResponse>
    
    @GET("admins/{id}")
    suspend fun getAdmin(@Path("id") id: Long): Response<AdminUser>
    
    @POST("admins")
    suspend fun createAdmin(@Body request: CreateAdminRequest): Response<AdminUser>
    
    @PUT("admins/{id}")
    suspend fun updateAdmin(
        @Path("id") id: Long,
        @Body request: UpdateAdminRequest
    ): Response<AdminUser>
    
    @DELETE("admins/{id}")
    suspend fun deleteAdmin(@Path("id") id: Long): Response<Unit>
    
    @POST("admins/{id}/suspend")
    suspend fun suspendAdmin(@Path("id") id: Long): Response<Unit>
    
    @POST("admins/{id}/activate")
    suspend fun activateAdmin(@Path("id") id: Long): Response<Unit>
    
    @GET("admins/me")
    suspend fun getCurrentAdmin(): Response<AdminUser>
    
    @GET("admins/activity")
    suspend fun getAdminActivity(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 50,
        @Query("admin_id") adminId: Long? = null,
        @Query("action") action: String? = null,
        @Query("from_date") fromDate: String? = null,
        @Query("to_date") toDate: String? = null
    ): Response<AdminActivityResponse>

    // ========== Platform Users Management ==========
    
    @GET("users")
    suspend fun getUsers(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("status") status: String? = null,
        @Query("kyc_status") kycStatus: String? = null,
        @Query("kyc_level") kycLevel: Int? = null,
        @Query("risk_score_min") riskScoreMin: Int? = null,
        @Query("risk_score_max") riskScoreMax: Int? = null,
        @Query("search") search: String? = null,
        @Query("from_date") fromDate: String? = null,
        @Query("to_date") toDate: String? = null
    ): Response<UserListResponse>
    
    @GET("users/{id}")
    suspend fun getUser(@Path("id") id: Long): Response<PlatformUser>
    
    @PUT("users/{id}")
    suspend fun updateUser(
        @Path("id") id: Long,
        @Body request: UpdateUserRequest
    ): Response<PlatformUser>
    
    @POST("users/{id}/suspend")
    suspend fun suspendUser(
        @Path("id") id: Long,
        @Body request: SuspendUserRequest
    ): Response<Unit>
    
    @POST("users/{id}/ban")
    suspend fun banUser(
        @Path("id") id: Long,
        @Body request: BanUserRequest
    ): Response<Unit>
    
    @POST("users/{id}/activate")
    suspend fun activateUser(@Path("id") id: Long): Response<Unit>
    
    @POST("users/{id}/tags")
    suspend fun updateUserTags(
        @Path("id") id: Long,
        @Body request: UpdateUserTagsRequest
    ): Response<Unit>
    
    @GET("users/{id}/transactions")
    suspend fun getUserTransactions(
        @Path("id") userId: Long,
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("status") status: String? = null,
        @Query("type") type: String? = null,
        @Query("chain") chain: String? = null
    ): Response<TransactionListResponse>
    
    @GET("users/{id}/kyc")
    suspend fun getUserKYC(@Path("id") userId: Long): Response<KYCApplication>
    
    @GET("users/stats")
    suspend fun getUserStats(): Response<UserStatsResponse>

    // ========== Transactions Management ==========
    
    @GET("transactions")
    suspend fun getTransactions(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("status") status: String? = null,
        @Query("type") type: String? = null,
        @Query("chain") chain: String? = null,
        @Query("user_id") userId: Long? = null,
        @Query("flagged") flagged: Boolean? = null,
        @Query("min_amount") minAmount: String? = null,
        @Query("max_amount") maxAmount: String? = null,
        @Query("from_date") fromDate: String? = null,
        @Query("to_date") toDate: String? = null,
        @Query("search") search: String? = null
    ): Response<TransactionListResponse>
    
    @GET("transactions/{id}")
    suspend fun getTransaction(@Path("id") id: Long): Response<Transaction>
    
    @POST("transactions/{id}/flag")
    suspend fun flagTransaction(
        @Path("id") id: Long,
        @Body request: FlagTransactionRequest
    ): Response<Unit>
    
    @POST("transactions/{id}/unflag")
    suspend fun unflagTransaction(@Path("id") id: Long): Response<Unit>
    
    @GET("transactions/stats")
    suspend fun getTransactionStats(
        @Query("from_date") fromDate: String? = null,
        @Query("to_date") toDate: String? = null,
        @Query("chain") chain: String? = null
    ): Response<TransactionStatsResponse>

    // ========== KYC Management ==========
    
    @GET("kyc")
    suspend fun getKYCApplications(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("status") status: String? = null,
        @Query("level") level: Int? = null,
        @Query("from_date") fromDate: String? = null,
        @Query("to_date") toDate: String? = null
    ): Response<KYCListResponse>
    
    @GET("kyc/{id}")
    suspend fun getKYCApplication(@Path("id") id: Long): Response<KYCApplication>
    
    @POST("kyc/{id}/approve")
    suspend fun approveKYC(
        @Path("id") id: Long,
        @Body request: ApproveKYCRequest
    ): Response<Unit>
    
    @POST("kyc/{id}/reject")
    suspend fun rejectKYC(
        @Path("id") id: Long,
        @Body request: RejectKYCRequest
    ): Response<Unit>
    
    @POST("kyc/{id}/request-more-info")
    suspend fun requestMoreInfo(
        @Path("id") id: Long,
        @Body request: RequestMoreInfoRequest
    ): Response<Unit>
    
    @GET("kyc/stats")
    suspend fun getKYCStats(): Response<KYCStatsResponse>

    // ========== Token Management ==========
    
    @GET("tokens")
    suspend fun getTokens(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("chain") chain: String? = null,
        @Query("is_active") isActive: Boolean? = null,
        @Query("is_verified") isVerified: Boolean? = null,
        @Query("search") search: String? = null
    ): Response<TokenListResponse>
    
    @GET("tokens/{id}")
    suspend fun getToken(@Path("id") id: Long): Response<Token>
    
    @POST("tokens")
    suspend fun createToken(@Body request: CreateTokenRequest): Response<Token>
    
    @PUT("tokens/{id}")
    suspend fun updateToken(
        @Path("id") id: Long,
        @Body request: UpdateTokenRequest
    ): Response<Token>
    
    @DELETE("tokens/{id}")
    suspend fun deleteToken(@Path("id") id: Long): Response<Unit>
    
    @POST("tokens/{id}/verify")
    suspend fun verifyToken(@Path("id") id: Long): Response<Unit>
    
    @POST("tokens/{id}/activate")
    suspend fun activateToken(@Path("id") id: Long): Response<Unit>
    
    @POST("tokens/{id}/deactivate")
    suspend fun deactivateToken(@Path("id") id: Long): Response<Unit>

    // ========== Token Listing Requests ==========
    
    @GET("token-listings")
    suspend fun getTokenListings(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("status") status: String? = null,
        @Query("tier") tier: String? = null
    ): Response<TokenListingListResponse>
    
    @GET("token-listings/{id}")
    suspend fun getTokenListing(@Path("id") id: Long): Response<TokenListingRequest>
    
    @POST("token-listings/{id}/approve")
    suspend fun approveTokenListing(@Path("id") id: Long): Response<Unit>
    
    @POST("token-listings/{id}/reject")
    suspend fun rejectTokenListing(
        @Path("id") id: Long,
        @Body request: RejectListingRequest
    ): Response<Unit>

    // ========== Withdrawal Management ==========
    
    @GET("withdrawals")
    suspend fun getWithdrawals(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("status") status: String? = null,
        @Query("token") token: String? = null,
        @Query("chain") chain: String? = null,
        @Query("user_id") userId: Long? = null,
        @Query("from_date") fromDate: String? = null,
        @Query("to_date") toDate: String? = null
    ): Response<WithdrawalListResponse>
    
    @GET("withdrawals/{id}")
    suspend fun getWithdrawal(@Path("id") id: Long): Response<WithdrawalRequest>
    
    @POST("withdrawals/{id}/approve")
    suspend fun approveWithdrawal(
        @Path("id") id: Long,
        @Body request: ApproveWithdrawalRequest
    ): Response<Unit>
    
    @POST("withdrawals/{id}/reject")
    suspend fun rejectWithdrawal(
        @Path("id") id: Long,
        @Body request: RejectWithdrawalRequest
    ): Response<Unit>
    
    @POST("withdrawals/{id}/process")
    suspend fun processWithdrawal(
        @Path("id") id: Long,
        @Body request: ProcessWithdrawalRequest
    ): Response<Unit>
    
    @GET("withdrawals/stats")
    suspend fun getWithdrawalStats(
        @Query("from_date") fromDate: String? = null,
        @Query("to_date") toDate: String? = null
    ): Response<WithdrawalStatsResponse>

    // ========== Fee Configuration ==========
    
    @GET("fees")
    suspend fun getFeeConfigs(): Response<FeeConfigListResponse>
    
    @POST("fees")
    suspend fun createFeeConfig(@Body request: CreateFeeConfigRequest): Response<FeeConfig>
    
    @PUT("fees/{id}")
    suspend fun updateFeeConfig(
        @Path("id") id: Long,
        @Body request: UpdateFeeConfigRequest
    ): Response<FeeConfig>
    
    @DELETE("fees/{id}")
    suspend fun deleteFeeConfig(@Path("id") id: Long): Response<Unit>
    
    @POST("fees/{id}/activate")
    suspend fun activateFeeConfig(@Path("id") id: Long): Response<Unit>
    
    @POST("fees/{id}/deactivate")
    suspend fun deactivateFeeConfig(@Path("id") id: Long): Response<Unit>

    // ========== White Label Management ==========
    
    @GET("whitelabels")
    suspend fun getWhiteLabels(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("status") status: String? = null,
        @Query("search") search: String? = null
    ): Response<WhiteLabelListResponse>
    
    @GET("whitelabels/{id}")
    suspend fun getWhiteLabel(@Path("id") id: Long): Response<WhiteLabel>
    
    @POST("whitelabels")
    suspend fun createWhiteLabel(@Body request: CreateWhiteLabelRequest): Response<WhiteLabel>
    
    @PUT("whitelabels/{id}")
    suspend fun updateWhiteLabel(
        @Path("id") id: Long,
        @Body request: UpdateWhiteLabelRequest
    ): Response<WhiteLabel>
    
    @DELETE("whitelabels/{id}")
    suspend fun deleteWhiteLabel(@Path("id") id: Long): Response<Unit>
    
    @POST("whitelabels/{id}/activate")
    suspend fun activateWhiteLabel(@Path("id") id: Long): Response<Unit>
    
    @POST("whitelabels/{id}/suspend")
    suspend fun suspendWhiteLabel(@Path("id") id: Long): Response<Unit>
    
    @GET("whitelabels/{id}/users")
    suspend fun getWhiteLabelUsers(
        @Path("id") whiteLabelId: Long,
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20
    ): Response<UserListResponse>

    // ========== Bot Management ==========
    
    @GET("bots")
    suspend fun getBots(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("status") status: String? = null,
        @Query("bot_type") botType: String? = null,
        @Query("user_id") userId: Long? = null
    ): Response<BotListResponse>
    
    @GET("bots/{id}")
    suspend fun getBot(@Path("id") id: Long): Response<BotInstance>
    
    @POST("bots/{id}/start")
    suspend fun startBot(@Path("id") id: Long): Response<Unit>
    
    @POST("bots/{id}/stop")
    suspend fun stopBot(@Path("id") id: Long): Response<Unit>
    
    @POST("bots/{id}/pause")
    suspend fun pauseBot(@Path("id") id: Long): Response<Unit>
    
    @GET("bots/stats")
    suspend fun getBotStats(): Response<BotStatsResponse>

    // ========== API Keys Management ==========
    
    @GET("api-keys")
    suspend fun getAPIKeys(): Response<APIKeyListResponse>
    
    @POST("api-keys")
    suspend fun createAPIKey(@Body request: CreateAPIKeyRequest): Response<APIKey>
    
    @DELETE("api-keys/{id}")
    suspend fun deleteAPIKey(@Path("id") id: Long): Response<Unit>
    
    @POST("api-keys/{id}/revoke")
    suspend fun revokeAPIKey(@Path("id") id: Long): Response<Unit>
    
    @POST("api-keys/{id}/activate")
    suspend fun activateAPIKey(@Path("id") id: Long): Response<Unit>

    // ========== Blockchain Management ==========
    
    @GET("blockchains")
    suspend fun getBlockchains(): Response<BlockchainListResponse>
    
    @GET("blockchains/{id}")
    suspend fun getBlockchain(@Path("id") id: String): Response<Blockchain>
    
    @POST("blockchains")
    suspend fun createBlockchain(@Body request: CreateBlockchainRequest): Response<Blockchain>
    
    @PUT("blockchains/{id}")
    suspend fun updateBlockchain(
        @Path("id") id: String,
        @Body request: UpdateBlockchainRequest
    ): Response<Blockchain>
    
    @POST("blockchains/{id}/activate")
    suspend fun activateBlockchain(@Path("id") id: String): Response<Unit>
    
    @POST("blockchains/{id}/deactivate")
    suspend fun deactivateBlockchain(@Path("id") id: String): Response<Unit>

    // ========== System Status ==========
    
    @GET("system/status")
    suspend fun getSystemStatus(): Response<SystemStatusResponse>
    
    @GET("system/health")
    suspend fun getSystemHealth(): Response<SystemHealthResponse>
    
    @GET("system/logs")
    suspend fun getSystemLogs(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 50,
        @Query("service") service: String? = null,
        @Query("level") level: String? = null,
        @Query("from_date") fromDate: String? = null,
        @Query("to_date") toDate: String? = null
    ): Response<SystemLogsResponse>

    // ========== Analytics & Reports ==========
    
    @GET("analytics/overview")
    suspend fun getAnalyticsOverview(): Response<AnalyticsData>
    
    @GET("analytics/users")
    suspend fun getUserAnalytics(
        @Query("from_date") fromDate: String? = null,
        @Query("to_date") toDate: String? = null,
        @Query("group_by") groupBy: String? = null
    ): Response<UserAnalyticsResponse>
    
    @GET("analytics/transactions")
    suspend fun getTransactionAnalytics(
        @Query("from_date") fromDate: String? = null,
        @Query("to_date") toDate: String? = null,
        @Query("group_by") groupBy: String? = null
    ): Response<TransactionAnalyticsResponse>
    
    @GET("analytics/revenue")
    suspend fun getRevenueAnalytics(
        @Query("from_date") fromDate: String? = null,
        @Query("to_date") toDate: String? = null,
        @Query("group_by") groupBy: String? = null
    ): Response<RevenueAnalyticsResponse>
    
    @GET("reports/generate")
    suspend fun generateReport(
        @Query("type") type: String,
        @Query("from_date") fromDate: String,
        @Query("to_date") toDate: String,
        @Query("format") format: String = "json"
    ): Response<ReportResponse>

    // ========== Notifications ==========
    
    @GET("notifications")
    suspend fun getNotifications(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("read") read: Boolean? = null
    ): Response<NotificationListResponse>
    
    @POST("notifications/{id}/read")
    suspend fun markNotificationRead(@Path("id") id: Long): Response<Unit>
    
    @POST("notifications/read-all")
    suspend fun markAllNotificationsRead(): Response<Unit>
    
    @POST("notifications/send")
    suspend fun sendNotification(@Body request: SendNotificationRequest): Response<Unit>

    // ========== Settings ==========
    
    @GET("settings")
    suspend fun getSettings(): Response<SettingsResponse>
    
    @PUT("settings")
    suspend fun updateSettings(@Body request: UpdateSettingsRequest): Response<SettingsResponse>
    
    @GET("settings/{key}")
    suspend fun getSetting(@Path("key") key: String): Response<SettingResponse>
    
    @PUT("settings/{key}")
    suspend fun updateSetting(
        @Path("key") key: String,
        @Body request: UpdateSettingRequest
    ): Response<SettingResponse>
}

// ========== Request Models ==========

data class LoginRequest(
    val email: String,
    val password: String
)

data class LoginResponse(
    val token: String,
    val refresh_token: String,
    val expires_at: String,
    val admin: AdminUser
)

data class RefreshTokenRequest(
    val refresh_token: String
)

data class Verify2FARequest(
    val code: String
)

data class Enable2FAResponse(
    val secret: String,
    val qr_code: String
)

data class Disable2FARequest(
    val code: String
)

data class ChangePasswordRequest(
    val current_password: String,
    val new_password: String,
    val confirm_password: String
)

data class ResetPasswordRequest(
    val email: String
)

data class CreateAdminRequest(
    val username: String,
    val email: String,
    val password: String,
    val role: String,
    val first_name: String?,
    val last_name: String?,
    val permissions: List<String>?
)

data class UpdateAdminRequest(
    val username: String?,
    val email: String?,
    val first_name: String?,
    val last_name: String?,
    val role: String?,
    val permissions: List<String>?,
    val department: String?
)

data class AdminListResponse(
    val data: List<AdminUser>,
    val pagination: Pagination
)

data class AdminActivityResponse(
    val data: List<AdminActivity>,
    val pagination: Pagination
)

data class AdminActivity(
    val id: Long,
    val admin_id: Long,
    val admin: AdminUser?,
    val action: String,
    val resource: String,
    val resource_id: String,
    val details: Map<String, Any>?,
    val ip_address: String,
    val user_agent: String,
    val status: String,
    val error_message: String?,
    val created_at: String
)

data class UpdateUserRequest(
    val status: String?,
    val kyc_level: Int?,
    val risk_score: Int?,
    val tags: List<String>?
)

data class SuspendUserRequest(
    val reason: String,
    val duration_days: Int?
)

data class BanUserRequest(
    val reason: String,
    val permanent: Boolean
)

data class UpdateUserTagsRequest(
    val tags: List<String>
)

data class UserListResponse(
    val data: List<PlatformUser>,
    val pagination: Pagination
)

data class UserStatsResponse(
    val total_users: Long,
    val active_users: Long,
    val suspended_users: Long,
    val banned_users: Long,
    val kyc_pending: Long,
    val kyc_approved: Long
)

data class FlagTransactionRequest(
    val reason: String
)

data class TransactionListResponse(
    val data: List<Transaction>,
    val pagination: Pagination
)

data class TransactionStatsResponse(
    val total_transactions: Long,
    val total_volume: String,
    val avg_transaction_size: String,
    val by_status: Map<String, Long>,
    val by_type: Map<String, Long>,
    val by_chain: Map<String, Long>
)

data class ApproveKYCRequest(
    val notes: String?
)

data class RejectKYCRequest(
    val reason: String
)

data class RequestMoreInfoRequest(
    val message: String
)

data class KYCListResponse(
    val data: List<KYCApplication>,
    val pagination: Pagination
)

data class KYCStatsResponse(
    val total: Long,
    val pending: Long,
    val approved: Long,
    val rejected: Long,
    val by_level: Map<Int, Long>
)

data class CreateTokenRequest(
    val name: String,
    val symbol: String,
    val contract_address: String,
    val chain: String,
    val decimals: Int,
    val total_supply: String,
    val logo_url: String?,
    val website: String?,
    val description: String?
)

data class UpdateTokenRequest(
    val name: String?,
    val logo_url: String?,
    val website: String?,
    val description: String?,
    val is_active: Boolean?
)

data class TokenListResponse(
    val data: List<Token>,
    val pagination: Pagination
)

data class RejectListingRequest(
    val reason: String
)

data class TokenListingListResponse(
    val data: List<TokenListingRequest>,
    val pagination: Pagination
)

data class ApproveWithdrawalRequest(
    val notes: String?
)

data class RejectWithdrawalRequest(
    val reason: String
)

data class ProcessWithdrawalRequest(
    val tx_hash: String,
    val notes: String?
)

data class WithdrawalListResponse(
    val data: List<WithdrawalRequest>,
    val pagination: Pagination
)

data class WithdrawalStatsResponse(
    val total: Long,
    val pending: Long,
    val approved: Long,
    val rejected: Long,
    val completed: Long,
    val total_amount: String
)

data class CreateFeeConfigRequest(
    val fee_type: String,
    val chain_id: Long?,
    val token_symbol: String?,
    val fee_amount_usd: String,
    val fee_percentage: String,
    val min_fee_usd: String,
    val max_fee_usd: String?
)

data class UpdateFeeConfigRequest(
    val fee_amount_usd: String?,
    val fee_percentage: String?,
    val min_fee_usd: String?,
    val max_fee_usd: String?
)

data class FeeConfigListResponse(
    val data: List<FeeConfig>
)

data class CreateWhiteLabelRequest(
    val name: String,
    val slug: String,
    val domain: String?,
    val primary_color: String,
    val secondary_color: String?,
    val contact_email: String?,
    val contact_phone: String?,
    val address: String?,
    val description: String?,
    val features: List<String>
)

data class UpdateWhiteLabelRequest(
    val name: String?,
    val domain: String?,
    val primary_color: String?,
    val secondary_color: String?,
    val contact_email: String?,
    val contact_phone: String?,
    val address: String?,
    val description: String?,
    val features: List<String>?,
    val fee_structure: FeeStructure?
)

data class WhiteLabelListResponse(
    val data: List<WhiteLabel>,
    val pagination: Pagination
)

data class BotListResponse(
    val data: List<BotInstance>,
    val pagination: Pagination
)

data class BotStatsResponse(
    val total_bots: Long,
    val running_bots: Long,
    val stopped_bots: Long,
    val error_bots: Long,
    val total_pnl: String,
    val total_volume: String
)

data class CreateAPIKeyRequest(
    val name: String,
    val permissions: APIKeyPermissions,
    val rate_limit: Int,
    val expires_at: String?
)

data class APIKeyListResponse(
    val data: List<APIKey>
)

data class CreateBlockchainRequest(
    val name: String,
    val symbol: String,
    val chain_id: Long,
    val chain_id_hex: String?,
    val is_evm: Boolean,
    val explorer_url: String?,
    val rpc_url: String?,
    val native_token_symbol: String
)

data class UpdateBlockchainRequest(
    val name: String?,
    val explorer_url: String?,
    val rpc_url: String?,
    val avg_gas_price_gwei: Double?
)

data class BlockchainListResponse(
    val data: List<Blockchain>
)

data class SystemStatusResponse(
    val services: List<SystemStatus>,
    val databases: List<SystemStatus>,
    val networks: List<SystemStatus>
)

data class SystemHealthResponse(
    val status: String,
    val uptime: String,
    val memory_usage: String,
    val cpu_usage: String,
    val disk_usage: String
)

data class SystemLogsResponse(
    val data: List<SystemLog>,
    val pagination: Pagination
)

data class SystemLog(
    val id: Long,
    val timestamp: String,
    val level: String,
    val service: String,
    val message: String,
    val metadata: Map<String, Any>?
)

data class UserAnalyticsResponse(
    val data: List<AnalyticsDataPoint>,
    val summary: UserAnalyticsSummary
)

data class AnalyticsDataPoint(
    val date: String,
    val value: Long
)

data class UserAnalyticsSummary(
    val total_new_users: Long,
    val total_active_users: Long,
    val avg_daily_active: Double,
    val retention_rate: Double
)

data class TransactionAnalyticsResponse(
    val data: List<AnalyticsDataPoint>,
    val summary: TransactionAnalyticsSummary
)

data class TransactionAnalyticsSummary(
    val total_transactions: Long,
    val total_volume: String,
    val avg_transaction_size: String
)

data class RevenueAnalyticsResponse(
    val data: List<RevenueDataPoint>,
    val summary: RevenueAnalyticsSummary
)

data class RevenueDataPoint(
    val date: String,
    val revenue: String,
    val breakdown: Map<String, String>
)

data class RevenueAnalyticsSummary(
    val total_revenue: String,
    val by_type: Map<String, String>
)

data class ReportResponse(
    val id: String,
    val url: String,
    val expires_at: String
)

data class NotificationListResponse(
    val data: List<AdminNotification>,
    val pagination: Pagination,
    val unread_count: Int
)

data class AdminNotification(
    val id: Long,
    val title: String,
    val message: String,
    val type: String,
    val read: Boolean,
    val created_at: String,
    val data: Map<String, Any>?
)

data class SendNotificationRequest(
    val user_id: Long,
    val title: String,
    val message: String,
    val type: String
)

data class SettingsResponse(
    val settings: Map<String, String>
)

data class UpdateSettingsRequest(
    val settings: Map<String, String>
)

data class SettingResponse(
    val key: String,
    val value: String
)

data class UpdateSettingRequest(
    val value: String
)

data class Pagination(
    val page: Int,
    val limit: Int,
    val total: Long,
    val total_pages: Int
)
