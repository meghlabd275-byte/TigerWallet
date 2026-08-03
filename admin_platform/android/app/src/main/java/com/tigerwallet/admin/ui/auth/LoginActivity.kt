package com.tigerwallet.admin.ui.auth

import android.content.Intent
import android.os.Bundle
import android.view.View
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.ViewModelProvider
import com.tigerwallet.admin.databinding.ActivityLoginBinding
import com.tigerwallet.admin.ui.dashboard.DashboardActivity
import com.tigerwallet.admin.util.PreferencesManager

class LoginActivity : AppCompatActivity() {
    private lateinit var binding: ActivityLoginBinding
    private lateinit var viewModel: LoginViewModel
    private lateinit var preferencesManager: PreferencesManager
    
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityLoginBinding.inflate(layoutInflater)
        setContentView(binding.root)
        
        preferencesManager = PreferencesManager(this)
        
        // Check if already logged in
        if (preferencesManager.isLoggedIn()) {
            navigateToDashboard()
            return
        }
        
        viewModel = ViewModelProvider(this)[LoginViewModel::class.java]
        
        setupObservers()
        setupListeners()
    }
    
    private fun setupObservers() {
        viewModel.isLoading.observe(this) { isLoading ->
            binding.progressBar.visibility = if (isLoading) View.VISIBLE else View.GONE
            binding.btnLogin.isEnabled = !isLoading
        }
        
        viewModel.loginState.observe(this) { state ->
            when (state) {
                is LoginState.Success -> {
                    preferencesManager.saveToken(state.token)
                    preferencesManager.saveAdminId(state.adminId)
                    preferencesManager.saveAdminEmail(state.adminEmail)
                    preferencesManager.saveAdminRole(state.adminRole)
                    navigateToDashboard()
                }
                is LoginState.Error -> {
                    Toast.makeText(this, state.message, Toast.LENGTH_LONG).show()
                }
                else -> {}
            }
        }
    }
    
    private fun setupListeners() {
        binding.btnLogin.setOnClickListener {
            val email = binding.etEmail.text.toString().trim()
            val password = binding.etPassword.text.toString()
            
            if (validateInput(email, password)) {
                viewModel.login(email, password)
            }
        }
    }
    
    private fun validateInput(email: String, password: String): Boolean {
        var isValid = true
        
        if (email.isEmpty()) {
            binding.tilEmail.error = "Email is required"
            isValid = false
        } else if (!android.util.Patterns.EMAIL_ADDRESS.matcher(email).matches()) {
            binding.tilEmail.error = "Invalid email format"
            isValid = false
        } else {
            binding.tilEmail.error = null
        }
        
        if (password.isEmpty()) {
            binding.tilPassword.error = "Password is required"
            isValid = false
        } else if (password.length < 6) {
            binding.tilPassword.error = "Password must be at least 6 characters"
            isValid = false
        } else {
            binding.tilPassword.error = null
        }
        
        return isValid
    }
    
    private fun navigateToDashboard() {
        val intent = Intent(this, DashboardActivity::class.java)
        startActivity(intent)
        finish()
    }
}

// Extended LoginState to include admin details
sealed class LoginState {
    object Idle : LoginState()
    data class Success(val token: String, val adminId: String, val adminEmail: String, val adminRole: String) : LoginState()
    data class Error(val message: String) : LoginState()
}
