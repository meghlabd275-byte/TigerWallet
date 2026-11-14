package com.tigerparty

import android.os.Bundle
import android.view.View
import android.widget.Button
import android.widget.EditText
import android.widget.ProgressBar
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.fragment.app.Fragment
import com.google.android.material.bottomnavigation.BottomNavigationView
import com.tigerparty.api.ProjectPartyApiException
import com.tigerparty.api.ProjectPartyApiService
import com.tigerparty.ui.DashboardFragment
import com.tigerparty.ui.LaunchpadFragment
import com.tigerparty.ui.MarketFragment
import com.tigerparty.ui.SettingsFragment
import com.tigerparty.ui.TokensFragment
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Single-activity host.
 *
 * Shows a real login/registration gate until the API client holds a token, then
 * swaps to the authenticated shell. The token lives in encrypted prefs, so a
 * returning user goes straight to the shell.
 */
class MainActivity : AppCompatActivity() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private lateinit var api: ProjectPartyApiService

    private lateinit var authContainer: View
    private lateinit var shellContainer: View
    private lateinit var bottomNav: BottomNavigationView
    private lateinit var progress: ProgressBar

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        api = ProjectPartyApiService.getInstance(this)

        authContainer = findViewById(R.id.authContainer)
        shellContainer = findViewById(R.id.shellContainer)
        bottomNav = findViewById(R.id.bottomNavigation)
        progress = findViewById(R.id.progress)

        wireAuthForm()
        bottomNav.setOnItemSelectedListener { item ->
            when (item.itemId) {
                R.id.nav_dashboard -> { show(DashboardFragment()); true }
                R.id.nav_tokens -> { show(TokensFragment()); true }
                R.id.nav_launchpad -> { show(LaunchpadFragment()); true }
                R.id.nav_market -> { show(MarketFragment()); true }
                R.id.nav_settings -> { show(SettingsFragment()); true }
                else -> false
            }
        }

        if (api.getToken().isNullOrBlank()) showAuth() else showShell()
    }

    private fun wireAuthForm() {
        val username = findViewById<EditText>(R.id.authEmail)
        val password = findViewById<EditText>(R.id.authPassword)
        val signIn = findViewById<Button>(R.id.btnSignIn)
        val register = findViewById<Button>(R.id.btnRegister)
        val error = findViewById<TextView>(R.id.authError)

        signIn.setOnClickListener {
            val u = username.text.toString().trim()
            val p = password.text.toString()
            if (u.isBlank() || p.isBlank()) {
                error.text = getString(R.string.auth_missing_fields)
                error.visibility = View.VISIBLE
                return@setOnClickListener
            }
            submit(error) { api.login(u, p) }
        }

        register.setOnClickListener {
            val u = username.text.toString().trim()
            val p = password.text.toString()
            // The backend enforces username >= 3 and password >= 8; checking here
            // turns a round-trip into immediate feedback.
            if (u.length < 3) {
                error.text = getString(R.string.auth_username_too_short)
                error.visibility = View.VISIBLE
                return@setOnClickListener
            }
            if (p.length < 8) {
                error.text = getString(R.string.auth_password_too_short)
                error.visibility = View.VISIBLE
                return@setOnClickListener
            }
            submit(error) { api.register(u, p) }
        }
    }

    private fun submit(errorView: TextView, call: suspend () -> Any) {
        errorView.visibility = View.GONE
        progress.visibility = View.VISIBLE
        scope.launch {
            try {
                withContext(Dispatchers.IO) { call() }
                progress.visibility = View.GONE
                if (api.getToken().isNullOrBlank()) {
                    errorView.text = getString(R.string.auth_no_token)
                    errorView.visibility = View.VISIBLE
                } else {
                    showShell()
                }
            } catch (t: Throwable) {
                progress.visibility = View.GONE
                errorView.text = when (t) {
                    is ProjectPartyApiException -> t.detail.ifBlank { t.message ?: "Request failed" }
                    else -> t.message ?: getString(R.string.auth_failed)
                }
                errorView.visibility = View.VISIBLE
            }
        }
    }

    private fun showAuth() {
        authContainer.visibility = View.VISIBLE
        shellContainer.visibility = View.GONE
    }

    fun showShell() {
        authContainer.visibility = View.GONE
        shellContainer.visibility = View.VISIBLE
        if (supportFragmentManager.findFragmentById(R.id.fragmentContainer) == null) {
            bottomNav.selectedItemId = R.id.nav_dashboard
        }
    }

    fun logout() {
        scope.launch {
            api.clearToken()
            supportFragmentManager.beginTransaction()
                .replace(R.id.fragmentContainer, Fragment())
                .commitAllowingStateLoss()
            showAuth()
            Toast.makeText(this@MainActivity, R.string.signed_out, Toast.LENGTH_SHORT).show()
        }
    }

    private fun show(fragment: Fragment) {
        supportFragmentManager.beginTransaction()
            .replace(R.id.fragmentContainer, fragment)
            .commit()
    }

    override fun onDestroy() {
        scope.cancel()
        super.onDestroy()
    }
}
