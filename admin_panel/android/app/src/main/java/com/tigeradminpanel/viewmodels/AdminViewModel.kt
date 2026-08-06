package com.tigeradminpanel.viewmodels

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import com.tigeradminpanel.api.*

/**
 * TigerWallet Admin ViewModel - Complete Implementation
 * Manages all business logic for the admin panel
 */
class AdminViewModel(application: Application) : AndroidViewModel(application) {
    
    private val apiService = AdminApiService("https://api.tigerwallet.com")
    
    // Auth State
    private val _isLoggedIn = MutableLiveData<Boolean>(false)
    val isLoggedIn: LiveData<Boolean> = _isLoggedIn
    
    private val _currentAdmin = MutableLiveData<Admin?>()
    val currentAdmin: LiveData<Admin?> = _currentAdmin
    
    // Theme
    private val _isDarkMode = MutableLiveData<Boolean>(false)
    val isDarkMode: LiveData<Boolean> = _isDarkMode
    
    // Loading States
    private val _isLoading = MutableLiveData<Boolean>(false)
    val isLoading: LiveData<Boolean> = _isLoading
    
    // Error
    private val _error = MutableLiveData<String?>()
    val error: LiveData<String?> = _error
    
    // Dashboard Stats
    private val _dashboardStats = MutableLiveData<DashboardStats?>()
    val dashboardStats: LiveData<DashboardStats?> = _dashboardStats
    
    // Users
    private val _users = MutableLiveData<List<User>>(emptyList())
    val users: LiveData<List<User>> = _users
    
    private val _selectedUser = MutableLiveData<User?>()
    val selectedUser: LiveData<User?> = _selectedUser
    
    // KYC
    private val _kycRequests = MutableLiveData<List<KYCRequest>>(emptyList())
    val kycRequests: LiveData<List<KYCRequest>> = _kycRequests
    
    // Transactions
    private val _transactions = MutableLiveData<List<Transaction>>(emptyList())
    val transactions: LiveData<List<Transaction>> = _transactions
    
    // Withdrawals
    private val _withdrawals = MutableLiveData<List<Withdrawal>>(emptyList())
    val withdrawals: LiveData<List<Withdrawal>> = _withdrawals
    
    // Tokens
    private val _tokens = MutableLiveData<List<Token>>(emptyList())
    val tokens: LiveData<List<Token>> = _tokens
    
    // Fee Rules
    private val _feeRules = MutableLiveData<List<FeeRule>>(emptyList())
    val feeRules: LiveData<List<FeeRule>> = _feeRules
    
    // Bots
    private val _bots = MutableLiveData<List<Bot>>(emptyList())
    val bots: LiveData<List<Bot>> = _bots
    
    // Support Tickets
    private val _tickets = MutableLiveData<List<SupportTicket>>(emptyList())
    val tickets: LiveData<List<SupportTicket>> = _tickets
    
    // ==================== AUTH ====================
    
