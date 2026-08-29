package com.tigeruserwallet

import android.os.Bundle
import android.view.Menu
import android.view.MenuItem
import androidx.appcompat.app.AppCompatActivity
import androidx.fragment.app.Fragment
import com.tigeruserwallet.api.UserWalletApiService
import com.tigeruserwallet.crypto.GoogleDriveBackup
import com.tigeruserwallet.databinding.ActivityMainBinding
import com.tigeruserwallet.fragments.DashboardFragment
import com.tigeruserwallet.fragments.FeaturesFragment
import com.tigeruserwallet.fragments.OnboardingFragment
import com.tigeruserwallet.fragments.SendFragment
import com.tigeruserwallet.fragments.SettingsFragment
import com.tigeruserwallet.fragments.TransactionsFragment
import com.tigeruserwallet.fragments.WalletsFragment
import com.tigeruserwallet.ui.ThemeManager

/**
 * No-registration entry: the app opens directly to the Create/Import onboarding
 * screen when no local wallet exists, else to the Dashboard. There is NO
 * login/register form — a transparent ephemeral session is auto-provisioned
 * behind the scenes (UserWalletApiService.ensureSession) before any backend
 * call. Mirrors web App.tsx.
 */
class MainActivity : AppCompatActivity() {
    private lateinit var binding: ActivityMainBinding

    override fun onCreate(savedInstanceState: Bundle?) {
        ThemeManager.init(this)
        super.onCreate(savedInstanceState)
        UserWalletApiService.init(this)
        GoogleDriveBackup.config = GoogleDriveBackup.Config(BuildConfig.GOOGLE_WEB_CLIENT_ID)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        setSupportActionBar(binding.toolbar)
        binding.bottomNavigation.setOnItemSelectedListener { item ->
            val frag: Fragment = when (item.itemId) {
                R.id.nav_dashboard -> DashboardFragment()
                R.id.nav_wallets -> WalletsFragment()
                R.id.nav_send -> SendFragment()
                R.id.nav_transactions -> TransactionsFragment()
                R.id.nav_more -> FeaturesFragment()
                R.id.nav_settings -> SettingsFragment()
                else -> DashboardFragment()
            }
            loadFragment(frag)
            true
        }

        if (savedInstanceState == null) {
            showEntry()
        }
    }

    /**
     * Gate: if no local wallet exists -> Onboarding (Create/Import). Else ->
     * Dashboard. The bottom nav is hidden during onboarding.
     */
    private fun showEntry() {
        if (UserWalletApiService.isOnboarded()) {
            binding.bottomNavigation.visibility = android.view.View.VISIBLE
            loadFragment(DashboardFragment())
        } else {
            binding.bottomNavigation.visibility = android.view.View.GONE
            loadFragment(OnboardingFragment())
        }
    }

    private fun loadFragment(fragment: Fragment): Boolean {
        supportFragmentManager.beginTransaction()
            .replace(R.id.fragmentContainer, fragment)
            .commit()
        return true
    }

    /** Push a feature fragment on the back stack (used by FeaturesFragment). */
    fun navigateToFeature(fragment: Fragment) {
        supportFragmentManager.beginTransaction()
            .replace(R.id.fragmentContainer, fragment)
            .addToBackStack(null)
            .commit()
    }

    /** Called by OnboardingFragment after a wallet is created/imported. */
    fun enterApp() {
        binding.bottomNavigation.visibility = android.view.View.VISIBLE
        binding.bottomNavigation.selectedItemId = R.id.nav_dashboard
        loadFragment(DashboardFragment())
    }

    /** Show the onboarding screen (e.g. after a full reset from Settings). */
    fun showOnboarding() {
        binding.bottomNavigation.visibility = android.view.View.GONE
        loadFragment(OnboardingFragment())
    }

    override fun onCreateOptionsMenu(menu: Menu): Boolean {
        menuInflater.inflate(R.menu.menu_main, menu)
        return true
    }

    override fun onOptionsItemSelected(item: MenuItem): Boolean = when (item.itemId) {
        R.id.action_toggle_theme -> {
            ThemeManager.toggle()
            recreate()
            true
        }
        else -> super.onOptionsItemSelected(item)
    }
}
