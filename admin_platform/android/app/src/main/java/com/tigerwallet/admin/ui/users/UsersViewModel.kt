package com.tigerwallet.admin.ui.users

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.tigerwallet.admin.data.api.NetworkClient
import com.tigerwallet.admin.data.api.model.UserResponse
import com.tigerwallet.admin.data.api.model.UsersResponse
import kotlinx.coroutines.launch

class UsersViewModel : ViewModel() {
    private val apiService = NetworkClient.getApiService()
    
    private val _usersState = MutableLiveData<UsersState>()
    val usersState: LiveData<UsersState> = _usersState
    
    private val _isLoading = MutableLiveData<Boolean>()
    val isLoading: LiveData<Boolean> = _isLoading
    
    private val _actionResult = MutableLiveData<ActionResult?>()
    val actionResult: LiveData<ActionResult?> = _actionResult
    
    private var currentPage = 1
    private val pageSize = 20
    private var currentStatus: String? = null
    private var currentSearch: String? = null
    
    init {
        loadUsers()
    }
    
    fun loadUsers(status: String? = null, search: String? = null) {
        currentStatus = status
        currentSearch = search
        currentPage = 1
        
        _isLoading.value = true
        
        viewModelScope.launch {
            try {
                val response = apiService.getUsers(
                    page = currentPage,
                    limit = pageSize,
                    status = status,
                    search = search
                )
                
                if (response.isSuccessful && response.body() != null) {
                    _usersState.value = UsersState.Success(response.body()!!)
                } else {
                    _usersState.value = UsersState.Error("Failed to load users")
                }
            } catch (e: Exception) {
                _usersState.value = UsersState.Error(e.message ?: "Unknown error")
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun loadNextPage() {
        if (_isLoading.value == true) return
        
        currentPage++
        _isLoading.value = true
        
        viewModelScope.launch {
            try {
                val response = apiService.getUsers(
                    page = currentPage,
                    limit = pageSize,
                    status = currentStatus,
                    search = currentSearch
                )
                
                if (response.isSuccessful && response.body() != null) {
                    val currentState = _usersState.value
                    if (currentState is UsersState.Success) {
                        val newUsers = currentState.data.data + response.body()!!.data
                        _usersState.value = UsersState.Success(
                            UsersResponse(newUsers, response.body()!!.meta)
                        )
                    }
                }
            } catch (e: Exception) {
                currentPage--
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun suspendUser(userId: String, reason: String) {
        viewModelScope.launch {
            try {
                val response = apiService.suspendUser(userId, com.tigerwallet.admin.data.api.model.SuspendRequest(reason))
                
                if (response.isSuccessful) {
                    _actionResult.value = ActionResult.Success("User suspended successfully")
                    loadUsers(currentStatus, currentSearch)
                } else {
                    _actionResult.value = ActionResult.Error("Failed to suspend user")
                }
            } catch (e: Exception) {
                _actionResult.value = ActionResult.Error(e.message ?: "Error")
            }
        }
    }
    
    fun banUser(userId: String, reason: String) {
        viewModelScope.launch {
            try {
                val response = apiService.banUser(userId, com.tigerwallet.admin.data.api.model.BanRequest(reason))
                
                if (response.isSuccessful) {
                    _actionResult.value = ActionResult.Success("User banned successfully")
                    loadUsers(currentStatus, currentSearch)
                } else {
                    _actionResult.value = ActionResult.Error("Failed to ban user")
                }
            } catch (e: Exception) {
                _actionResult.value = ActionResult.Error(e.message ?: "Error")
            }
        }
    }
    
    fun clearActionResult() {
        _actionResult.value = null
    }
    
    fun search(query: String) {
        loadUsers(currentStatus, query.ifBlank { null })
    }
    
    fun filterByStatus(status: String?) {
        loadUsers(status, currentSearch)
    }
}

sealed class UsersState {
    data class Success(val data: UsersResponse) : UsersState()
    data class Error(val message: String) : UsersState()
    object Loading : UsersState()
}

sealed class ActionResult {
    data class Success(val message: String) : ActionResult()
    data class Error(val message: String) : ActionResult()
}
