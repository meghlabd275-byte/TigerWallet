package com.tigerwallet.admin.ui.tokens

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.tigerwallet.admin.data.model.Token
import com.tigerwallet.admin.data.repository.AdminRepository
import kotlinx.coroutines.launch

class TokensViewModel : ViewModel() {
    private val repository = AdminRepository()
    
    private val _tokens = MutableLiveData<List<Token>>()
    val tokens: LiveData<List<Token>> = _tokens
    
    private val _isLoading = MutableLiveData<Boolean>()
    val isLoading: LiveData<Boolean> = _isLoading
    
    private val _error = MutableLiveData<String?>()
    val error: LiveData<String?> = _error
    
    fun loadTokens() {
        _isLoading.value = true
        _error.value = null
        
        viewModelScope.launch {
            try {
                val response = repository.getTokens()
                _tokens.value = response
                _isLoading.value = false
            } catch (e: Exception) {
                _error.value = e.message
                _isLoading.value = false
            }
        }
    }
    
    fun verifyToken(tokenId: String) {
        viewModelScope.launch {
            try {
                repository.verifyToken(tokenId)
                loadTokens()
            } catch (e: Exception) {
                _error.value = e.message
            }
        }
    }
    
    fun deleteToken(tokenId: String) {
        viewModelScope.launch {
            try {
                repository.deleteToken(tokenId)
                loadTokens()
            } catch (e: Exception) {
                _error.value = e.message
            }
        }
    }
}
