package com.tigerwallet.admin.ui.kyc

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.tigerwallet.admin.data.model.KYCSubmission
import com.tigerwallet.admin.data.repository.AdminRepository
import kotlinx.coroutines.launch

class KYCViewModel : ViewModel() {
    private val repository = AdminRepository()
    
    private val _submissions = MutableLiveData<List<KYCSubmission>>()
    val submissions: LiveData<List<KYCSubmission>> = _submissions
    
    private val _isLoading = MutableLiveData<Boolean>()
    val isLoading: LiveData<Boolean> = _isLoading
    
    private val _error = MutableLiveData<String?>()
    val error: LiveData<String?> = _error
    
    fun loadSubmissions(status: String? = null, level: Int? = null) {
        _isLoading.value = true
        _error.value = null
        
        viewModelScope.launch {
            try {
                val response = repository.getKYCSubmissions(status = status, level = level)
                _submissions.value = response
                _isLoading.value = false
            } catch (e: Exception) {
                _error.value = e.message
                _isLoading.value = false
            }
        }
    }
    
    fun approveKYC(submissionId: String, notes: String?) {
        viewModelScope.launch {
            try {
                repository.approveKYC(submissionId, notes)
                loadSubmissions()
            } catch (e: Exception) {
                _error.value = e.message
            }
        }
    }
    
    fun rejectKYC(submissionId: String, reason: String) {
        viewModelScope.launch {
            try {
                repository.rejectKYC(submissionId, reason)
                loadSubmissions()
            } catch (e: Exception) {
                _error.value = e.message
            }
        }
    }
}
