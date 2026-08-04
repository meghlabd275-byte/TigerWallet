package com.tigermaster

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.net.HttpURLConnection
import java.net.URL
import org.json.JSONArray
import org.json.JSONObject

// ============================================================================
// MasterWallet API Configuration
// ============================================================================
object ApiConfig {
    const val BASE_URL = "https://api.tigerwallet.io/master"
    const val API_VERSION = "/api/v1"
    
    // Supported Networks (103+ chains)
    val NETWORKS = mapOf(
        1L to "Ethereum",
        56L to "BNB Chain",
        137L to "Polygon",
        42161L to "Arbitrum",
        10L to "Optimism",
        43114L to "Avalanche",
        8453L to "Base",
        324L to "zkSync Era",
        59144L to "Linea",
        534352L to "Scroll"
    )
}

class MasterWalletViewModel : ViewModel() {
    private val _masterWallet = MutableStateFlow<MasterWalletData?>(null)
    val masterWallet: StateFlow<MasterWalletData?> = _masterWallet.asStateFlow()
    
    private val _subWallets = MutableStateFlow<List<SubWalletData>>(emptyList())
    val subWallets: StateFlow<List<SubWalletData>> = _subWallets.asStateFlow()
    
    private val _transactions = MutableStateFlow<List<TransactionData>>(emptyList())
    val transactions: StateFlow<List<TransactionData>> = _transactions.asStateFlow()
    
    private val _autoSignRules = MutableStateFlow<List<AutoSignRuleData>>(emptyList())
    val autoSignRules: StateFlow<List<AutoSignRuleData>> = _autoSignRules.asStateFlow()
    
    private val _users = MutableStateFlow<List<UserData>>(emptyList())
    val users: StateFlow<List<UserData>> = _users.asStateFlow()
    
    private val _networks = MutableStateFlow<List<NetworkData>>(emptyList())
    val networks: StateFlow<List<NetworkData>> = _networks.asStateFlow()
    
    private val _tokens = MutableStateFlow<List<TokenData>>(emptyList())
    val tokens: StateFlow<List<TokenData>> = _tokens.asStateFlow()
    
    private val _volumeStats = MutableStateFlow<VolumeStatsData?>(null)
    val volumeStats: StateFlow<VolumeStatsData?> = _volumeStats.asStateFlow()
    
    private val _whitelist = MutableStateFlow<List<WhitelistEntry>>(emptyList())
    val whitelist: StateFlow<List<WhitelistEntry>> = _whitelist.asStateFlow()
    
    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()
    
    private val _isDarkMode = MutableStateFlow(false)
    val isDarkMode: StateFlow<Boolean> = _isDarkMode.asStateFlow()
    
    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error.asStateFlow()
    
    private var authToken: String? = null
    
    init {
        loadData()
    }
    
    fun setAuthToken(token: String) {
        authToken = token
    }
    
    fun toggleDarkMode() {
        _isDarkMode.value = !_isDarkMode.value
    }
    
    // ============================================================================
    // Real API Calls
    // ============================================================================
    
