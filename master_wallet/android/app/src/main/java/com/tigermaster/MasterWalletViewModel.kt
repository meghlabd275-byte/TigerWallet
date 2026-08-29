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
                loadKillSwitchStatus()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    /** Read-only SuperAdmin kill-switch state (GET /api/v1/kill-switch/status). */
    fun loadKillSwitchStatus() {
        viewModelScope.launch(Dispatchers.IO) {
            try {
                apiGet("/kill-switch/status")?.let { _killSwitch.value = JSONObject(it) }
            } catch (_: Exception) { /* status unknown — leave previous state */ }
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

    private fun apiPut(endpoint: String, body: String): String? {
        return try {
            val url = URL("${ApiConfig.BASE_URL}${ApiConfig.API_VERSION}$endpoint")
            val conn = url.openConnection() as HttpURLConnection
            conn.requestMethod = "PUT"
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

    // ============================================================================
    // Feature-governance surfaces (treasury, multisig, fees, policies, users,
    // chains, tokens, flags, webhooks, notifications, audit, analytics).
    // Every loader reads the real backend response; nothing is fabricated.
    // ============================================================================

    private val _selectedFeature = MutableStateFlow<String?>(null)
    val selectedFeature: StateFlow<String?> = _selectedFeature.asStateFlow()
    fun openFeature(f: String) { _selectedFeature.value = f }
    fun closeFeature() { _selectedFeature.value = null }

    private val _fees = MutableStateFlow<List<JSONObject>>(emptyList())
    val fees: StateFlow<List<JSONObject>> = _fees.asStateFlow()
    private val _policies = MutableStateFlow<List<JSONObject>>(emptyList())
    val policies: StateFlow<List<JSONObject>> = _policies.asStateFlow()
    private val _auditLogs = MutableStateFlow<List<JSONObject>>(emptyList())
    val auditLogs: StateFlow<List<JSONObject>> = _auditLogs.asStateFlow()
    private val _notifications = MutableStateFlow<List<JSONObject>>(emptyList())
    val notifications: StateFlow<List<JSONObject>> = _notifications.asStateFlow()
    private val _webhooks = MutableStateFlow<List<JSONObject>>(emptyList())
    val webhooks: StateFlow<List<JSONObject>> = _webhooks.asStateFlow()
    private val _featureFlags = MutableStateFlow<List<JSONObject>>(emptyList())
    val featureFlags: StateFlow<List<JSONObject>> = _featureFlags.asStateFlow()
    private val _multisigWallets = MutableStateFlow<List<JSONObject>>(emptyList())
    val multisigWallets: StateFlow<List<JSONObject>> = _multisigWallets.asStateFlow()
    private val _multisigTxs = MutableStateFlow<List<JSONObject>>(emptyList())
    val multisigTxs: StateFlow<List<JSONObject>> = _multisigTxs.asStateFlow()
    private val _treasury = MutableStateFlow<JSONObject?>(null)
    val treasury: StateFlow<JSONObject?> = _treasury.asStateFlow()
    private val _treasuryTxs = MutableStateFlow<List<JSONObject>>(emptyList())
    val treasuryTxs: StateFlow<List<JSONObject>> = _treasuryTxs.asStateFlow()
    private val _autoSignLogs = MutableStateFlow<List<JSONObject>>(emptyList())
    val autoSignLogs: StateFlow<List<JSONObject>> = _autoSignLogs.asStateFlow()
    private val _autoSignPolicy = MutableStateFlow<JSONObject?>(null)
    val autoSignPolicy: StateFlow<JSONObject?> = _autoSignPolicy.asStateFlow()
    private val _evmChains = MutableStateFlow<List<JSONObject>>(emptyList())
    val evmChains: StateFlow<List<JSONObject>> = _evmChains.asStateFlow()
    private val _nonEvmChains = MutableStateFlow<List<JSONObject>>(emptyList())
    val nonEvmChains: StateFlow<List<JSONObject>> = _nonEvmChains.asStateFlow()
    private val _userTokens = MutableStateFlow<List<JSONObject>>(emptyList())
    val userTokens: StateFlow<List<JSONObject>> = _userTokens.asStateFlow()
    private val _analytics = MutableStateFlow<Map<String, String>>(emptyMap())
    val analytics: StateFlow<Map<String, String>> = _analytics.asStateFlow()
    private val _passkeys = MutableStateFlow<List<JSONObject>>(emptyList())
    val passkeys: StateFlow<List<JSONObject>> = _passkeys.asStateFlow()
    private val _subWalletList = MutableStateFlow<List<JSONObject>>(emptyList())
    val subWalletList: StateFlow<List<JSONObject>> = _subWalletList.asStateFlow()
    private val _liveEvent = MutableStateFlow<String?>(null)
    val liveEvent: StateFlow<String?> = _liveEvent.asStateFlow()
    private val _killSwitch = MutableStateFlow<JSONObject?>(null)
    val killSwitch: StateFlow<JSONObject?> = _killSwitch.asStateFlow()

    /** Parse either a raw JSON array or a {"<key>":[...]} envelope. */
    private fun parseList(raw: String?, key: String): List<JSONObject> {
        if (raw.isNullOrBlank()) return emptyList()
        return try {
            val trimmed = raw.trim()
            val arr = if (trimmed.startsWith("[")) JSONArray(trimmed)
            else JSONObject(trimmed).optJSONArray(key) ?: run {
                val obj = JSONObject(trimmed)
                var found: JSONArray? = null
                for (k in obj.keys()) {
                    val v = obj.optJSONArray(k)
                    if (v != null) { found = v; break }
                }
                found ?: JSONArray()
            }
            (0 until arr.length()).mapNotNull { arr.optJSONObject(it) }
        } catch (e: Exception) {
            emptyList()
        }
    }

    /** Load one feature surface on demand (called when its screen opens). */
    fun loadFeature(feature: String) {
        viewModelScope.launch(Dispatchers.IO) {
            val id = requireWalletId() ?: run {
                _error.value = "No master wallet selected"
                return@launch
            }
            try {
                when (feature) {
                    "subwallets" -> _subWalletList.value = parseList(apiGet("/master-wallet/$id/sub-wallets"), "sub_wallets")
                    "treasury" -> {
                        apiGet("/master-wallet/$id/treasury")?.let { _treasury.value = JSONObject(it) }
                        _treasuryTxs.value = parseList(apiGet("/master-wallet/$id/treasury/transactions"), "transactions")
                    }
                    "multisig" -> {
                        _multisigWallets.value = parseList(apiGet("/master-wallet/$id/multisig/wallets"), "multisig_wallets")
                    }
                    "autosign" -> {
                        loadAutoSignRules()
                        _autoSignLogs.value = parseList(apiGet("/master-wallet/$id/auto-sign-logs"), "logs")
                        apiGet("/master-wallet/$id/auto-sign-policy")?.let {
                            val obj = JSONObject(it)
                            _autoSignPolicy.value = obj.optJSONObject("policy") ?: obj
                        }
                    }
                    "fees" -> _fees.value = parseList(apiGet("/master-wallet/$id/fees"), "fees")
                    "policies" -> _policies.value = parseList(apiGet("/master-wallet/$id/policies"), "policies")
                    "users" -> loadUsers()
                    "audit" -> _auditLogs.value = parseList(apiGet("/master-wallet/$id/audit"), "logs")
                    "notifications" -> _notifications.value = parseList(apiGet("/master-wallet/$id/notifications"), "notifications")
                    "webhooks" -> {
                        _webhooks.value = parseList(apiGet("/master-wallet/$id/webhooks"), "webhooks")
                        _notifications.value = parseList(apiGet("/master-wallet/$id/notifications"), "notifications")
                    }
                    "flags" -> _featureFlags.value = parseList(apiGet("/master-wallet/$id/feature-flags"), "feature_flags")
                    "chains" -> {
                        _evmChains.value = parseList(apiGet("/master-wallet/$id/user-chains/evm"), "chains")
                        _nonEvmChains.value = parseList(apiGet("/master-wallet/$id/user-chains/nonevm"), "chains")
                    }
                    "tokens" -> _userTokens.value = parseList(apiGet("/master-wallet/$id/user-tokens"), "tokens")
                    "passkeys" -> _passkeys.value = parseList(apiGet("/master-wallet/$id/passkey/credentials"), "passkeys")
                    "killswitch" -> loadKillSwitchStatus()
                    "analytics" -> {
                        val out = mutableMapOf<String, String>()
                        apiGet("/master-wallet/$id/analytics/volume")?.let { r ->
                            val o = JSONObject(r); for (k in o.keys()) out["volume.$k"] = o.optString(k)
                        }
                        apiGet("/master-wallet/$id/analytics/transactions")?.let { r ->
                            val o = JSONObject(r)
                            if (o.length() <= 8) for (k in o.keys()) out["transactions.$k"] = o.optString(k)
                            else out["transactions.count"] = parseList(r, "transactions").size.toString()
                        }
                        apiGet("/master-wallet/$id/analytics/wallets")?.let { r ->
                            val o = JSONObject(r)
                            if (o.length() <= 8) for (k in o.keys()) out["wallets.$k"] = o.optString(k)
                            else out["wallets.count"] = parseList(r, "wallets").size.toString()
                        }
                        _analytics.value = out
                    }
                }
            } catch (e: Exception) {
                _error.value = e.message
            }
        }
    }

    /** POST a JSON body to a wallet-scoped route, then reload the surface. */
    private fun featureAction(feature: String, build: (id: String) -> Boolean) {
        viewModelScope.launch(Dispatchers.IO) {
            _isLoading.value = true
            try {
                val id = requireWalletId() ?: run {
                    _error.value = "No master wallet selected"
                    return@launch
                }
                if (!build(id)) {
                    _error.value = "Request failed"
                    return@launch
                }
                loadFeature(feature)
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun treasuryTransfer(to: String, amount: String, password: String) =
        featureAction("treasury") { id ->
            apiPost("/master-wallet/$id/treasury/transfer",
                JSONObject().put("to", to).put("amount", amount).put("password", password).toString()) != null
        }

    fun treasurySweep(subWalletId: String, password: String) =
        featureAction("treasury") { id ->
            apiPost("/master-wallet/$id/treasury/sweep",
                JSONObject().put("sub_wallet_id", subWalletId).put("password", password).toString()) != null
        }

    fun createMultisig(name: String, ownersCsv: String, threshold: Int) =
        featureAction("multisig") { id ->
            val owners = JSONArray()
            ownersCsv.split(",").map { it.trim() }.filter { it.isNotEmpty() }.forEach { owners.put(it) }
            apiPost("/master-wallet/$id/multisig/wallets",
                JSONObject().put("name", name).put("owners", owners).put("threshold", threshold).toString()) != null
        }

    fun loadMultisigTxs(multisigId: String) {
        viewModelScope.launch(Dispatchers.IO) {
            val id = requireWalletId() ?: return@launch
            _multisigTxs.value = parseList(apiGet("/master-wallet/$id/multisig/wallets/$multisigId/transactions"), "transactions")
        }
    }

    fun signMultisig(txId: String) =
        featureAction("multisig") { id ->
            apiPost("/master-wallet/$id/multisig/transactions/$txId/sign", "{}") != null
        }

    fun executeMultisig(txId: String) =
        featureAction("multisig") { id ->
            apiPost("/master-wallet/$id/multisig/transactions/$txId/execute", "{}") != null
        }

    fun createFee(feeType: String, percentage: Double, fixed: String) =
        featureAction("fees") { id ->
            apiPost("/master-wallet/$id/fees",
                JSONObject().put("fee_type", feeType).put("fee_percentage", percentage).put("fee_fixed", fixed).toString()) != null
        }

    fun toggleFee(fid: String, active: Boolean) =
        featureAction("fees") { id ->
            apiPut("/master-wallet/$id/fees/$fid", JSONObject().put("is_active", active).toString()) != null
        }

    fun deleteFee(fid: String) =
        featureAction("fees") { id -> apiDelete("/master-wallet/$id/fees/$fid") }

    fun createPolicy(name: String, policyType: String) =
        featureAction("policies") { id ->
            apiPost("/master-wallet/$id/policies",
                JSONObject().put("name", name).put("policy_type", policyType).toString()) != null
        }

    fun deletePolicy(pid: String) =
        featureAction("policies") { id -> apiDelete("/master-wallet/$id/policies/$pid") }

    fun createUser(email: String, password: String, name: String, role: String) =
        featureAction("users") { id ->
            apiPost("/master-wallet/$id/users",
                JSONObject().put("email", email).put("password", password).put("name", name).put("role", role).toString()) != null
        }

    fun deleteUser(uid: String) =
        featureAction("users") { id -> apiDelete("/master-wallet/$id/users/$uid") }

    fun setAutoSignPolicyEnabled(enabled: Boolean) =
        featureAction("autosign") { id ->
            apiPut("/master-wallet/$id/auto-sign-policy", JSONObject().put("enabled", enabled).toString()) != null
        }

    fun createWebhook(name: String, url: String, eventsCsv: String) =
        featureAction("webhooks") { id ->
            val events = JSONArray()
            eventsCsv.split(",").map { it.trim() }.filter { it.isNotEmpty() }.forEach { events.put(it) }
            apiPost("/master-wallet/$id/webhooks",
                JSONObject().put("name", name).put("url", url).put("events", events).toString()) != null
        }

    fun deleteWebhook(wid: String) =
        featureAction("webhooks") { id -> apiDelete("/master-wallet/$id/webhooks/$wid") }

    fun createNotification(type: String, title: String, message: String) =
        featureAction("webhooks") { id ->
            apiPost("/master-wallet/$id/notifications",
                JSONObject().put("notification_type", type).put("title", title).put("message", message).toString()) != null
        }

    fun addFeatureFlag(flagKey: String) =
        featureAction("flags") { id ->
            apiPost("/master-wallet/$id/feature-flags",
                JSONObject().put("flag_key", flagKey).put("is_enabled", true).toString()) != null
        }

    fun toggleFeatureFlag(flagId: Long, enabled: Boolean) =
        featureAction("flags") { id ->
            apiPut("/master-wallet/$id/feature-flags/$flagId", JSONObject().put("is_enabled", enabled).toString()) != null
        }

    fun removeFeatureFlag(flagId: Long) =
        featureAction("flags") { id -> apiDelete("/master-wallet/$id/feature-flags/$flagId") }

    fun addEvmChain(chainId: Long, name: String, rpcUrl: String, symbol: String) =
        featureAction("chains") { id ->
            apiPost("/master-wallet/$id/user-chains/evm",
                JSONObject().put("chain_id", chainId).put("name", name).put("rpc_url", rpcUrl).put("symbol", symbol).toString()) != null
        }

    fun removeEvmChain(chainId: Long) =
        featureAction("chains") { id -> apiDelete("/master-wallet/$id/user-chains/evm/$chainId") }

    fun addNonEvmChain(chainId: Long, name: String, chainType: String, rpcUrl: String, derivationPath: String) =
        featureAction("chains") { id ->
            apiPost("/master-wallet/$id/user-chains/nonevm",
                JSONObject().put("chain_id", chainId).put("name", name).put("chain_type", chainType)
                    .put("rpc_url", rpcUrl).put("derivation_path", derivationPath).toString()) != null
        }

    fun removeNonEvmChain(chainId: Long) =
        featureAction("chains") { id -> apiDelete("/master-wallet/$id/user-chains/nonevm/$chainId") }

    fun addUserToken(chainId: Long, symbol: String, name: String, contractAddress: String, decimals: Int) =
        featureAction("tokens") { id ->
            apiPost("/master-wallet/$id/user-tokens",
                JSONObject().put("chain_id", chainId).put("symbol", symbol).put("name", name)
                    .put("contract_address", contractAddress).put("decimals", decimals).toString()) != null
        }

    fun removeUserToken(tokenId: Long) =
        featureAction("tokens") { id -> apiDelete("/master-wallet/$id/user-tokens/$tokenId") }

    fun deletePasskey(credId: String) =
        featureAction("passkeys") { id ->
            apiDelete("/master-wallet/$id/passkey/credentials/" + java.net.URLEncoder.encode(credId, "UTF-8"))
        }

    /** Register a platform passkey via CredentialManager (API 34+) with an
     *  AndroidKeyStore P-256 fallback, then POST it to the backend. */
    fun registerPasskey(context: android.content.Context, label: String, onDone: (String) -> Unit) {
        viewModelScope.launch(Dispatchers.IO) {
            try {
                val id = requireWalletId() ?: run { onDone("No master wallet selected"); return@launch }
                val mws = com.tigermaster.services.MasterWalletService()
                mws.setAuthToken(authToken)
                val svc = com.tigermaster.services.PasskeyService(context, mws)
                svc.initialize()
                val cred = svc.registerPasskey(
                    masterId = id,
                    relyingPartyId = "tigerwallet.app",
                    relyingPartyName = "TigerWallet Master",
                    userId = id,
                    userName = "master-owner",
                    label = label
                )
                onDone(if (cred != null) "Passkey registered." else "Passkey registration failed")
                loadFeature("passkeys")
            } catch (e: Exception) {
                onDone(e.message ?: "error")
            }
        }
    }

    /** Live backend /ws feed: real balance/transaction events update the
     *  dashboard instantly. Disconnect-safe; polling refresh remains. */
    private var wsService: com.tigermaster.services.WebSocketService? = null
    private var wsJob: kotlinx.coroutines.Job? = null

    fun startLiveFeed() {
        if (wsJob != null) return
        val id = _masterWallet.value?.id ?: return
        val svc = com.tigermaster.services.WebSocketService()
        wsService = svc
        svc.connect(id, authToken)
        wsJob = viewModelScope.launch {
            svc.messages.collect { msg ->
                _liveEvent.value = msg.type + if (msg.data.isNotBlank()) " · " + msg.data.take(80) else ""
                if (msg.type == "balance" || msg.type == "transaction" || msg.type == "tx" || msg.channel == "transactions") {
                    loadData()
                }
            }
        }
    }

    fun stopLiveFeed() {
        wsJob?.cancel(); wsJob = null
        wsService?.disconnect(); wsService = null
    }

    fun transferFromSubWallet(sid: String, to: String, amount: String, password: String) =
        featureAction("subwallets") { id ->
            apiPost("/master-wallet/$id/sub-wallets/$sid/transfer",
                JSONObject().put("to", to).put("amount", amount).put("password", password).toString()) != null
        }

    fun sendSigned(to: String, amount: String, token: String, password: String, onDone: (String) -> Unit) {
        viewModelScope.launch(Dispatchers.IO) {
            try {
                val id = requireWalletId() ?: run { onDone("No master wallet selected"); return@launch }
                val body = JSONObject().put("to", to).put("amount", amount).put("password", password)
                if (token.isNotBlank()) body.put("token", token)
                val resp = apiPost("/master-wallet/$id/sign", body.toString())
                if (resp != null) {
                    val obj = JSONObject(resp)
                    val hash = obj.optString("transaction_hash", obj.optString("tx_hash", obj.optString("hash", "")))
                    onDone("Transaction submitted to the blockchain network" + (if (hash.isNotBlank()) ": $hash" else ""))
                    loadData()
                } else onDone("Send failed")
            } catch (e: Exception) { onDone(e.message ?: "error") }
        }
    }

    fun checkAutoSignPolicyNow(txType: String, value: String, onDone: (String) -> Unit) {
        viewModelScope.launch(Dispatchers.IO) {
            try {
                val id = requireWalletId() ?: run { onDone("No master wallet selected"); return@launch }
                val resp = apiPost("/master-wallet/$id/check-auto-sign-policy",
                    JSONObject().put("tx_type", txType).put("value", value).toString())
                if (resp != null) {
                    val obj = JSONObject(resp)
                    onDone((if (obj.optBoolean("allowed")) "ALLOWED" else "DENIED") + " — " + obj.optString("reason", ""))
                } else onDone("Policy check failed")
            } catch (e: Exception) { onDone(e.message ?: "error") }
        }
    }

    fun autoSignTransactionNow(mnemonic: String, chainId: Long, chainType: String, txType: String, to: String, value: String, tokenAddress: String, onDone: (String) -> Unit) {
        viewModelScope.launch(Dispatchers.IO) {
            try {
                val id = requireWalletId() ?: run { onDone("No master wallet selected"); return@launch }
                val body = JSONObject()
                    .put("mnemonic", mnemonic).put("chain_id", chainId).put("chain_type", chainType)
                    .put("tx_type", txType).put("to_address", to).put("value", value)
                if (tokenAddress.isNotBlank()) body.put("token_address", tokenAddress)
                val resp = apiPost("/master-wallet/$id/auto-sign-transaction", body.toString())
                if (resp != null) {
                    val obj = JSONObject(resp)
                    val hash = obj.optString("transaction_hash", obj.optString("tx_hash", obj.optString("hash", "")))
                    onDone("Transaction submitted to the blockchain network" + (if (hash.isNotBlank()) ": $hash" else ""))
                } else onDone("Auto-sign failed")
            } catch (e: Exception) { onDone(e.message ?: "error") }
        }
    }

    fun userWalletAutoSign(mnemonic: String, chainId: Long, chainType: String, txType: String, onDone: (String) -> Unit) {
        viewModelScope.launch(Dispatchers.IO) {
            try {
                val id = requireWalletId() ?: run { onDone("No master wallet selected"); return@launch }
                val resp = apiPost("/master-wallet/$id/user-wallet-auto-sign",
                    JSONObject().put("mnemonic", mnemonic).put("chain_id", chainId)
                        .put("chain_type", chainType).put("tx_type", txType).toString())
                onDone(if (resp != null) "UserWallet auto-sign configuration saved." else "Auto-sign setup failed")
            } catch (e: Exception) { onDone(e.message ?: "error") }
        }
    }

    fun revenuePayout(to: String, amount: String, password: String, withdrawalId: String, onDone: (String) -> Unit) {
        viewModelScope.launch(Dispatchers.IO) {
            try {
                val id = requireWalletId() ?: run { onDone("No master wallet selected"); return@launch }
                val resp = apiPost("/master-wallet/$id/revenue-payout",
                    JSONObject().put("to", to).put("amount", amount)
                        .put("password", password).put("withdrawal_id", withdrawalId).toString())
                if (resp != null) {
                    val obj = JSONObject(resp)
                    val hash = obj.optString("transaction_hash", obj.optString("tx_hash", obj.optString("hash", "")))
                    onDone("Payout submitted to the blockchain network" + (if (hash.isNotBlank()) ": $hash" else ""))
                } else onDone("Payout failed (SuperAdmin co-sign required)")
            } catch (e: Exception) { onDone(e.message ?: "error") }
        }
    }

    fun updateEvmChain(chainId: Long, name: String, rpcUrl: String, symbol: String) =
        featureAction("chains") { id ->
            apiPut("/master-wallet/$id/user-chains/evm/$chainId",
                JSONObject().put("name", name).put("rpc_url", rpcUrl).put("symbol", symbol).toString()) != null
        }

    fun updateNonEvmChain(chainId: Long, name: String, chainType: String, rpcUrl: String, derivationPath: String) =
        featureAction("chains") { id ->
            apiPut("/master-wallet/$id/user-chains/nonevm/$chainId",
                JSONObject().put("name", name).put("chain_type", chainType)
                    .put("rpc_url", rpcUrl).put("derivation_path", derivationPath).toString()) != null
        }

    fun updateUserToken(tokenId: Long, symbol: String, name: String, contractAddress: String, decimals: Int) =
        featureAction("tokens") { id ->
            apiPut("/master-wallet/$id/user-tokens/$tokenId",
                JSONObject().put("symbol", symbol).put("name", name)
                    .put("contract_address", contractAddress).put("decimals", decimals).toString()) != null
        }

    fun requestWithdrawal(toAddress: String, amountWei: String, currency: String, chainId: Long, onDone: (String) -> Unit) {
        viewModelScope.launch(Dispatchers.IO) {
            try {
                val id = requireWalletId() ?: run {
                    onDone("No master wallet selected")
                    return@launch
                }
                val resp = apiPost("/master-wallet/$id/withdrawal-request",
                    JSONObject().put("to_address", toAddress).put("amount_wei", amountWei)
                        .put("currency", currency).put("chain_id", chainId).toString())
                onDone(if (resp != null) "Withdrawal request filed (SuperAdmin co-sign required): $resp"
                else "Withdrawal request failed")
            } catch (e: Exception) {
                onDone(e.message ?: "error")
            }
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
