package com.tigerwallet.admin.ui.activities

import android.os.Bundle
import android.view.Menu
import android.view.MenuItem
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.fragment.app.Fragment
import com.google.android.material.bottomnavigation.BottomNavigationView
import com.tigerwallet.admin.R
import com.tigerwallet.admin.ui.fragments.*

/**
 * Main Admin Activity
 * Main entry point for the admin application with bottom navigation
 */
class MainActivity : AppCompatActivity() {

    private lateinit var bottomNavigation: BottomNavigationView

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
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

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_login)
        
        // Check if already logged in
        if (TigerAdminApplication.instance.isLoggedIn()) {
            navigateToMain()
            return
        }
        
        setupLoginForm()
    }

    private fun setupLoginForm() {
        // Setup login form with email/password fields
        // On successful login, save session and navigate to main
    }

    private fun navigateToMain() {
        // Navigate to main activity
    }
}

/**
 * Base Activity with common functionality
 */
abstract class BaseActivity : AppCompatActivity() {

    protected fun showLoading() {
        // Show loading dialog
    }

    protected fun hideLoading() {
        // Hide loading dialog
    }

    protected fun showError(message: String) {
        Toast.makeText(this, message, Toast.LENGTH_SHORT).show()
    }

    protected fun showSuccess(message: String) {
        Toast.makeText(this, message, Toast.LENGTH_SHORT).show()
    }

    protected fun isNetworkAvailable(): Boolean {
        // Check network connectivity
        return true
    }
}
