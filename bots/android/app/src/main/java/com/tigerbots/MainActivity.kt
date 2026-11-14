package com.tigerbots

import android.content.Intent
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
import com.tigerbots.api.BotApiException
import com.tigerbots.api.BotApiService
import com.tigerbots.ui.BotsFragment
import com.tigerbots.ui.DashboardFragment
import com.tigerbots.ui.MarketFragment
import com.tigerbots.ui.SettingsFragment
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
    private lateinit var api: BotApiService

    private lateinit var authContainer: View
    private lateinit var shellContainer: View
    private lateinit var bottomNav: BottomNavigationView
    private lateinit var progress: ProgressBar

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        api = BotApiService.getInstance(this)

        authContainer = findViewById(R.id.authContainer)
        shellContainer = findViewById(R.id.shellContainer)
        bottomNav = findViewById(R.id.bottomNavigation)
        progress = findViewById(R.id.progress)

        wireAuthForm()
        bottomNav.setOnItemSelectedListener { item ->
            when (item.itemId) {
                R.id.nav_dashboard -> { show(DashboardFragment()); true }
                R.id.nav_bots -> { show(BotsFragment()); true }
                R.id.nav_market -> { show(MarketFragment()); true }
                R.id.nav_settings -> { show(SettingsFragment()); true }
                else -> false
            }
        }

        if (api.getToken().isNullOrBlank()) {
            showAuth()
        } else {
            showShell()
        }
    }

    private fun wireAuthForm() {
        val email = findViewById<EditText>(R.id.authEmail)
        val password = findViewById<EditText>(R.id.authPassword)
        val signIn = findViewById<Button>(R.id.btnSignIn)
        val register = findViewById<Button>(R.id.btnRegister)
        val error = findViewById<TextView>(R.id.authError)

        signIn.setOnClickListener {
            val e = email.text.toString().trim()
            val p = password.text.toString()
            if (e.isBlank() || p.isBlank()) {
                error.text = getString(R.string.auth_missing_fields)
                error.visibility = View.VISIBLE
                return@setOnClickListener
            }
            submit(error) { api.login(e, p) }
        }

        register.setOnClickListener {
            val e = email.text.toString().trim()
            val p = password.text.toString()
            if (e.isBlank() || p.length < 8) {
                error.text = getString(R.string.auth_password_too_short)
                error.visibility = View.VISIBLE
                return@setOnClickListener
            }
            submit(error) { api.register(e, p) }
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
                    is BotApiException -> t.detail.ifBlank { t.message ?: "Request failed" }
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
            try { withContext(Dispatchers.IO) { api.logout() } } catch (_: Throwable) { api.clearToken() }
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
