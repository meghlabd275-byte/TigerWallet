package com.tigerwallet.admin.ui.withdrawals

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.tigerwallet.admin.data.model.Withdrawal
import com.tigerwallet.admin.data.repository.AdminRepository
import kotlinx.coroutines.launch

class WithdrawalsViewModel : ViewModel() {
    private val repository = AdminRepository()
    
    private val _withdrawals = MutableLiveData<List<Withdrawal>>()
    val withdrawals: LiveData<List<Withdrawal>> = _withdrawals
    
    private val _isLoading = MutableLiveData<Boolean>()
    val isLoading: LiveData<Boolean> = _isLoading
    
    private val _error = MutableLiveData<String?>()
    val error: LiveData<String?> = _error
    
    fun loadWithdrawals(status: String? = null) {
        _isLoading.value = true
        _error.value = null
        
        viewModelScope.launch {
            try {
                val response = repository.getWithdrawals(status = status)
                _withdrawals.value = response
                _isLoading.value = false
            } catch (e: Exception) {
                _error.value = e.message
                _isLoading.value = false
            }
        }
    }
    
    fun approveWithdrawal(withdrawalId: String) {
        viewModelScope.launch {
            try {
                repository.approveWithdrawal(withdrawalId)
                loadWithdrawals()
            } catch (e: Exception) {
                _error.value = e.message
            }
        }
    }
    
    fun rejectWithdrawal(withdrawalId: String, reason: String) {
        viewModelScope.launch {
            try {
                repository.rejectWithdrawal(withdrawalId, reason)
                loadWithdrawals()
            } catch (e: Exception) {
                _error.value = e.message
            }
        }
    }
}
