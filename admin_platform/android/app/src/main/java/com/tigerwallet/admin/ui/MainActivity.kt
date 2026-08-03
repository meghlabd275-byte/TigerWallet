package com.tigerwallet.admin.ui

import android.content.Intent
import android.os.Bundle
import android.view.Menu
import android.view.MenuItem
import androidx.appcompat.app.AppCompatActivity
import androidx.appcompat.app.AppCompatDelegate
import androidx.fragment.app.Fragment
import com.google.android.material.bottomnavigation.BottomNavigationView
import com.tigerwallet.admin.R
import com.tigerwallet.admin.databinding.ActivityMainBinding
import com.tigerwallet.admin.ui.dashboard.DashboardFragment
import com.tigerwallet.admin.ui.users.UsersFragment
import com.tigerwallet.admin.ui.transactions.TransactionsFragment
import com.tigerwallet.admin.ui.tokens.TokensFragment
import com.tigerwallet.admin.ui.kyc.KYCFragment
import com.tigerwallet.admin.ui.withdrawals.WithdrawalsFragment
import com.tigerwallet.admin.ui.chains.ChainsFragment
import com.tigerwallet.admin.ui.fees.FeesFragment
import com.tigerwallet.admin.ui.whitelabels.WhiteLabelsFragment
import com.tigerwallet.admin.util.PreferencesManager

class MainActivity : AppCompatActivity() {
    private lateinit var binding: ActivityMainBinding
    private lateinit var preferencesManager: PreferencesManager
    
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        
        preferencesManager = PreferencesManager(this)
        
        // Apply theme
        val isDarkMode = preferencesManager.isDarkMode()
        AppCompatDelegate.setDefaultNightMode(
            if (isDarkMode) AppCompatDelegate.MODE_NIGHT_YES 
            else AppCompatDelegate.MODE_NIGHT_NO
        )
        
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)
        
        setSupportActionBar(binding.toolbar)
        
        if (savedInstanceState == null) {
            loadFragment(DashboardFragment())
        }
        
        setupBottomNavigation()
    }
    
    private fun setupBottomNavigation() {
        binding.bottomNavigation.setOnItemSelectedListener { item ->
            val fragment: Fragment = when (item.itemId) {
                R.id.nav_dashboard -> DashboardFragment()
                R.id.nav_users -> UsersFragment()
                R.id.nav_transactions -> TransactionsFragment()
                R.id.nav_tokens -> TokensFragment()
                R.id.nav_kyc -> KYCFragment()
                R.id.nav_withdrawals -> WithdrawalsFragment()
                R.id.nav_chains -> ChainsFragment()
                R.id.nav_fees -> FeesFragment()
                R.id.nav_whitelabels -> WhiteLabelsFragment()
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
        return true
    }
    
    override fun onOptionsItemSelected(item: MenuItem): Boolean {
        return when (item.itemId) {
            R.id.action_logout -> {
                logout()
                true
            }
            R.id.action_toggle_theme -> {
                toggleTheme()
                true
            }
            else -> super.onOptionsItemSelected(item)
        }
    }
    
    private fun logout() {
        preferencesManager.clearSession()
        val intent = Intent(this, LoginActivity::class.java)
        startActivity(intent)
        finish()
    }
    
    private fun toggleTheme() {
        val currentMode = preferencesManager.isDarkMode()
        preferencesManager.setDarkMode(!currentMode)
        
        AppCompatDelegate.setDefaultNightMode(
            if (!currentMode) AppCompatDelegate.MODE_NIGHT_YES 
            else AppCompatDelegate.MODE_NIGHT_NO
        )
        
        recreate()
    }
}
