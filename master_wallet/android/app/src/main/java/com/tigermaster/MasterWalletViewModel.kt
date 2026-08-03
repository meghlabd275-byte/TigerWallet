package com.tigermaster

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

class MasterWalletViewModel : ViewModel() {
    private val _masterWallet = MutableStateFlow<MasterWalletData?>(null)
    val masterWallet: StateFlow<MasterWalletData?> = _masterWallet.asStateFlow()
    
    private val _subWallets = MutableStateFlow<List<SubWalletData>>(emptyList())
    val subWallets: StateFlow<List<SubWalletData>> = _subWallets.asStateFlow()
    
    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()
    
    private val _isDarkMode = MutableStateFlow(false)
    val isDarkMode: StateFlow<Boolean> = _isDarkMode.asStateFlow()
    
    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error.asStateFlow()
    
    init {
        loadData()
    }
    
    fun toggleDarkMode() {
        _isDarkMode.value = !_isDarkMode.value
    }
    
    private fun loadData() {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                // Simulate API call - in production, this would call the actual API
                _masterWallet.value = MasterWalletData(
                    address = "0x742d35Cc6634C0532925a3b844Bc9e7595f",
                    totalVolume = "12,500,000",
                    subWalletCount = 15,
                    userCount = 8,
                    pendingTx = 3
                )
                
                _subWallets.value = listOf(
                    SubWalletData("User Wallet 1", "0x111...111", "45,000", "Active"),
                    SubWalletData("User Wallet 2", "0x222...222", "23,500", "Active"),
                    SubWalletData("User Wallet 3", "0x333...333", "12,000", "Inactive"),
                    SubWalletData("User Wallet 4", "0x444...444", "8,750", "Active"),
                    SubWalletData("User Wallet 5", "0x555...555", "5,200", "Active")
                )
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun createSubWallet(name: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                kotlinx.coroutines.delay(1000)
                loadData()
            } catch (e: Exception) {
                _error.value = e.message
            } finally {
                _isLoading.value = false
            }
        }
    }
}

data class MasterWalletData(
    val address: String,
    val totalVolume: String,
    val subWalletCount: Int,
    val userCount: Int,
    val pendingTx: Int
)
