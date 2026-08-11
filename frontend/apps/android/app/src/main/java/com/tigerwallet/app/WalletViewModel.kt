//
//  WalletViewModel.kt
//  TigerWallet - Android Wallet ViewModel
//

package com.tigerwallet.app

import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import java.net.HttpURLConnection
import java.net.URL

class WalletViewModel(application: android.app.Application) : AndroidViewModel(application) {
    private val _wallet = MutableStateFlow<Wallet?>(null)
    val wallet: StateFlow<Wallet?> = _wallet.asStateFlow()
    
    private val _selectedChain = MutableStateFlow(Chain.chains[0])
    val selectedChain: StateFlow<Chain> = _selectedChain.asStateFlow()
    
    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()
    
    private val _isDarkMode = MutableStateFlow(false)
    val isDarkMode: StateFlow<Boolean> = _isDarkMode.asStateFlow()
    
    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error.asStateFlow()
    
    init {
        loadWallet()
    }
    
    fun selectChain(chain: Chain) {
        _selectedChain.value = chain
        loadWallet()
    }
    
    fun toggleDarkMode() {
        _isDarkMode.value = !_isDarkMode.value
    }
    
    private fun loadWallet() {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                // REAL backend integration: fetch wallet data from the Go
                // wallet_api backend. No hardcoded mock wallet, no simulated
                // delay. If no wallet address is stored (user hasn't created /
                // imported one yet), surface an honest "not connected" state
                // rather than fabricating a wallet.
                val address = getStoredAddress()
                if (address.isNullOrEmpty()) {
                    _wallet.value = null
                    _error.value = "No wallet connected. Create or import a wallet to view balances."
                    return@launch
                }
                val chain = _selectedChain.value
                val resp = httpGet(
                    "$BACKEND_URL/api/v1/public/balance?address=$address&chain_id=${chain.id}"
                )
                if (resp == null) {
                    _wallet.value = null
                    _error.value = "Unable to reach wallet backend."
                    return@launch
                }
                val nativeBalance = extractField(resp, "balance") ?: "0"
                val nativeSymbol = chain.symbol
                val tokensResp = httpGet(
                    "$BACKEND_URL/api/v1/public/tokens?address=$address&chain_id=${chain.id}"
                )
                val tokens = parseTokens(tokensResp, nativeSymbol, nativeBalance)
                _error.value = null
                _wallet.value = Wallet(
                    address = address,
                    totalBalance = tokens.sumOf { it.usdValue },
                    nativeBalance = nativeBalance,
                    chain = chain,
                    tokens = tokens
                )
            } catch (e: Exception) {
                _wallet.value = null
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun sendTransaction(to: String, amount: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                // Sending requires an authenticated session (JWT) + wallet id
                // so the backend can unlock the encrypted seed (POST /api/v1/send).
                // Without those we MUST NOT simulate a transfer.
                val token = getStoredAuthToken()
                val walletId = getStoredWalletId()
                if (token.isNullOrEmpty() || walletId.isNullOrEmpty()) {
                    _error.value = "Sign in required to send transactions."
                    return@launch
                }
                val body = """{"wallet_id":"$walletId","to":"$to","amount":"$amount","chain_id":${_selectedChain.value.id}}"""
                val resp = httpPost("$BACKEND_URL/api/v1/send", body, token)
                if (resp == null) {
                    _error.value = "Send failed: backend unreachable."
                    return@launch
                }
                loadWallet()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun swap(fromToken: Token, toToken: Token, amount: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                // Real swaps are executed via the backend swap/DEX service.
                // Without an authenticated session we refuse rather than simulate.
                val token = getStoredAuthToken()
                if (token.isNullOrEmpty()) {
                    _error.value = "Sign in required to swap tokens."
                    return@launch
                }
                val body = """{"from":"${fromToken.symbol}","to":"${toToken.symbol}","amount":"$amount","chain_id":${_selectedChain.value.id}}"""
                val resp = httpPost("$BACKEND_URL/api/v1/swap", body, token)
                if (resp == null) {
                    _error.value = "Swap failed: backend unreachable."
                    return@launch
                }
                loadWallet()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }

    // ---- Real HTTP + persistence helpers (no third-party deps) ----

    companion object {
        private const val BACKEND_URL = "http://localhost:8443"
    }

    private fun getStoredAddress(): String? = prefs().getString("wallet_address", null)
    private fun getStoredWalletId(): String? = prefs().getString("wallet_id", null)
    private fun getStoredAuthToken(): String? = prefs().getString("auth_token", null)

    private fun prefs() =
        getApplication<android.app.Application>().getSharedPreferences("tigerwallet", android.content.Context.MODE_PRIVATE)

    private fun httpGet(urlString: String): String? = try {
        val conn = (URL(urlString).openConnection() as HttpURLConnection).apply {
            requestMethod = "GET"
            connectTimeout = 10000
            readTimeout = 15000
        }
        conn.inputStream.bufferedReader().use { it.readText() }
    } catch (e: Exception) { null }

    private fun httpPost(urlString: String, body: String, authToken: String): String? = try {
        val conn = (URL(urlString).openConnection() as HttpURLConnection).apply {
            requestMethod = "POST"
            connectTimeout = 10000
            readTimeout = 15000
            doOutput = true
            setRequestProperty("Content-Type", "application/json")
            setRequestProperty("Authorization", "Bearer $authToken")
        }
        conn.outputStream.use { it.write(body.toByteArray()) }
        conn.inputStream.bufferedReader().use { it.readText() }
    } catch (e: Exception) { null }

    private fun extractField(json: String, field: String): String? {
        val key = "\""$field\":\""
        val start = json.indexOf(key)
        if (start < 0) return null
        val s = start + key.length
        val e = json.indexOf('"', s)
        return if (e < 0) null else json.substring(s, e)
    }

    private fun parseTokens(tokensResp: String?, nativeSymbol: String, nativeBalance: String): List<Token> {
        val out = mutableListOf(Token(nativeSymbol, nativeSymbol, nativeBalance, 0.0))
        if (tokensResp.isNullOrEmpty()) return out
        val re = Regex("""\{"symbol":"([^"]+)","name":"([^"]+)","balance":"([^"]+)"""")
        for (m in re.findAll(tokensResp)) {
            out.add(Token(m.groupValues[1], m.groupValues[2], m.groupValues[3], 0.0))
        }
        return out
    }


data class Wallet(
    val address: String,
    val totalBalance: Double,
    val nativeBalance: String,
    val chain: Chain,
    val tokens: List<Token>
)
