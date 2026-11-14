package com.tigerbots.ui

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.TextView
import androidx.fragment.app.Fragment
import com.tigerbots.R
import com.tigerbots.api.BotApiService
import com.tigerbots.util.Fmt
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject

/**
 * Dashboard: live backend health, the caller's bot summary and subscription.
 *
 * Every value is read from the backend on load; nothing is precomputed.
 */
class DashboardFragment : Fragment() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private lateinit var api: BotApiService

    private lateinit var healthView: TextView
    private lateinit var botsView: TextView
    private lateinit var subView: TextView
    private lateinit var errorView: TextView

    override fun onCreateView(
        inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?,
    ): View = inflater.inflate(R.layout.fragment_dashboard, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        api = BotApiService.getInstance(requireContext())
        healthView = view.findViewById(R.id.dashboardHealth)
        botsView = view.findViewById(R.id.dashboardBots)
        subView = view.findViewById(R.id.dashboardSubscription)
        errorView = view.findViewById(R.id.dashboardError)

        view.findViewById<Button>(R.id.btnRefreshDashboard).setOnClickListener { load() }
        load()
    }

    private fun load() {
        errorView.visibility = View.GONE
        scope.launch {
            try {
                val (health, bots, sub) = withContext(Dispatchers.IO) {
                    Triple(api.health(), api.listBots(), runCatching { api.getSubscription() }.getOrNull())
                }
                healthView.text = buildString {
                    append("Status: ").append(health.status.ifBlank { "unknown" })
                    if (health.service.isNotBlank()) append("\nService: ").append(health.service)
                }

                val arr = Fmt.array(bots, "bots", "items", "data")
                var running = 0
                var stopped = 0
                for (i in 0 until arr.length()) {
                    when (Fmt.status(arr.optJSONObject(i))) {
                        "running", "active" -> running++
                        else -> stopped++
                    }
                }
                botsView.text = buildString {
                    append("Total: ").append(arr.length())
                    append("\nRunning: ").append(running)
                    append("\nInactive: ").append(stopped)
                }

                subView.text = renderSubscription(sub)
            } catch (t: Throwable) {
                errorView.text = t.message ?: getString(R.string.load_failed)
                errorView.visibility = View.VISIBLE
            }
        }
    }

    private fun renderSubscription(o: JSONObject?): String {
        if (o == null) return "Unavailable"
        val sub = o.optJSONObject("subscription") ?: o
        val tier = Fmt.str(sub, "tier").ifBlank { "free" }
        val bots = Fmt.str(sub, "max_bots").ifBlank { Fmt.str(sub, "bot_limit") }
        return buildString {
            append("Tier: ").append(tier)
            if (bots.isNotBlank()) append("\nBot limit: ").append(bots)
            val expires = Fmt.time(sub, "expires_at", "expiresAt")
            if (expires.isNotBlank()) append("\nExpires: ").append(expires)
        }
    }

    override fun onDestroyView() {
        scope.cancel()
        super.onDestroyView()
    }
}
