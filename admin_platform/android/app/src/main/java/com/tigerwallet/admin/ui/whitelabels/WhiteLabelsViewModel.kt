package com.tigerwallet.admin.ui.whitelabels

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.tigerwallet.admin.data.model.WhiteLabel
import com.tigerwallet.admin.data.repository.AdminRepository
import kotlinx.coroutines.launch

class WhiteLabelsViewModel : ViewModel() {
    private val repository = AdminRepository()
    
    private val _whiteLabels = MutableLiveData<List<WhiteLabel>>()
    val whiteLabels: LiveData<List<WhiteLabel>> = _whiteLabels
    
    private val _isLoading = MutableLiveData<Boolean>()
    val isLoading: LiveData<Boolean> = _isLoading
    
    private val _error = MutableLiveData<String?>()
    val error: LiveData<String?> = _error
    
    fun loadWhiteLabels(status: String? = null) {
        _isLoading.value = true
        _error.value = null
        
        viewModelScope.launch {
            try {
                val response = repository.getWhiteLabels(status = status)
                _whiteLabels.value = response
                _isLoading.value = false
            } catch (e: Exception) {
                _error.value = e.message
                _isLoading.value = false
            }
        }
    }
    
    fun approveWhiteLabel(whiteLabelId: String) {
        viewModelScope.launch {
            try {
                repository.approveWhiteLabel(whiteLabelId)
                loadWhiteLabels()
            } catch (e: Exception) {
                _error.value = e.message
            }
        }
    }
    
    fun suspendWhiteLabel(whiteLabelId: String, reason: String) {
        viewModelScope.launch {
            try {
                repository.suspendWhiteLabel(whiteLabelId, reason)
                loadWhiteLabels()
            } catch (e: Exception) {
                _error.value = e.message
            }
        }
    }
}