    private fun loadData() {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                // Load all data from API
                loadMasterWallet()
                loadSubWallets()
                loadTransactions()
                loadAutoSignRules()
                loadUsers()
                loadNetworks()
                loadTokens()
                loadVolumeStats()
                loadWhitelist()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    private suspend fun loadMasterWallet() = withContext(Dispatchers.IO) {
        try {
            val response = apiGet("/master/wallet")
            if (response != null) {
                val json = JSONObject(response)
                _masterWallet.value = MasterWalletData(
                    address = json.optString("address", ""),
                    totalVolume = json.optString("totalVolume", "0"),
                    subWalletCount = json.optInt("subWalletCount", 0),
                    userCount = json.optInt("userCount", 0),
                    pendingTx = json.optInt("pendingTx", 0)
                )
            }
        } catch (e: Exception) {
            _error.value = "Failed to load master wallet: ${e.message}"
        }
    }
    
    private suspend fun loadSubWallets() = withContext(Dispatchers.IO) {
        try {
            val response = apiGet("/subwallets")
            if (response != null) {
                val jsonArray = JSONArray(response)
                val list = mutableListOf<SubWalletData>()
                for (i in 0 until jsonArray.length()) {
                    val obj = jsonArray.getJSONObject(i)
                    list.add(SubWalletData(
                        name = obj.optString("name", ""),
                        address = obj.optString("address", ""),
                        balance = obj.optString("balance", "0"),
                        status = obj.optString("status", "Active")
                    ))
                }
                _subWallets.value = list
            }
        } catch (e: Exception) {
            _error.value = "Failed to load subwallets: ${e.message}"
        }
    }
    
    private suspend fun loadTransactions() = withContext(Dispatchers.IO) {
        try {
            val response = apiGet("/transactions")
            if (response != null) {
                val jsonArray = JSONArray(response)
                val list = mutableListOf<TransactionData>()
                for (i in 0 until jsonArray.length()) {
                    val obj = jsonArray.getJSONObject(i)
                    list.add(TransactionData(
                        id = obj.optString("id", ""),
                        hash = obj.optString("hash", ""),
                        from = obj.optString("from", ""),
                        to = obj.optString("to", ""),
                        amount = obj.optString("amount", "0"),
                        token = obj.optString("token", ""),
                        chain = obj.optString("chain", ""),
                        status = obj.optString("status", "Pending"),
                        timestamp = obj.optLong("timestamp", System.currentTimeMillis())
                    ))
                }
                _transactions.value = list
            }
        } catch (e: Exception) {
            _error.value = "Failed to load transactions: ${e.message}"
        }
    }
    
    private suspend fun loadAutoSignRules() = withContext(Dispatchers.IO) {
        try {
            val response = apiGet("/auto-sign/rules")
            if (response != null) {
                val jsonArray = JSONArray(response)
                val list = mutableListOf<AutoSignRuleData>()
                for (i in 0 until jsonArray.length()) {
                    val obj = jsonArray.getJSONObject(i)
                    list.add(AutoSignRuleData(
                        id = obj.optString("id", ""),
                        name = obj.optString("name", ""),
                        maxAmount = obj.optString("maxAmount", "0"),
                        chain = obj.optString("chain", ""),
                        enabled = obj.optBoolean("enabled", false)
                    ))
                }
                _autoSignRules.value = list
            }
        } catch (e: Exception) {
            _error.value = "Failed to load auto-sign rules: ${e.message}"
        }
    }
    
    private suspend fun loadUsers() = withContext(Dispatchers.IO) {
        try {
            val response = apiGet("/users")
            if (response != null) {
                val jsonArray = JSONArray(response)
                val list = mutableListOf<UserData>()
                for (i in 0 until jsonArray.length()) {
                    val obj = jsonArray.getJSONObject(i)
                    list.add(UserData(
                        id = obj.optString("id", ""),
                        email = obj.optString("email", ""),
                        name = obj.optString("name", ""),
                        role = obj.optString("role", ""),
                        walletAddress = obj.optString("walletAddress", ""),
                        permissions = obj.optJSONArray("permissions")?.let { arr ->
                            (0 until arr.length()).map { arr.getString(it) }
                        } ?: emptyList()
                    ))
                }
                _users.value = list
            }
        } catch (e: Exception) {
            _error.value = "Failed to load users: ${e.message}"
        }
    }
    
    private suspend fun loadNetworks() = withContext(Dispatchers.IO) {
        try {
            val response = apiGet("/networks")
            if (response != null) {
                val jsonArray = JSONArray(response)
                val list = mutableListOf<NetworkData>()
                for (i in 0 until jsonArray.length()) {
                    val obj = jsonArray.getJSONObject(i)
                    list.add(NetworkData(
                        id = obj.optLong("id", 0),
                        name = obj.optString("name", ""),
                        symbol = obj.optString("symbol", ""),
                        rpcUrl = obj.optString("rpcUrl", ""),
                        explorerUrl = obj.optString("explorerUrl", ""),
                        isEnabled = obj.optBoolean("isEnabled", true)
                    ))
                }
                _networks.value = list
            }
        } catch (e: Exception) {
            // Use default networks
            _networks.value = ApiConfig.NETWORKS.map { (id, name) ->
                NetworkData(id = id, name = name, symbol = "", rpcUrl = "", explorerUrl = "", isEnabled = true)
            }
        }
    }
    
    private suspend fun loadTokens() = withContext(Dispatchers.IO) {
        try {
            val response = apiGet("/tokens")
            if (response != null) {
                val jsonArray = JSONArray(response)
                val list = mutableListOf<TokenData>()
                for (i in 0 until jsonArray.length()) {
                    val obj = jsonArray.getJSONObject(i)
                    list.add(TokenData(
                        id = obj.optString("id", ""),
                        name = obj.optString("name", ""),
                        symbol = obj.optString("symbol", ""),
                        chainId = obj.optLong("chainId", 0),
                        address = obj.optString("address", ""),
                        decimals = obj.optInt("decimals", 18)
                    ))
                }
                _tokens.value = list
            }
        } catch (e: Exception) {
            _error.value = "Failed to load tokens: ${e.message}"
        }
    }
    
    private suspend fun loadVolumeStats() = withContext(Dispatchers.IO) {
        try {
            val response = apiGet("/analytics/volume")
            if (response != null) {
                val json = JSONObject(response)
                _volumeStats.value = VolumeStatsData(
                    totalVolume = json.optString("totalVolume", "0"),
                    dailyVolume = json.optString("dailyVolume", "0"),
                    monthlyVolume = json.optString("monthlyVolume", "0"),
                    txCount = json.optInt("txCount", 0)
                )
            }
        } catch (e: Exception) {
            _error.value = "Failed to load volume stats: ${e.message}"
        }
    }
    
    private suspend fun loadWhitelist() = withContext(Dispatchers.IO) {
        try {
            val response = apiGet("/whitelist")
            if (response != null) {
                val jsonArray = JSONArray(response)
                val list = mutableListOf<WhitelistEntry>()
                for (i in 0 until jsonArray.length()) {
                    val obj = jsonArray.getJSONObject(i)
                    list.add(WhitelistEntry(
                        address = obj.optString("address", ""),
                        label = obj.optString("label", ""),
                        isVerified = obj.optBoolean("isVerified", false)
                    ))
                }
                _whitelist.value = list
            }
        } catch (e: Exception) {
            _error.value = "Failed to load whitelist: ${e.message}"
        }
    }
    
    // ============================================================================
    // CRUD Operations
    // ============================================================================
    
    fun createSubWallet(name: String, chain: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val response = apiPost("/subwallets", """{"name":"$name","chain":"$chain"}""")
                if (response != null) {
                    loadSubWallets()
                }
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun deleteSubWallet(id: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                apiDelete("/subwallets/$id")
                loadSubWallets()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun approveTransaction(id: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                apiPost("/transactions/$id/approve", "{}")
                loadTransactions()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun rejectTransaction(id: String, reason: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                apiPost("/transactions/$id/reject", """{"reason":"$reason"}""")
                loadTransactions()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun createAutoSignRule(name: String, maxAmount: String, chain: String, enabled: Boolean) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val body = """{"name":"$name","maxAmount":"$maxAmount","chain":"$chain","enabled":$enabled}"""
                apiPost("/auto-sign/rules", body)
                loadAutoSignRules()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun toggleAutoSignRule(id: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                apiPost("/auto-sign/rules/$id/toggle", "{}")
                loadAutoSignRules()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun addToWhitelist(address: String, label: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                apiPost("/whitelist", """{"address":"$address","label":"$label"}""")
                loadWhitelist()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun removeFromWhitelist(address: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                apiDelete("/whitelist/$address")
                loadWhitelist()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    // ============================================================================
    // HTTP Helper Methods
    // ============================================================================
    
    private fun apiGet(endpoint: String): String? {
        return try {
            val url = URL("${ApiConfig.BASE_URL}${ApiConfig.API_VERSION}$endpoint")
            val conn = url.openConnection() as HttpURLConnection
            conn.requestMethod = "GET"
            conn.setRequestProperty("Content-Type", "application/json")
            authToken?.let { conn.setRequestProperty("Authorization", "Bearer $it") }
            conn.connectTimeout = 10000
            conn.readTimeout = 10000
            
            if (conn.responseCode == 200) {
                conn.inputStream.bufferedReader().readText()
            } else null
        } catch (e: Exception) {
            null
        }
    }
    
    private fun apiPost(endpoint: String, body: String): String? {
        return try {
            val url = URL("${ApiConfig.BASE_URL}${ApiConfig.API_VERSION}$endpoint")
            val conn = url.openConnection() as HttpURLConnection
            conn.requestMethod = "POST"
            conn.setRequestProperty("Content-Type", "application/json")
            authToken?.let { conn.setRequestProperty("Authorization", "Bearer $it") }
            conn.doOutput = true
            conn.connectTimeout = 10000
            conn.readTimeout = 10000
            
            conn.outputStream.write(body.toByteArray())
            
            if (conn.responseCode in 200..299) {
                conn.inputStream.bufferedReader().readText()
            } else null
        } catch (e: Exception) {
            null
        }
    }
    
    private fun apiDelete(endpoint: String): Boolean {
        return try {
            val url = URL("${ApiConfig.BASE_URL}${ApiConfig.API_VERSION}$endpoint")
            val conn = url.openConnection() as HttpURLConnection
            conn.requestMethod = "DELETE"
            conn.setRequestProperty("Content-Type", "application/json")
            authToken?.let { conn.setRequestProperty("Authorization", "Bearer $it") }
            conn.connectTimeout = 10000
            conn.responseCode in 200..299
        } catch (e: Exception) {
            false
        }
    }
}

// ============================================================================
// Data Classes
// ============================================================================

data class MasterWalletData(
    val address: String,
    val totalVolume: String,
    val subWalletCount: Int,
    val userCount: Int,
    val pendingTx: Int
)

data class SubWalletData(
    val name: String,
    val address: String,
    val balance: String,
    val status: String
)

data class TransactionData(
    val id: String,
    val hash: String,
    val from: String,
    val to: String,
    val amount: String,
    val token: String,
    val chain: String,
    val status: String,
    val timestamp: Long
)

data class AutoSignRuleData(
    val id: String,
    val name: String,
    val maxAmount: String,
    val chain: String,
    val enabled: Boolean
)

data class UserData(
    val id: String,
    val email: String,
    val name: String,
    val role: String,
    val walletAddress: String,
    val permissions: List<String>
)

data class NetworkData(
    val id: Long,
    val name: String,
    val symbol: String,
    val rpcUrl: String,
    val explorerUrl: String,
    val isEnabled: Boolean
)

data class TokenData(
    val id: String,
    val name: String,
    val symbol: String,
    val chainId: Long,
    val address: String,
    val decimals: Int
)

data class VolumeStatsData(
    val totalVolume: String,
    val dailyVolume: String,
    val monthlyVolume: String,
    val txCount: Int
)

data class WhitelistEntry(
    val address: String,
    val label: String,
    val isVerified: Boolean
)
