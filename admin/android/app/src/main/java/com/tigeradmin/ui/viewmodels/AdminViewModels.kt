package com.tigeradmin.ui.viewmodels

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.tigeradmin.data.model.*
import com.tigeradmin.data.repository.*
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch

/**
 * ViewModel for Dashboard
 */
class DashboardViewModel(
    private val analyticsRepository: AnalyticsRepository
) : ViewModel() {

    private val _analyticsData = MutableStateFlow<AnalyticsData?>(null)
    val analyticsData: StateFlow<AnalyticsData?> = _analyticsData

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error

    init {
        loadDashboardData()
    }

    fun loadDashboardData() {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            
            try {
                val result = analyticsRepository.getAnalyticsOverview()
                result.onSuccess { data ->
                    _analyticsData.value = data
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
}

/**
 * ViewModel for Users Management
 */
class UsersViewModel(
    private val userRepository: UserRepository
) : ViewModel() {

    private val _users = MutableStateFlow<List<PlatformUser>>(emptyList())
    val users: StateFlow<List<PlatformUser>> = _users

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error

    private var currentStatusFilter: String? = null
    private var currentKycFilter: String? = null
    private var currentSearchQuery: String? = null

    init {
        loadUsers()
    }

    fun loadUsers(status: String? = currentStatusFilter, kycStatus: String? = currentKycFilter, search: String? = currentSearchQuery) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            
            try {
                val result = userRepository.getUsers(status = status, kycStatus = kycStatus, search = search)
                result.onSuccess { response ->
                    _users.value = response.data
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun suspendUser(userId: Long, reason: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val result = userRepository.suspendUser(userId, reason)
                result.onSuccess {
                    loadUsers() // Refresh list
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun banUser(userId: Long, reason: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val result = userRepository.banUser(userId, reason, true)
                result.onSuccess {
                    loadUsers() // Refresh list
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun activateUser(userId: Long) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val result = userRepository.activateUser(userId)
                result.onSuccess {
                    loadUsers() // Refresh list
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun setStatusFilter(status: String?) {
        currentStatusFilter = status
        loadUsers()
    }

    fun setKycFilter(kycStatus: String?) {
        currentKycFilter = kycStatus
        loadUsers()
    }

    fun setSearchQuery(query: String?) {
        currentSearchQuery = query
        loadUsers()
    }
}

/**
 * ViewModel for Transactions
 */
class TransactionsViewModel(
    private val transactionRepository: TransactionRepository
) : ViewModel() {

    private val _transactions = MutableStateFlow<List<Transaction>>(emptyList())
    val transactions: StateFlow<List<Transaction>> = _transactions

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error

    init {
        loadTransactions()
    }

    fun loadTransactions(status: String? = null, flagged: Boolean? = null) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            
            try {
                val result = transactionRepository.getTransactions(status = status, flagged = flagged)
                result.onSuccess { response ->
                    _transactions.value = response.data
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun flagTransaction(txId: Long, reason: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val result = transactionRepository.flagTransaction(txId, reason)
                result.onSuccess {
                    loadTransactions()
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun unflagTransaction(txId: Long) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val result = transactionRepository.unflagTransaction(txId)
                result.onSuccess {
                    loadTransactions()
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
}

/**
 * ViewModel for KYC Management
 */
class KYCViewModel(
    private val kycRepository: KYCRepository
) : ViewModel() {

    private val _applications = MutableStateFlow<List<KYCApplication>>(emptyList())
    val applications: StateFlow<List<KYCApplication>> = _applications

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error

    init {
        loadKYCApplications()
    }

    fun loadKYCApplications(status: String? = null) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            
            try {
                val result = kycRepository.getKYCApplications(status = status)
                result.onSuccess { response ->
                    _applications.value = response.data
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun approveKYC(kycId: Long) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val result = kycRepository.approveKYC(kycId)
                result.onSuccess {
                    loadKYCApplications()
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun rejectKYC(kycId: Long, reason: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val result = kycRepository.rejectKYC(kycId, reason)
                result.onSuccess {
                    loadKYCApplications()
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
}

/**
 * ViewModel for Tokens
 */
class TokensViewModel(
    private val tokenRepository: TokenRepository
) : ViewModel() {

    private val _tokens = MutableStateFlow<List<Token>>(emptyList())
    val tokens: StateFlow<List<Token>> = _tokens

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error

    init {
        loadTokens()
    }

    fun loadTokens(chain: String? = null, isActive: Boolean? = null) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            
            try {
                val result = tokenRepository.getTokens(chain = chain, isActive = isActive)
                result.onSuccess { response ->
                    _tokens.value = response.data
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun activateToken(tokenId: Long) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val result = tokenRepository.activateToken(tokenId)
                result.onSuccess {
                    loadTokens()
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun deactivateToken(tokenId: Long) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val result = tokenRepository.deactivateToken(tokenId)
                result.onSuccess {
                    loadTokens()
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun verifyToken(tokenId: Long) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val result = tokenRepository.verifyToken(tokenId)
                result.onSuccess {
                    loadTokens()
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
}

/**
 * ViewModel for Withdrawals
 */
class WithdrawalsViewModel(
    private val withdrawalRepository: WithdrawalRepository
) : ViewModel() {

    private val _withdrawals = MutableStateFlow<List<WithdrawalRequest>>(emptyList())
    val withdrawals: StateFlow<List<WithdrawalRequest>> = _withdrawals

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error

    init {
        loadWithdrawals()
    }

    fun loadWithdrawals(status: String? = null) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            
            try {
                val result = withdrawalRepository.getWithdrawals(status = status)
                result.onSuccess { response ->
                    _withdrawals.value = response.data
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun approveWithdrawal(withdrawalId: Long) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val result = withdrawalRepository.approveWithdrawal(withdrawalId)
                result.onSuccess {
                    loadWithdrawals()
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun rejectWithdrawal(withdrawalId: Long, reason: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val result = withdrawalRepository.rejectWithdrawal(withdrawalId, reason)
                result.onSuccess {
                    loadWithdrawals()
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun processWithdrawal(withdrawalId: Long, txHash: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val result = withdrawalRepository.processWithdrawal(withdrawalId, txHash)
                result.onSuccess {
                    loadWithdrawals()
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
}

/**
 * ViewModel for System Status
 */
class SystemViewModel(
    private val systemRepository: SystemRepository
) : ViewModel() {

    private val _systemStatus = MutableStateFlow<List<SystemStatus>>(emptyList())
    val systemStatus: StateFlow<List<SystemStatus>> = _systemStatus

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error

    init {
        loadSystemStatus()
    }

    fun loadSystemStatus() {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            
            try {
                val result = systemRepository.getSystemStatus()
                result.onSuccess { status ->
                    _systemStatus.value = status.services + status.databases + status.networks
                }.onFailure { e ->
                    _error.value = e.message
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
}

/**
 * ViewModel Factory
 */
class AdminViewModelFactory(
    private val apiService: com.tigeradmin.data.api.AdminApiService
) : ViewModelProvider.Factory {
    
    @Suppress("UNCHECKED_CAST")
    override fun <T : ViewModel> create(modelClass: Class<T>): T {
        return when {
            modelClass.isAssignableFrom(DashboardViewModel::class.java) -> {
                DashboardViewModel(AnalyticsRepository(apiService)) as T
            }
            modelClass.isAssignableFrom(UsersViewModel::class.java) -> {
                UsersViewModel(UserRepository(apiService)) as T
            }
            modelClass.isAssignableFrom(TransactionsViewModel::class.java) -> {
                TransactionsViewModel(TransactionRepository(apiService)) as T
            }
            modelClass.isAssignableFrom(KYCViewModel::class.java) -> {
                KYCViewModel(KYCRepository(apiService)) as T
            }
            modelClass.isAssignableFrom(TokensViewModel::class.java) -> {
                TokensViewModel(TokenRepository(apiService)) as T
            }
            modelClass.isAssignableFrom(WithdrawalsViewModel::class.java) -> {
                WithdrawalsViewModel(WithdrawalRepository(apiService)) as T
            }
            modelClass.isAssignableFrom(SystemViewModel::class.java) -> {
                SystemViewModel(SystemRepository(apiService)) as T
            }
            else -> throw IllegalArgumentException("Unknown ViewModel class: ${modelClass.name}")
        }
    }
}