    fun login(email: String, password: String) {
        _isLoading.value = true
        _error.value = null
        
        apiService.login(email, password, object : ApiCallback<LoginResponse> {
            override fun onSuccess(data: LoginResponse) {
                _isLoggedIn.value = true
                _currentAdmin.value = data.admin
                _isLoading.value = false
                loadDashboardStats()
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun logout() {
        apiService.logout(object : ApiCallback<Unit> {
            override fun onSuccess(data: Unit) {
                _isLoggedIn.value = false
                _currentAdmin.value = null
                clearAllData()
            }
            
            override fun onError(error: String) {
                _isLoggedIn.value = false
                _currentAdmin.value = null
                clearAllData()
            }
        })
    }
    
    // ==================== THEME ====================
    
    fun toggleTheme() {
        _isDarkMode.value = !(_isDarkMode.value ?: false)
    }
    
    fun setDarkMode(enabled: Boolean) {
        _isDarkMode.value = enabled
    }
    
    // ==================== DASHBOARD ====================
    
    fun loadDashboardStats() {
        _isLoading.value = true
        apiService.getDashboardStats(object : ApiCallback<DashboardStats> {
            override fun onSuccess(data: DashboardStats) {
                _dashboardStats.value = data
                _isLoading.value = false
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun loadVolumeAnalytics(period: String = "7d") {
        apiService.getVolumeAnalytics(period, object : ApiCallback<VolumeAnalytics> {
            override fun onSuccess(data: VolumeAnalytics) {
                // Handle volume data
            }
            
            override fun onError(error: String) {
                _error.value = error
            }
        })
    }
    
    fun loadRevenueAnalytics(period: String = "30d") {
        apiService.getRevenueAnalytics(period, object : ApiCallback<RevenueAnalytics> {
            override fun onSuccess(data: RevenueAnalytics) {
                // Handle revenue data
            }
            
            override fun onError(error: String) {
                _error.value = error
            }
        })
    }
    
    // ==================== USERS ====================
    
    fun loadUsers(page: Int = 1, search: String? = null, status: String? = null, kycStatus: String? = null) {
        _isLoading.value = true
        apiService.getUsers(page, 20, search, status, kycStatus, object : ApiCallback<ListResponse<User>> {
            override fun onSuccess(data: ListResponse<User>) {
                _users.value = data.data
                _isLoading.value = false
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun loadUserDetails(userId: String) {
        _isLoading.value = true
        apiService.getUser(userId, object : ApiCallback<User> {
            override fun onSuccess(data: User) {
                _selectedUser.value = data
                _isLoading.value = false
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun banUser(userId: String, reason: String) {
        _isLoading.value = true
        apiService.banUser(userId, reason, object : ApiCallback<User> {
            override fun onSuccess(data: User) {
                _selectedUser.value = data
                _isLoading.value = false
                loadUsers()
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun unbanUser(userId: String) {
        _isLoading.value = true
        apiService.unbanUser(userId, object : ApiCallback<User> {
            override fun onSuccess(data: User) {
                _selectedUser.value = data
                _isLoading.value = false
                loadUsers()
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    // ==================== KYC ====================
    
    fun loadKYCRequests(status: String? = null) {
        _isLoading.value = true
        apiService.getKYCRequests(1, status, object : ApiCallback<ListResponse<KYCRequest>> {
            override fun onSuccess(data: ListResponse<KYCRequest>) {
                _kycRequests.value = data.data
                _isLoading.value = false
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun approveKYC(kycId: String) {
        _isLoading.value = true
        apiService.approveKYC(kycId, object : ApiCallback<KYCRequest> {
            override fun onSuccess(data: KYCRequest) {
                _isLoading.value = false
                loadKYCRequests()
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun rejectKYC(kycId: String, reason: String) {
        _isLoading.value = true
        apiService.rejectKYC(kycId, reason, object : ApiCallback<KYCRequest> {
            override fun onSuccess(data: KYCRequest) {
                _isLoading.value = false
                loadKYCRequests()
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    // ==================== TRANSACTIONS ====================
    
    fun loadTransactions(status: String? = null, token: String? = null, chain: String? = null) {
        _isLoading.value = true
        apiService.getTransactions(1, status, token, chain, object : ApiCallback<ListResponse<Transaction>> {
            override fun onSuccess(data: ListResponse<Transaction>) {
                _transactions.value = data.data
                _isLoading.value = false
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun getTransactionDetails(txId: String) {
        _isLoading.value = true
        apiService.getTransaction(txId, object : ApiCallback<Transaction> {
            override fun onSuccess(data: Transaction) {
                _isLoading.value = false
                // Handle transaction details
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    // ==================== WITHDRAWALS ====================
    
    fun loadWithdrawals(status: String? = null) {
        _isLoading.value = true
        apiService.getWithdrawals(1, status, object : ApiCallback<ListResponse<Withdrawal>> {
            override fun onSuccess(data: ListResponse<Withdrawal>) {
                _withdrawals.value = data.data
                _isLoading.value = false
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun approveWithdrawal(withdrawalId: String) {
        _isLoading.value = true
        apiService.approveWithdrawal(withdrawalId, object : ApiCallback<Withdrawal> {
            override fun onSuccess(data: Withdrawal) {
                _isLoading.value = false
                loadWithdrawals()
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun rejectWithdrawal(withdrawalId: String, reason: String) {
        _isLoading.value = true
        apiService.rejectWithdrawal(withdrawalId, reason, object : ApiCallback<Withdrawal> {
            override fun onSuccess(data: Withdrawal) {
                _isLoading.value = false
                loadWithdrawals()
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    // ==================== TOKENS ====================
    
    fun loadTokens() {
        _isLoading.value = true
        apiService.getTokens(object : ApiCallback<ListResponse<Token>> {
            override fun onSuccess(data: ListResponse<Token>) {
                _tokens.value = data.data
                _isLoading.value = false
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun createToken(name: String, symbol: String, decimals: Int, contractAddress: String) {
        _isLoading.value = true
        val tokenData = org.json.JSONObject().apply {
            put("name", name)
            put("symbol", symbol)
            put("decimals", decimals)
            put("contract_address", contractAddress)
        }
        
        apiService.createToken(tokenData, object : ApiCallback<Token> {
            override fun onSuccess(data: Token) {
                _isLoading.value = false
                loadTokens()
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun toggleToken(tokenId: String, activate: Boolean) {
        _isLoading.value = true
        val callback = object : ApiCallback<Token> {
            override fun onSuccess(data: Token) {
                _isLoading.value = false
                loadTokens()
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        }
        
        if (activate) {
            apiService.activateToken(tokenId, callback)
        } else {
            apiService.deactivateToken(tokenId, callback)
        }
    }
    
    // ==================== FEES ====================
    
    fun loadFeeRules() {
        _isLoading.value = true
        apiService.getFeeRules(object : ApiCallback<ListResponse<FeeRule>> {
            override fun onSuccess(data: ListResponse<FeeRule>) {
                _feeRules.value = data.data
                _isLoading.value = false
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun createFeeRule(name: String, feeType: String, feeValue: Double) {
        _isLoading.value = true
        val feeData = org.json.JSONObject().apply {
            put("name", name)
            put("fee_type", feeType)
            put("fee_value", feeValue)
        }
        
        apiService.createFeeRule(feeData, object : ApiCallback<FeeRule> {
            override fun onSuccess(data: FeeRule) {
                _isLoading.value = false
                loadFeeRules()
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun deleteFeeRule(feeId: String) {
        _isLoading.value = true
        apiService.deleteFeeRule(feeId, object : ApiCallback<Unit> {
            override fun onSuccess(data: Unit) {
                _isLoading.value = false
                loadFeeRules()
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    // ==================== BOTS ====================
    
    fun loadBots() {
        _isLoading.value = true
        apiService.getBots(object : ApiCallback<ListResponse<Bot>> {
            override fun onSuccess(data: ListResponse<Bot>) {
                _bots.value = data.data
                _isLoading.value = false
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun createBot(name: String, botType: String, config: org.json.JSONObject) {
        _isLoading.value = true
        config.put("name", name)
        config.put("bot_type", botType)
        
        apiService.createBot(config, object : ApiCallback<Bot> {
            override fun onSuccess(data: Bot) {
                _isLoading.value = false
                loadBots()
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun startBot(botId: String) {
        _isLoading.value = true
        apiService.startBot(botId, object : ApiCallback<Bot> {
            override fun onSuccess(data: Bot) {
                _isLoading.value = false
                loadBots()
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun stopBot(botId: String) {
        _isLoading.value = true
        apiService.stopBot(botId, object : ApiCallback<Bot> {
            override fun onSuccess(data: Bot) {
                _isLoading.value = false
                loadBots()
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    // ==================== SUPPORT ====================
    
    fun loadTickets(status: String? = null) {
        _isLoading.value = true
        apiService.getTickets(status, object : ApiCallback<ListResponse<SupportTicket>> {
            override fun onSuccess(data: ListResponse<SupportTicket>) {
                _tickets.value = data.data
                _isLoading.value = false
            }
            
            override fun onError(error: String) {
                _error.value = error
                _isLoading.value = false
            }
        })
    }
    
    fun addTicketMessage(ticketId: String, message: String, isInternal: Boolean = false) {
        apiService.addTicketMessage(ticketId, message, isInternal, object : ApiCallback<TicketMessage> {
            override fun onSuccess(data: TicketMessage) {
                loadTickets()
            }
            
            override fun onError(error: String) {
                _error.value = error
            }
        })
    }
    
    fun closeTicket(ticketId: String) {
        apiService.closeTicket(ticketId, object : ApiCallback<SupportTicket> {
            override fun onSuccess(data: SupportTicket) {
                loadTickets()
            }
            
            override fun onError(error: String) {
                _error.value = error
            }
        })
    }
    
    // ==================== NOTIFICATIONS ====================
    
    fun sendNotification(title: String, message: String, userId: Int? = null) {
        apiService.sendNotification(title, message, userId, false, false, object : ApiCallback<Unit> {
            override fun onSuccess(data: Unit) {
                // Notification sent
            }
            
            override fun onError(error: String) {
                _error.value = error
            }
        })
    }
    
    fun broadcastNotification(title: String, message: String) {
        apiService.broadcastNotification(title, message, object : ApiCallback<BroadcastResponse> {
            override fun onSuccess(data: BroadcastResponse) {
                // Broadcast completed
            }
            
            override fun onError(error: String) {
                _error.value = error
            }
        })
    }
    
    // ==================== HELPERS ====================
    
    private fun clearAllData() {
        _users.value = emptyList()
        _selectedUser.value = null
        _kycRequests.value = emptyList()
        _transactions.value = emptyList()
        _withdrawals.value = emptyList()
        _tokens.value = emptyList()
        _feeRules.value = emptyList()
        _bots.value = emptyList()
        _tickets.value = emptyList()
        _dashboardStats.value = null
    }
    
    fun clearError() {
        _error.value = null
    }
}
