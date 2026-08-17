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
    // Canonical MasterWallet backend (see CANONICAL_API_CONTRACT.md)
    const val BASE_URL = "http://localhost:8450"
    const val API_VERSION = "/api/v1"
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
    private var masterWalletId: String? = null
    
    init {
        loadData()
    }
    
    fun setAuthToken(token: String) {
        authToken = token
        loadData()
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
            val response = apiGet("/master-wallet") ?: return@withContext
            val json = JSONObject(response)
            val wallets = json.optJSONArray("wallets")
            if (wallets != null && wallets.length() > 0) {
                val first = wallets.getJSONObject(0)
                masterWalletId = first.optString("id", "")
                _masterWallet.value = MasterWalletData(
                    id = first.optString("id", ""),
                    address = first.optString("address", ""),
                    name = first.optString("name", ""),
                    createdAt = first.optString("created_at", "")
                )
            }
        } catch (e: Exception) {
            _error.value = "Failed to load master wallet: ${e.message}"
        }
    }
    
    private fun requireWalletId(): String? =
        masterWalletId?.takeIf { it.isNotBlank() }

    private suspend fun loadSubWallets() = withContext(Dispatchers.IO) {
        val id = requireWalletId() ?: return@withContext
        try {
            val response = apiGet("/master-wallet/$id/sub-wallets") ?: return@withContext
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
        } catch (e: Exception) {
            _error.value = "Failed to load subwallets: ${e.message}"
        }
    }
    
    private suspend fun loadTransactions() = withContext(Dispatchers.IO) {
        val id = requireWalletId() ?: return@withContext
        try {
            val response = apiGet("/master-wallet/$id/transactions") ?: return@withContext
            val json = JSONObject(response)
            val jsonArray = json.optJSONArray("transactions") ?: JSONArray(response)
            val list = mutableListOf<TransactionData>()
            for (i in 0 until jsonArray.length()) {
                val obj = jsonArray.getJSONObject(i)
                list.add(TransactionData(
                    id = obj.optString("id", ""),
                    hash = obj.optString("hash", obj.optString("transaction_hash", "")),
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
        } catch (e: Exception) {
            _error.value = "Failed to load transactions: ${e.message}"
        }
    }
    
    private suspend fun loadAutoSignRules() = withContext(Dispatchers.IO) {
        val id = requireWalletId() ?: return@withContext
        try {
            val response = apiGet("/master-wallet/$id/auto-sign") ?: return@withContext
            val jsonArray = JSONArray(response)
            val list = mutableListOf<AutoSignRuleData>()
            for (i in 0 until jsonArray.length()) {
                val obj = jsonArray.getJSONObject(i)
                list.add(AutoSignRuleData(
                    id = obj.optString("id", ""),
                    name = obj.optString("name", ""),
                    maxAmount = obj.optString("maxAmount", obj.optString("threshold", "0")),
                    chain = obj.optString("chain", ""),
                    enabled = obj.optBoolean("enabled", false)
                ))
            }
            _autoSignRules.value = list
        } catch (e: Exception) {
            _error.value = "Failed to load auto-sign rules: ${e.message}"
        }
    }
    
    private suspend fun loadUsers() = withContext(Dispatchers.IO) {
        val id = requireWalletId() ?: return@withContext
        try {
            val response = apiGet("/master-wallet/$id/users") ?: return@withContext
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
        } catch (e: Exception) {
            _error.value = "Failed to load users: ${e.message}"
        }
    }
    
    private suspend fun loadNetworks() = withContext(Dispatchers.IO) {
        try {
            // Public endpoint: no wallet id required
            val response = apiGet("/chains") ?: return@withContext
            val json = JSONObject(response)
            val jsonArray = json.optJSONArray("chains") ?: JSONArray(response)
            val list = mutableListOf<NetworkData>()
            for (i in 0 until jsonArray.length()) {
                val obj = jsonArray.getJSONObject(i)
                list.add(NetworkData(
                    id = obj.optLong("id", obj.optLong("chain_id", 0)),
                    name = obj.optString("name", ""),
                    symbol = obj.optString("symbol", ""),
                    rpcUrl = obj.optString("rpcUrl", obj.optString("rpc_url", "")),
                    explorerUrl = obj.optString("explorerUrl", obj.optString("explorer_url", "")),
                    isEnabled = obj.optBoolean("isEnabled", obj.optBoolean("is_enabled", true))
                ))
            }
            _networks.value = list
        } catch (e: Exception) {
            _error.value = "Failed to load networks: ${e.message}"
        }
    }
    
    private suspend fun loadTokens() = withContext(Dispatchers.IO) {
        // No canonical tokens list endpoint; derived from wallet balances instead of fake data.
        _tokens.value = emptyList()
    }
    
    private suspend fun loadVolumeStats() = withContext(Dispatchers.IO) {
        val id = requireWalletId() ?: return@withContext
        try {
            val response = apiGet("/master-wallet/$id/analytics/volume") ?: return@withContext
            val json = JSONObject(response)
            _volumeStats.value = VolumeStatsData(
                totalVolume = json.optString("totalVolume", json.optString("total_volume", "0")),
                dailyVolume = json.optString("dailyVolume", json.optString("daily_volume", "0")),
                monthlyVolume = json.optString("monthlyVolume", json.optString("monthly_volume", "0")),
                txCount = json.optInt("txCount", json.optInt("tx_count", 0))
            )
        } catch (e: Exception) {
            _error.value = "Failed to load volume stats: ${e.message}"
        }
    }
    
    private suspend fun loadWhitelist() = withContext(Dispatchers.IO) {
        val id = requireWalletId() ?: return@withContext
        try {
            val response = apiGet("/master-wallet/$id/webhooks") ?: return@withContext
            val jsonArray = JSONArray(response)
            val list = mutableListOf<WhitelistEntry>()
            for (i in 0 until jsonArray.length()) {
                val obj = jsonArray.getJSONObject(i)
                list.add(WhitelistEntry(
                    address = obj.optString("address", obj.optString("url", "")),
                    label = obj.optString("label", obj.optString("name", "")),
                    isVerified = obj.optBoolean("isVerified", obj.optBoolean("active", false))
                ))
            }
            _whitelist.value = list
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
                val id = requireWalletId()
                if (id == null) {
                    _error.value = "No master wallet selected"
                    return@launch
                }
                val body = JSONObject()
                    .put("name", name)
                    .put("password", "")
                    .put("chain_id", chain)
                    .toString()
                apiPost("/master-wallet/$id/sub-wallets", body)
                loadSubWallets()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun deleteSubWallet(sid: String) {
        // The canonical MasterWallet backend (port 8450) exposes no DELETE
        // /master-wallet/:id/sub-wallets/:sid route. Sub-wallets are derived
        // HD children of the master wallet and cannot be deleted on-chain;
        // deletion is a governance record that requires a SuperAdmin-gated
        // route that is not part of the canonical contract. Fail-closed.
        viewModelScope.launch {
            _error.value =
                "Sub-wallet deletion is not supported by the canonical MasterWallet backend"
        }
    }
    
    fun approveTransaction(tid: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val id = requireWalletId()
                if (id == null) {
                    _error.value = "No master wallet selected"
                    return@launch
                }
                apiPost("/master-wallet/$id/transactions/$tid/approve", "{}")
                loadTransactions()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun rejectTransaction(tid: String, reason: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val id = requireWalletId()
                if (id == null) {
                    _error.value = "No master wallet selected"
                    return@launch
                }
                val body = JSONObject().put("reason", reason).toString()
                apiPost("/master-wallet/$id/transactions/$tid/reject", body)
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
                val id = requireWalletId()
                if (id == null) {
                    _error.value = "No master wallet selected"
                    return@launch
                }
                val body = JSONObject()
                    .put("name", name)
                    .put("threshold", maxAmount)
                    .put("chain", chain)
                    .put("enabled", enabled)
                    .toString()
                apiPost("/master-wallet/$id/auto-sign", body)
                loadAutoSignRules()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun deleteAutoSignRule(rid: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val id = requireWalletId()
                if (id == null) {
                    _error.value = "No master wallet selected"
                    return@launch
                }
                apiDelete("/master-wallet/$id/auto-sign/$rid")
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
                val id = requireWalletId()
                if (id == null) {
                    _error.value = "No master wallet selected"
                    return@launch
                }
                val body = JSONObject()
                    .put("address", address)
                    .put("label", label)
                    .toString()
                apiPost("/master-wallet/$id/webhooks", body)
                loadWhitelist()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun removeFromWhitelist(wid: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val id = requireWalletId()
                if (id == null) {
                    _error.value = "No master wallet selected"
                    return@launch
                }
                apiDelete("/master-wallet/$id/webhooks/$wid")
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
    val id: String,
    val address: String,
    val name: String,
    val createdAt: String
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
