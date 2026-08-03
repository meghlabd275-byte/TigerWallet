package com.tigerwallet.admin.ui.chains

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.tigerwallet.admin.data.model.Chain
import com.tigerwallet.admin.data.repository.AdminRepository
import kotlinx.coroutines.launch

class ChainsViewModel : ViewModel() {
    private val repository = AdminRepository()
    
    private val _chains = MutableLiveData<List<Chain>>()
    val chains: LiveData<List<Chain>> = _chains
    
    private val _isLoading = MutableLiveData<Boolean>()
    val isLoading: LiveData<Boolean> = _isLoading
    
    private val _error = MutableLiveData<String?>()
    val error: LiveData<String?> = _error
    
    fun loadChains() {
        _isLoading.value = true
        _error.value = null
        
        viewModelScope.launch {
            try {
                val response = repository.getChains()
                _chains.value = response
                _isLoading.value = false
            } catch (e: Exception) {
                _error.value = e.message
                _isLoading.value = false
            }
        }
    }
    
    fun toggleChainActive(chainId: String, isActive: Boolean) {
        viewModelScope.launch {
            try {
                repository.updateChain(chainId, isActive)
                loadChains()
            } catch (e: Exception) {
                _error.value = e.message
            }
        }
    }
}
