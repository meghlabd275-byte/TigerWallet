package com.tigerwallet.admin.ui.activities

import android.content.Intent
import android.os.Bundle
import android.view.Menu
import android.view.MenuItem
import android.widget.Button
import android.widget.EditText
import android.widget.ProgressBar
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.fragment.app.Fragment
import com.google.android.material.bottomnavigation.BottomNavigationView
import com.tigerwallet.admin.R
import com.tigerwallet.admin.TigerAdminApplication
import com.tigerwallet.admin.data.repository.AdminRepository
import com.tigerwallet.admin.ui.fragments.*
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Main Admin Activity
 * Main entry point for the admin application with bottom navigation
 */
class MainActivity : AppCompatActivity() {

    private lateinit var bottomNavigation: BottomNavigationView

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Require an authenticated session before showing the dashboard.
        if (!TigerAdminApplication.instance.isLoggedIn()) {
            startActivity(Intent(this, LoginActivity::class.java))
            finish()
            return
        }

        setContentView(R.layout.activity_main)

        setupToolbar()
        setupBottomNavigation()

        // Load default fragment
        if (savedInstanceState == null) {
            loadFragment(DashboardFragment())
        }
    }

    private fun setupToolbar() {
        setSupportActionBar(findViewById(R.id.toolbar))
        supportActionBar?.title = "TigerWallet Admin"
    }

    private fun setupBottomNavigation() {
        bottomNavigation = findViewById(R.id.bottom_navigation)
        
        bottomNavigation.setOnItemSelectedListener { item ->
            val fragment: Fragment = when (item.itemId) {
                R.id.nav_dashboard -> DashboardFragment()
                R.id.nav_users -> UsersFragment()
                R.id.nav_transactions -> TransactionsFragment()
                R.id.nav_kyc -> KYCFragment()
                R.id.nav_tokens -> TokensFragment()
                R.id.nav_withdrawals -> WithdrawalsFragment()
                R.id.nav_system -> SystemFragment()
                else -> DashboardFragment()
            }
            loadFragment(fragment)
            true
        }
    }

    private fun loadFragment(fragment: Fragment) {
        supportFragmentManager.beginTransaction()
            .replace(R.id.fragment_container, fragment)
            .commit()
    }

    override fun onCreateOptionsMenu(menu: Menu): Boolean {
        menuInflater.inflate(R.menu.main_menu, menu)
        // Add a "Domains" submenu for the 12 admin domain screens.
        val domains = menu.addSubMenu(0, MENU_DOMAINS_GROUP, 0, "Domains")
        domainFragments.forEachIndexed { index, entry ->
            domains.add(0, MENU_DOMAIN_BASE + index, 0, entry.first)
        }
        return true
    }

    override fun onOptionsItemSelected(item: MenuItem): Boolean {
        val domainIndex = item.itemId - MENU_DOMAIN_BASE
        if (domainIndex in domainFragments.indices) {
            loadFragment(domainFragments[domainIndex].second.invoke())
            supportActionBar?.title = domainFragments[domainIndex].first
            return true
        }
        return when (item.itemId) {
            R.id.action_notifications -> {
                // Open notifications
                true
            }
            R.id.action_settings -> {
                // Open settings
                true
            }
            R.id.action_logout -> {
                logout()
                true
            }
            else -> super.onOptionsItemSelected(item)
        }
    }

    companion object {
        private const val MENU_DOMAINS_GROUP = 1001
        private const val MENU_DOMAIN_BASE = 2000

        // Domain title -> fragment factory. Wired into the toolbar "Domains" submenu.
        private val domainFragments: List<Pair<String, () -> Fragment>> = listOf(
            "Futures" to { FuturesFragment() },
            "Options" to { OptionsFragment() },
            "Copy Trading" to { CopyTradingFragment() },
            "Convert" to { ConvertFragment() },
            "On-Ramp" to { OnRampFragment() },
            "Off-Ramp" to { OffRampFragment() },
            "P2P Clients" to { P2PClientsFragment() },
            "P2P Merchants" to { P2PMerchantsFragment() },
            "Partners" to { PartnersFragment() },
            "Rewards" to { RewardsFragment() },
            "Marketing" to { MarketingFragment() },
            "Roles" to { RolesFragment() },
            "Permissions" to { PermissionsFragment() }
        )
    }

    private fun logout() {
        TigerAdminApplication.instance.logout()
        finish()
    }

    fun showToast(message: String) {
        Toast.makeText(this, message, Toast.LENGTH_SHORT).show()
    }
}

/**
 * Login Activity
 * Handles admin authentication
 */
class LoginActivity : AppCompatActivity() {

    private lateinit var emailInput: EditText
    private lateinit var passwordInput: EditText
    private lateinit var loginButton: Button
    private lateinit var progressBar: ProgressBar
    private lateinit var adminRepository: AdminRepository

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_login)

        // Check if already logged in
        if (TigerAdminApplication.instance.isLoggedIn()) {
            navigateToMain()
            return
        }

        emailInput = findViewById(R.id.emailInput)
        passwordInput = findViewById(R.id.passwordInput)
        loginButton = findViewById(R.id.loginButton)
        progressBar = findViewById(R.id.loginProgress)
        adminRepository = AdminRepository(TigerAdminApplication.instance.getApiService())

        setupLoginForm()
    }

    private fun setupLoginForm() {
        loginButton.setOnClickListener {
            val email = emailInput.text.toString().trim()
            val password = passwordInput.text.toString()
            if (email.isEmpty() || password.isEmpty()) {
                Toast.makeText(this, R.string.login_error_empty, Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            performLogin(email, password)
        }
    }

    private fun performLogin(email: String, password: String) {
        setLoading(true)
        CoroutineScope(Dispatchers.Main).launch {
            try {
                val result = withContext(Dispatchers.IO) {
                    adminRepository.login(email, password)
                }
                result.onSuccess { loginResponse ->
                    TigerAdminApplication.instance.sessionManager.saveSession(
                        authToken = loginResponse.token,
                        refreshToken = loginResponse.refresh_token,
                        expiresAt = loginResponse.expires_at,
                        adminUser = loginResponse.admin
                    )
                    navigateToMain()
                }.onFailure { error ->
                    Toast.makeText(this@LoginActivity, error.message ?: getString(R.string.login_error_generic), Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Toast.makeText(this@LoginActivity, e.message ?: getString(R.string.login_error_generic), Toast.LENGTH_SHORT).show()
            } finally {
                setLoading(false)
            }
        }
    }

    private fun setLoading(loading: Boolean) {
        loginButton.isEnabled = !loading
        progressBar.visibility = if (loading) ProgressBar.VISIBLE else ProgressBar.GONE
    }

    private fun navigateToMain() {
        startActivity(Intent(this, MainActivity::class.java))
        finish()
    }
}

/**
 * Base Activity with common functionality
 */
abstract class BaseActivity : AppCompatActivity() {

    protected fun showError(message: String) {
        Toast.makeText(this, message, Toast.LENGTH_SHORT).show()
    }

    protected fun showSuccess(message: String) {
        Toast.makeText(this, message, Toast.LENGTH_SHORT).show()
    }

    protected fun isNetworkAvailable(): Boolean {
        val cm = getSystemService(android.content.Context.CONNECTIVITY_SERVICE)
                as android.net.ConnectivityManager
        val network = cm.activeNetwork ?: return false
        val capabilities = cm.getNetworkCapabilities(network) ?: return false
        return capabilities.hasCapability(android.net.NetworkCapabilities.NET_CAPABILITY_INTERNET)
    }
}
