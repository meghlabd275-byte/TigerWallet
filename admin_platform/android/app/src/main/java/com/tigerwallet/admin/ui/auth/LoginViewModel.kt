package com.tigerwallet.admin.ui.auth

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.tigerwallet.admin.data.api.NetworkClient
import com.tigerwallet.admin.data.api.model.LoginRequest
import com.tigerwallet.admin.util.PreferencesManager
import kotlinx.coroutines.launch

class LoginViewModel : ViewModel() {
    private val apiService = NetworkClient.getApiService()
    
    private val _loginState = MutableLiveData<LoginState>()
    val loginState: LiveData<LoginState> = _loginState
    
    private val _isLoading = MutableLiveData<Boolean>()
    val isLoading: LiveData<Boolean> = _isLoading
    
    fun login(email: String, password: String) {
        if (email.isBlank() || password.isBlank()) {
            _loginState.value = LoginState.Error("Email and password are required")
            return
        }
        
        _isLoading.value = true
        
        viewModelScope.launch {
            try {
                val response = apiService.login(LoginRequest(email, password))
                
                if (response.isSuccessful && response.body() != null) {
                    val loginResponse = response.body()!!
                    _loginState.value = LoginState.Success(loginResponse)
                } else {
                    val errorMsg = response.errorBody()?.string() ?: "Login failed"
                    _loginState.value = LoginState.Error(errorMsg)
                }
            } catch (e: Exception) {
                _loginState.value = LoginState.Error(e.message ?: "Network error")
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun resetState() {
        _loginState.value = LoginState.Idle
    }
}

sealed class LoginState {
    object Idle : LoginState()
    object Loading : LoginState()
    data class Success(val token: String) : LoginState()
    data class Error(val message: String) : LoginState()
}
