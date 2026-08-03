package com.tigerwallet.admin.data.api

import com.tigerwallet.admin.data.api.model.*
import retrofit2.Response
import retrofit2.http.*

interface ApiService {
    // Auth
    @POST("api/v1/auth/login")
    suspend fun login(@Body request: LoginRequest): Response<LoginResponse>

    @POST("api/v1/auth/logout")
    suspend fun logout(): Response<Unit>

    @POST("api/v1/auth/refresh")
    suspend fun refreshToken(@Body request: RefreshTokenRequest): Response<LoginResponse>

    @GET("api/v1/auth/me")
    suspend fun getCurrentAdmin(): Response<AdminResponse>

    // Users
    @GET("api/v1/users")
    suspend fun getUsers(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("status") status: String? = null,
        @Query("kyc_status") kycStatus: String? = null,
        @Query("search") search: String? = null
    ): Response<UsersResponse>

    @GET("api/v1/users/{id}")
    suspend fun getUser(@Path("id") id: String): Response<UserResponse>

    @PUT("api/v1/users/{id}")
    suspend fun updateUser(@Path("id") id: String, @Body request: UpdateUserRequest): Response<UserResponse>

    @POST("api/v1/users/{id}/suspend")
    suspend fun suspendUser(@Path("id") id: String, @Body request: SuspendRequest): Response<Unit>

    @POST("api/v1/users/{id}/ban")
    suspend fun banUser(@Path("id") id: String, @Body request: BanRequest): Response<Unit>

    // KYC
    @GET("api/v1/kyc")
    suspend fun getKYCSubmissions(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("status") status: String? = null,
        @Query("level") level: Int? = null
    ): Response<KYCListResponse>

    @GET("api/v1/kyc/{id}")
    suspend fun getKYC(@Path("id") id: String): Response<KYCResponse>

    @POST("api/v1/kyc/{id}/approve")
    suspend fun approveKYC(@Path("id") id: String, @Body request: ApproveKYCRequest): Response<Unit>

    @POST("api/v1/kyc/{id}/reject")
    suspend fun rejectKYC(@Path("id") id: String, @Body request: RejectKYCRequest): Response<Unit>

    // Tokens
    @GET("api/v1/tokens")
    suspend fun getTokens(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("status") status: String? = null,
        @Query("chain") chain: String? = null,
        @Query("search") search: String? = null
    ): Response<TokensResponse>

    @GET("api/v1/tokens/{id}")
    suspend fun getToken(@Path("id") id: String): Response<TokenResponse>

    @POST("api/v1/tokens")
    suspend fun createToken(@Body request: CreateTokenRequest): Response<TokenResponse>

    @PUT("api/v1/tokens/{id}")
    suspend fun updateToken(@Path("id") id: String, @Body request: UpdateTokenRequest): Response<TokenResponse>

    @DELETE("api/v1/tokens/{id}")
    suspend fun deleteToken(@Path("id") id: String): Response<Unit>

    @POST("api/v1/tokens/{id}/verify")
    suspend fun verifyToken(@Path("id") id: String): Response<Unit>

    // Pairs
    @GET("api/v1/pairs")
    suspend fun getPairs(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("status") status: String? = null,
        @Query("chain") chain: String? = null
    ): Response<PairsResponse>

    @GET("api/v1/pairs/{id}")
    suspend fun getPair(@Path("id") id: String): Response<PairResponse>

    @POST("api/v1/pairs")
    suspend fun createPair(@Body request: CreatePairRequest): Response<PairResponse>

    @PUT("api/v1/pairs/{id}")
    suspend fun updatePair(@Path("id") id: String, @Body request: UpdatePairRequest): Response<PairResponse>

    @DELETE("api/v1/pairs/{id}")
    suspend fun deletePair(@Path("id") id: String): Response<Unit>

