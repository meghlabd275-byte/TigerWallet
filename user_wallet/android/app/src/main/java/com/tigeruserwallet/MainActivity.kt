package com.tigeruserwallet

import android.os.Bundle
import android.view.Menu
import android.view.MenuItem
import androidx.appcompat.app.AppCompatActivity
import androidx.appcompat.app.AppCompatDelegate
import androidx.fragment.app.Fragment
import com.tigeruserwallet.databinding.ActivityMainBinding
import com.tigeruserwallet.fragments.DashboardFragment
import com.tigeruserwallet.fragments.SendFragment
import com.tigeruserwallet.fragments.StartFragment
import com.tigeruserwallet.fragments.WalletsFragment
import com.tigeruserwallet.fragments.TransactionsFragment
import com.tigeruserwallet.fragments.SettingsFragment
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class MainActivity : AppCompatActivity(), StartFragment.StartHost {
    private lateinit var binding: ActivityMainBinding

    // Allow hosted fragments to switch screens (e.g. Dashboard -> KYC) without
    // exposing the private loadFragment internals.
    fun navigateTo(fragment: Fragment) {
        loadFragment(fragment)
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        UserWalletApiService.init(this)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        setSupportActionBar(binding.toolbar)

        if (savedInstanceState == null) {
            showStartOrDashboard()
        }

        binding.bottomNavigation.setOnItemSelectedListener { item ->
            when (item.itemId) {
                R.id.nav_dashboard -> loadFragment(DashboardFragment())
                R.id.nav_wallets -> loadFragment(WalletsFragment())
                R.id.nav_send -> loadFragment(SendFragment())
                R.id.nav_transactions -> loadFragment(TransactionsFragment())
                R.id.nav_settings -> loadFragment(SettingsFragment())
            }
            true
        }
    }

    // First-open gate: if a token + wallets already exist, unlock straight into
    // Dashboard (no passcode/biometric infra -> "unlock" == go to Dashboard).
    // Otherwise show the guest "Get Started" screen.
    private fun showStartOrDashboard() {
        if (UserWalletApiService.isAuthenticated()) {
            CoroutineScope(Dispatchers.IO).launch {
                val hasWallets = try {
                    UserWalletApiService.getWallets().isNotEmpty()
                } catch (e: Exception) {
                    false
                }
                withContext(Dispatchers.Main) {
                    if (hasWallets) loadFragment(DashboardFragment())
                    else loadFragment(StartFragment())
                }
            }
        } else {
            loadFragment(StartFragment())
        }
    }

    // StartFragment.StartHost: guestAuth already ran inside StartFragment, so by
    // the time we get here a token is persisted. Open WalletsFragment in the
    // requested create/import mode.
    override fun onGuestReady(mode: StartFragment.Mode) {
        val walletMode = when (mode) {
            StartFragment.Mode.IMPORT -> WalletsFragment.Mode.IMPORT
            else -> WalletsFragment.Mode.CREATE
        }
        loadFragment(WalletsFragment().apply { entryMode = walletMode })
        binding.bottomNavigation.selectedItemId = R.id.nav_wallets
    }

    private fun loadFragment(fragment: Fragment): Boolean {
        supportFragmentManager.beginTransaction()
            .replace(R.id.fragmentContainer, fragment)
            .commit()
        return true
    }

    override fun onCreateOptionsMenu(menu: Menu): Boolean {
        menuInflater.inflate(R.menu.menu_main, menu)
        return true
    }

    override fun onOptionsItemSelected(item: MenuItem): Boolean {
        return when (item.itemId) {
            R.id.action_toggle_theme -> {
                toggleTheme()
                true
            }
            else -> super.onOptionsItemSelected(item)
        }
    }

    private fun toggleTheme() {
        val currentMode = AppCompatDelegate.getDefaultNightMode()
        val newMode = if (currentMode == AppCompatDelegate.MODE_NIGHT_YES) {
            AppCompatDelegate.MODE_NIGHT_NO
        } else {
            AppCompatDelegate.MODE_NIGHT_YES
        }
        AppCompatDelegate.setDefaultNightMode(newMode)
    }
}
