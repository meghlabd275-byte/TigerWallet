//
//  WalletViewModel.kt
//  TigerWallet - Android Wallet ViewModel
//

package com.tigerwallet.app

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

class WalletViewModel : ViewModel() {
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
                // Simulate API call - in production, this would call the actual API
                val mockWallet = Wallet(
                    address = "0x742d35Cc6634C0532925a3b844Bc9e7595f",
                    totalBalance = 12450.00,
                    nativeBalance = "4.2",
                    chain = _selectedChain.value,
                    tokens = listOf(
                        Token("ETH", "Ethereum", "4.2", 12600.0),
                        Token("USDT", "Tether USD", "1000", 1000.0),
                        Token("BNB", "BNB", "5.2", 1560.0)
                    )
                )
                _wallet.value = mockWallet
            } catch (e: Exception) {
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
                // Simulate transaction
                kotlinx.coroutines.delay(2000)
                // Refresh wallet
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
                // Simulate swap
                kotlinx.coroutines.delay(2000)
                loadWallet()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
}

data class Wallet(
    val address: String,
    val totalBalance: Double,
    val nativeBalance: String,
    val chain: Chain,
    val tokens: List<Token>
)