    // Transactions
    @GET("api/v1/transactions")
    suspend fun getTransactions(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("status") status: String? = null,
        @Query("type") type: String? = null,
        @Query("user_id") userId: String? = null
    ): Response<TransactionsResponse>

    @GET("api/v1/transactions/{id}")
    suspend fun getTransaction(@Path("id") id: String): Response<TransactionResponse>

    // Withdrawals
    @GET("api/v1/withdrawals")
    suspend fun getWithdrawals(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("status") status: String? = null
    ): Response<WithdrawalsResponse>

    @POST("api/v1/withdrawals/{id}/approve")
    suspend fun approveWithdrawal(@Path("id") id: String): Response<Unit>

    @POST("api/v1/withdrawals/{id}/reject")
    suspend fun rejectWithdrawal(@Path("id") id: String, @Body request: RejectWithdrawalRequest): Response<Unit>

    // Chains
    @GET("api/v1/chains")
    suspend fun getChains(): Response<ChainsResponse>

    @GET("api/v1/chains/{id}")
    suspend fun getChain(@Path("id") id: String): Response<ChainResponse>

    @POST("api/v1/chains")
    suspend fun createChain(@Body request: CreateChainRequest): Response<ChainResponse>

    @PUT("api/v1/chains/{id}")
    suspend fun updateChain(@Path("id") id: String, @Body request: UpdateChainRequest): Response<ChainResponse>

    // Fees
    @GET("api/v1/fees")
    suspend fun getFees(): Response<FeesResponse>

    @POST("api/v1/fees")
    suspend fun createFee(@Body request: CreateFeeRequest): Response<FeeResponse>

    @PUT("api/v1/fees/{id}")
    suspend fun updateFee(@Path("id") id: String, @Body request: UpdateFeeRequest): Response<FeeResponse>

    // White Labels
    @GET("api/v1/white-labels")
    suspend fun getWhiteLabels(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("status") status: String? = null
    ): Response<WhiteLabelsResponse>

    @GET("api/v1/white-labels/{id}")
    suspend fun getWhiteLabel(@Path("id") id: String): Response<WhiteLabelResponse>

    @POST("api/v1/white-labels")
    suspend fun createWhiteLabel(@Body request: CreateWhiteLabelRequest): Response<WhiteLabelResponse>

    @PUT("api/v1/white-labels/{id}")
    suspend fun updateWhiteLabel(@Path("id") id: String, @Body request: UpdateWhiteLabelRequest): Response<WhiteLabelResponse>

    @POST("api/v1/white-labels/{id}/approve")
    suspend fun approveWhiteLabel(@Path("id") id: String): Response<Unit>

    @POST("api/v1/white-labels/{id}/suspend")
    suspend fun suspendWhiteLabel(@Path("id") id: String, @Body request: SuspendWhiteLabelRequest): Response<Unit>

    // Dashboard
    @GET("api/v1/dashboard")
    suspend fun getDashboardStats(): Response<DashboardResponse>

    @GET("api/v1/dashboard/analytics")
    suspend fun getAnalytics(@Query("period") period: String = "24h"): Response<AnalyticsResponse>

    // Admins
    @GET("api/v1/admins")
    suspend fun getAdmins(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("role") role: String? = null
    ): Response<AdminsResponse>

    @POST("api/v1/admins")
    suspend fun createAdmin(@Body request: CreateAdminRequest): Response<AdminResponse>

    @PUT("api/v1/admins/{id}")
    suspend fun updateAdmin(@Path("id") id: String, @Body request: UpdateAdminRequest): Response<AdminResponse>

    @DELETE("api/v1/admins/{id}")
    suspend fun deleteAdmin(@Path("id") id: String): Response<Unit>

    // Audit Logs
    @GET("api/v1/audit")
    suspend fun getAuditLogs(
        @Query("page") page: Int = 1,
        @Query("limit") limit: Int = 20,
        @Query("admin_id") adminId: String? = null,
        @Query("action") action: String? = null
    ): Response<AuditLogsResponse>
}
