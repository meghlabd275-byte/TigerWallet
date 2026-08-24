package com.tigerwallet.admin.data.repository

import com.tigerwallet.admin.data.api.AdminApiService
import com.tigerwallet.admin.data.model.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * Admin Repository
 * Handles all admin-related data operations
 */
class AdminRepository(private val apiService: AdminApiService) {

    suspend fun login(email: String, password: String): Result<LoginResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.login(LoginRequest(email, password))
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception(response.errorBody()?.string() ?: "Login failed"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun logout(): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.logout()
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Logout failed"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getAdmins(page: Int = 1, limit: Int = 20, role: String? = null): Result<AdminListResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getAdmins(page, limit, role)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch admins"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getAdmin(id: Long): Result<AdminUser> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getAdmin(id)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch admin"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createAdmin(request: CreateAdminRequest): Result<AdminUser> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.createAdmin(request)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to create admin"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun updateAdmin(id: Long, request: UpdateAdminRequest): Result<AdminUser> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.updateAdmin(id, request)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to update admin"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deleteAdmin(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.deleteAdmin(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to delete admin"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun suspendAdmin(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.suspendAdmin(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to suspend admin"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun activateAdmin(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.activateAdmin(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to activate admin"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getCurrentAdmin(): Result<AdminUser> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getCurrentAdmin()
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch current admin"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getAdminActivity(
        page: Int = 1,
        limit: Int = 50,
        adminId: Long? = null,
        action: String? = null
    ): Result<AdminActivityResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getAdminActivity(page, limit, adminId, action)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch admin activity"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

/**
 * User Repository
 * Handles all user-related data operations
 */
class UserRepository(private val apiService: AdminApiService) {

    suspend fun getUsers(
        page: Int = 1,
        limit: Int = 20,
        status: String? = null,
        kycStatus: String? = null,
        search: String? = null
    ): Result<UserListResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getUsers(page, limit, status, kycStatus, search = search)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch users"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getUser(id: Long): Result<PlatformUser> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getUser(id)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch user"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun updateUser(id: Long, request: UpdateUserRequest): Result<PlatformUser> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.updateUser(id, request)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to update user"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun suspendUser(id: Long, reason: String, durationDays: Int? = null): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.suspendUser(id, SuspendUserRequest(reason, durationDays))
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to suspend user"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun banUser(id: Long, reason: String, permanent: Boolean = true): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.banUser(id, BanUserRequest(reason, permanent))
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to ban user"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun activateUser(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.activateUser(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to activate user"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getUserStats(): Result<UserStatsResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getUserStats()
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch user stats"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

/**
 * Transaction Repository
 * Handles all transaction-related data operations
 */
class TransactionRepository(private val apiService: AdminApiService) {

    suspend fun getTransactions(
        page: Int = 1,
        limit: Int = 20,
        status: String? = null,
        type: String? = null,
        chain: String? = null,
        flagged: Boolean? = null,
        search: String? = null
    ): Result<TransactionListResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getTransactions(page, limit, status, type, chain, flagged = flagged, search = search)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch transactions"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getTransaction(id: Long): Result<Transaction> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getTransaction(id)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch transaction"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun flagTransaction(id: Long, reason: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.flagTransaction(id, FlagTransactionRequest(reason))
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to flag transaction"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun unflagTransaction(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.unflagTransaction(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to unflag transaction"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getTransactionStats(): Result<TransactionStatsResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getTransactionStats()
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch transaction stats"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

/**
 * KYC Repository
 * Handles all KYC-related data operations
 */
class KYCRepository(private val apiService: AdminApiService) {

    suspend fun getKYCApplications(
        page: Int = 1,
        limit: Int = 20,
        status: String? = null,
        level: Int? = null
    ): Result<KYCListResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getKYCApplications(page, limit, status, level)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch KYC applications"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getKYCApplication(id: Long): Result<KYCApplication> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getKYCApplication(id)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch KYC application"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun approveKYC(id: Long, notes: String? = null): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.approveKYC(id, ApproveKYCRequest(notes))
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to approve KYC"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun rejectKYC(id: Long, reason: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.rejectKYC(id, RejectKYCRequest(reason))
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to reject KYC"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getKYCStats(): Result<KYCStatsResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getKYCStats()
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch KYC stats"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

/**
 * Token Repository
 * Handles all token-related data operations
 */
class TokenRepository(private val apiService: AdminApiService) {

    suspend fun getTokens(
        page: Int = 1,
        limit: Int = 20,
        chain: String? = null,
        isActive: Boolean? = null,
        search: String? = null
    ): Result<TokenListResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getTokens(page, limit, chain, isActive, search = search)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch tokens"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getToken(id: Long): Result<Token> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getToken(id)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch token"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createToken(request: CreateTokenRequest): Result<Token> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.createToken(request)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to create token"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun updateToken(id: Long, request: UpdateTokenRequest): Result<Token> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.updateToken(id, request)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to update token"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun verifyToken(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.verifyToken(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to verify token"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun activateToken(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.activateToken(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to activate token"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deactivateToken(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.deactivateToken(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to deactivate token"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getTokenListings(
        page: Int = 1,
        limit: Int = 20,
        status: String? = null
    ): Result<TokenListingListResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getTokenListings(page, limit, status)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch token listings"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun approveTokenListing(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.approveTokenListing(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to approve token listing"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun rejectTokenListing(id: Long, reason: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.rejectTokenListing(id, RejectListingRequest(reason))
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to reject token listing"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

/**
 * Withdrawal Repository
 * Handles all withdrawal-related data operations
 */
class WithdrawalRepository(private val apiService: AdminApiService) {

    suspend fun getWithdrawals(
        page: Int = 1,
        limit: Int = 20,
        status: String? = null,
        token: String? = null,
        chain: String? = null
    ): Result<WithdrawalListResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getWithdrawals(page, limit, status, token, chain)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch withdrawals"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getWithdrawal(id: Long): Result<WithdrawalRequest> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getWithdrawal(id)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch withdrawal"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun approveWithdrawal(id: Long, notes: String? = null): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.approveWithdrawal(id, ApproveWithdrawalRequest(notes))
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to approve withdrawal"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun rejectWithdrawal(id: Long, reason: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.rejectWithdrawal(id, RejectWithdrawalRequest(reason))
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to reject withdrawal"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun processWithdrawal(id: Long, txHash: String, notes: String? = null): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.processWithdrawal(id, ProcessWithdrawalRequest(txHash, notes))
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to process withdrawal"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getWithdrawalStats(): Result<WithdrawalStatsResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getWithdrawalStats()
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch withdrawal stats"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

/**
 * White Label Repository
 * Handles all white label-related data operations
 */
class WhiteLabelRepository(private val apiService: AdminApiService) {

    suspend fun getWhiteLabels(
        page: Int = 1,
        limit: Int = 20,
        status: String? = null,
        search: String? = null
    ): Result<WhiteLabelListResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getWhiteLabels(page, limit, status, search)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch white labels"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getWhiteLabel(id: Long): Result<WhiteLabel> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getWhiteLabel(id)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch white label"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createWhiteLabel(request: CreateWhiteLabelRequest): Result<WhiteLabel> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.createWhiteLabel(request)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to create white label"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun updateWhiteLabel(id: Long, request: UpdateWhiteLabelRequest): Result<WhiteLabel> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.updateWhiteLabel(id, request)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to update white label"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun activateWhiteLabel(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.activateWhiteLabel(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to activate white label"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun suspendWhiteLabel(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.suspendWhiteLabel(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to suspend white label"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

/**
 * Analytics Repository
 * Handles all analytics-related data operations
 */
class AnalyticsRepository(private val apiService: AdminApiService) {

    suspend fun getAnalyticsOverview(): Result<AnalyticsData> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getAnalyticsOverview()
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch analytics overview"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getUserAnalytics(
        fromDate: String? = null,
        toDate: String? = null,
        groupBy: String? = null
    ): Result<UserAnalyticsResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getUserAnalytics(fromDate, toDate, groupBy)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch user analytics"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getTransactionAnalytics(
        fromDate: String? = null,
        toDate: String? = null,
        groupBy: String? = null
    ): Result<TransactionAnalyticsResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getTransactionAnalytics(fromDate, toDate, groupBy)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch transaction analytics"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getRevenueAnalytics(
        fromDate: String? = null,
        toDate: String? = null,
        groupBy: String? = null
    ): Result<RevenueAnalyticsResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getRevenueAnalytics(fromDate, toDate, groupBy)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch revenue analytics"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun generateReport(
        type: String,
        fromDate: String,
        toDate: String,
        format: String = "json"
    ): Result<ReportResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.generateReport(type, fromDate, toDate, format)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to generate report"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

/**
 * System Repository
 * Handles all system-related data operations
 */
class SystemRepository(private val apiService: AdminApiService) {

    suspend fun getSystemStatus(): Result<SystemStatusResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getSystemStatus()
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch system status"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getSystemHealth(): Result<SystemHealthResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getSystemHealth()
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch system health"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getSystemLogs(
        page: Int = 1,
        limit: Int = 50,
        service: String? = null,
        level: String? = null
    ): Result<SystemLogsResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getSystemLogs(page, limit, service, level)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch system logs"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

/**
 * Bot Repository
 * Handles all bot-related data operations
 */
class BotRepository(private val apiService: AdminApiService) {

    suspend fun getBots(
        page: Int = 1,
        limit: Int = 20,
        status: String? = null,
        botType: String? = null
    ): Result<BotListResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getBots(page, limit, status, botType)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch bots"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getBot(id: Long): Result<BotInstance> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getBot(id)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch bot"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun startBot(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.startBot(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to start bot"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun stopBot(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.stopBot(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to stop bot"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun pauseBot(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.pauseBot(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to pause bot"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getBotStats(): Result<BotStatsResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getBotStats()
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch bot stats"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

/**
 * Fee Repository
 * Handles all fee-related data operations
 */
class FeeRepository(private val apiService: AdminApiService) {

    suspend fun getFeeConfigs(): Result<FeeConfigListResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getFeeConfigs()
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch fee configs"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createFeeConfig(request: CreateFeeConfigRequest): Result<FeeConfig> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.createFeeConfig(request)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to create fee config"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun updateFeeConfig(id: Long, request: UpdateFeeConfigRequest): Result<FeeConfig> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.updateFeeConfig(id, request)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to update fee config"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deleteFeeConfig(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.deleteFeeConfig(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to delete fee config"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun activateFeeConfig(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.activateFeeConfig(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to activate fee config"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deactivateFeeConfig(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.deactivateFeeConfig(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to deactivate fee config"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

/**
 * API Key Repository
 * Handles all API key-related data operations
 */
class APIKeyRepository(private val apiService: AdminApiService) {

    suspend fun getAPIKeys(): Result<APIKeyListResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getAPIKeys()
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch API keys"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createAPIKey(request: CreateAPIKeyRequest): Result<APIKey> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.createAPIKey(request)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to create API key"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deleteAPIKey(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.deleteAPIKey(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to delete API key"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun revokeAPIKey(id: Long): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.revokeAPIKey(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to revoke API key"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

/**
 * Blockchain Repository
 * Handles all blockchain-related data operations
 */
class BlockchainRepository(private val apiService: AdminApiService) {

    suspend fun getBlockchains(): Result<BlockchainListResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getBlockchains()
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch blockchains"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getBlockchain(id: String): Result<Blockchain> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getBlockchain(id)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to fetch blockchain"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createBlockchain(request: CreateBlockchainRequest): Result<Blockchain> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.createBlockchain(request)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to create blockchain"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun updateBlockchain(id: String, request: UpdateBlockchainRequest): Result<Blockchain> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.updateBlockchain(id, request)
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to update blockchain"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun activateBlockchain(id: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.activateBlockchain(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to activate blockchain"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deactivateBlockchain(id: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.deactivateBlockchain(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Failed to deactivate blockchain"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
