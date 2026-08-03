package com.tigerwallet.admin.ui.fees

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.tigerwallet.admin.data.model.Fee
import com.tigerwallet.admin.data.repository.AdminRepository
import kotlinx.coroutines.launch

class FeesViewModel : ViewModel() {
    private val repository = AdminRepository()
    
    private val _fees = MutableLiveData<List<Fee>>()
    val fees: LiveData<List<Fee>> = _fees
    
    private val _isLoading = MutableLiveData<Boolean>()
    val isLoading: LiveData<Boolean> = _isLoading
    
    private val _error = MutableLiveData<String?>()
    val error: LiveData<String?> = _error
    
    fun loadFees() {
        _isLoading.value = true
        _error.value = null
        
        viewModelScope.launch {
            try {
                val response = repository.getFees()
                _fees.value = response
                _isLoading.value = false
            } catch (e: Exception) {
                _error.value = e.message
                _isLoading.value = false
            }
        }
    }
    
    fun toggleFeeActive(feeId: String, isActive: Boolean) {
        viewModelScope.launch {
            try {
                repository.updateFee(feeId, isActive)
                loadFees()
            } catch (e: Exception) {
                _error.value = e.message
            }
        }
    }
}
