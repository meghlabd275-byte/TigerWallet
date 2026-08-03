package com.tigerwallet.admin.ui.dashboard

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.tigerwallet.admin.data.api.NetworkClient
import com.tigerwallet.admin.data.api.model.DashboardResponse
import kotlinx.coroutines.launch

class DashboardViewModel : ViewModel() {
    private val apiService = NetworkClient.getApiService()
    
    private val _dashboardStats = MutableLiveData<DashboardState>()
    val dashboardStats: LiveData<DashboardState> = _dashboardStats
    
    private val _isLoading = MutableLiveData<Boolean>()
    val isLoading: LiveData<Boolean> = _isLoading
    
    private val _error = MutableLiveData<String?>()
    val error: LiveData<String?> = _error
    
    init {
        loadDashboard()
    }
    
    fun loadDashboard() {
        _isLoading.value = true
        _error.value = null
        
        viewModelScope.launch {
            try {
                val response = apiService.getDashboardStats()
                
                if (response.isSuccessful && response.body() != null) {
                    _dashboardStats.value = DashboardState.Success(response.body()!!)
                } else {
                    _error.value = "Failed to load dashboard"
                    _dashboardStats.value = DashboardState.Error("Failed to load dashboard")
                }
            } catch (e: Exception) {
                _error.value = e.message
                _dashboardStats.value = DashboardState.Error(e.message ?: "Unknown error")
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun refresh() {
        loadDashboard()
    }
}

sealed class DashboardState {
    data class Success(val data: DashboardResponse) : DashboardState()
    data class Error(val message: String) : DashboardState()
    object Loading : DashboardState()
}
