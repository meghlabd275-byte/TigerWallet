package com.tigerbots.ui

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.RadioGroup
import android.widget.TextView
import androidx.fragment.app.Fragment
import com.tigerbots.MainActivity
import com.tigerbots.R
import com.tigerbots.TigerBotsApp
import com.tigerbots.api.BotApiService
import com.tigerbots.util.Fmt
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Settings: backend endpoint, appearance, subscription, and — for callers the
 * backend reports as admin — platform statistics.
 *
 * The admin card is only populated when `adminStats` actually succeeds; a
 * non-admin caller sees it disabled rather than an empty shell.
 */
class SettingsFragment : Fragment() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private lateinit var api: BotApiService

    private lateinit var urlInput: EditText
    private lateinit var statusView: TextView
    private lateinit var subView: TextView
    private lateinit var adminView: TextView
    private lateinit var adminCard: View
    private lateinit var errorView: TextView
    private lateinit var themeGroup: RadioGroup

    override fun onCreateView(
        inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?,
    ): View = inflater.inflate(R.layout.fragment_settings, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        api = BotApiService.getInstance(requireContext())
        urlInput = view.findViewById(R.id.settingsBackendUrl)
        statusView = view.findViewById(R.id.settingsStatus)
        subView = view.findViewById(R.id.settingsSubscription)
        adminView = view.findViewById(R.id.settingsAdmin)
        adminCard = view.findViewById(R.id.settingsAdminCard)
        errorView = view.findViewById(R.id.settingsError)
        themeGroup = view.findViewById(R.id.settingsThemeGroup)

        urlInput.setText(api.currentBaseUrl())

        view.findViewById<Button>(R.id.btnSaveBackend).setOnClickListener { saveBackend() }
        view.findViewById<Button>(R.id.btnLogout).setOnClickListener {
            (activity as? MainActivity)?.logout()
        }

        themeGroup.check(
            when (TigerBotsApp.currentMode(requireContext())) {
                TigerBotsApp.MODE_LIGHT -> R.id.themeLight
                TigerBotsApp.MODE_DARK -> R.id.themeDark
                else -> R.id.themeSystem
            }
        )
        themeGroup.setOnCheckedChangeListener { _, checkedId ->
            val mode = when (checkedId) {
                R.id.themeLight -> TigerBotsApp.MODE_LIGHT
                R.id.themeDark -> TigerBotsApp.MODE_DARK
                else -> TigerBotsApp.MODE_SYSTEM
            }
            TigerBotsApp.writeNightMode(requireContext(), mode)
        }

        load()
    }

    private fun saveBackend() {
        errorView.visibility = View.GONE
        try {
            api.setBaseUrl(urlInput.text.toString())
            statusView.text = getString(R.string.settings_backend_saved)
            load()
        } catch (t: Throwable) {
            errorView.text = t.message ?: getString(R.string.settings_backend_invalid)
            errorView.visibility = View.VISIBLE
        }
    }

    private fun load() {
        scope.launch {
            try {
                val health = withContext(Dispatchers.IO) { runCatching { api.health() }.getOrNull() }
                statusView.text = if (health == null) {
                    getString(R.string.settings_unreachable)
                } else {
                    getString(R.string.settings_reachable, health.status, health.service)
                }

                val sub = withContext(Dispatchers.IO) { runCatching { api.getSubscription() }.getOrNull() }
                subView.text = if (sub == null) {
                    getString(R.string.empty_state)
                } else {
                    val body = sub.optJSONObject("subscription") ?: sub
                    buildString {
                        append("Tier: ").append(Fmt.str(body, "tier").ifBlank { "free" })
                        val maxBots = Fmt.str(body, "max_bots").ifBlank { Fmt.str(body, "bot_limit") }
                        if (maxBots.isNotBlank()) append("\nBot limit: ").append(maxBots)
                        val expires = Fmt.time(body, "expires_at")
                        if (expires.isNotBlank()) append("\nExpires: ").append(expires)
                    }
                }

                // Admin statistics are only meaningful for an admin caller; the
                // backend rejects everyone else, so the card stays hidden.
                val stats = withContext(Dispatchers.IO) { runCatching { api.adminStats() }.getOrNull() }
                if (stats == null) {
                    adminCard.visibility = View.GONE
                } else {
                    adminCard.visibility = View.VISIBLE
                    adminView.text = buildString {
                        append("Users: ").append(Fmt.str(stats, "total_users").ifBlank { "0" })
                        append("\nBots: ").append(Fmt.str(stats, "total_bots").ifBlank { "0" })
                        append("\nRunning: ").append(Fmt.str(stats, "running_bots").ifBlank { "0" })
                    }
                }
            } catch (t: Throwable) {
                errorView.text = t.message ?: getString(R.string.load_failed)
                errorView.visibility = View.VISIBLE
            }
        }
    }

    override fun onDestroyView() {
        scope.cancel()
        super.onDestroyView()
    }
}
