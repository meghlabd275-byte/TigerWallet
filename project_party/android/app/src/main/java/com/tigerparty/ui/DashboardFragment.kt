package com.tigerparty.ui

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.LinearLayout
import android.widget.TextView
import androidx.fragment.app.Fragment
import com.tigerparty.R
import com.tigerparty.api.ProjectPartyApiService
import com.tigerparty.util.Fmt
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject

/**
 * Dashboard: platform health, aggregate market figures and featured listings.
 *
 * All three come from public backend routes, so the screen renders before the
 * caller signs in.
 */
class DashboardFragment : Fragment() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private lateinit var api: ProjectPartyApiService

    private lateinit var healthView: TextView
    private lateinit var marketView: TextView
    private lateinit var featuredView: LinearLayout
    private lateinit var errorView: TextView

    override fun onCreateView(
        inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?,
    ): View = inflater.inflate(R.layout.fragment_dashboard, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        api = ProjectPartyApiService.getInstance(requireContext())
        healthView = view.findViewById(R.id.dashboardHealth)
        marketView = view.findViewById(R.id.dashboardMarket)
        featuredView = view.findViewById(R.id.dashboardFeatured)
        errorView = view.findViewById(R.id.dashboardError)

        view.findViewById<Button>(R.id.btnRefreshDashboard).setOnClickListener { load() }
        load()
    }

    private fun load() {
        errorView.visibility = View.GONE
        scope.launch {
            try {
                val health = withContext(Dispatchers.IO) { runCatching { api.getHealth() }.getOrNull() }
                healthView.text = if (health == null) {
                    getString(R.string.backend_unreachable)
                } else {
                    getString(R.string.backend_status, Fmt.str(health, "status").ifBlank { "unknown" })
                }

                val market = withContext(Dispatchers.IO) { runCatching { api.getMarket() }.getOrNull() }
                marketView.text = renderMarket(market)

                val featured = withContext(Dispatchers.IO) { runCatching { api.getFeatured() }.getOrNull() }
                renderFeatured(featured)
            } catch (t: Throwable) {
                errorView.text = t.message ?: getString(R.string.load_failed)
                errorView.visibility = View.VISIBLE
            }
        }
    }

    private fun renderMarket(market: JSONObject?): String {
        if (market == null) return getString(R.string.empty_state)
        return buildString {
            append("Listed tokens: ").append(Fmt.str(market, "total_tokens").ifBlank { "0" })
            append("\nActive listings: ").append(Fmt.str(market, "total_listings").ifBlank { "0" })
            append("\nActive launchpads: ").append(Fmt.str(market, "total_launchpads").ifBlank { "0" })
            val volume = Fmt.str(market, "volume_24h")
            if (volume.isNotBlank()) append("\n24h volume: ").append(volume)
        }
    }

    private fun renderFeatured(featured: JSONObject?) {
        featuredView.removeAllViews()
        val arr = Fmt.array(featured, "featured", "tokens", "items")
        if (arr.length() == 0) {
            featuredView.addView(label(getString(R.string.empty_state)))
            return
        }
        for (i in 0 until arr.length()) {
            val token = arr.optJSONObject(i) ?: continue
            val name = Fmt.str(token, "name")
            val symbol = Fmt.str(token, "symbol")
            val chain = Fmt.str(token, "chain")
            featuredView.addView(label(
                listOf(name, symbol, chain).filter { it.isNotBlank() }.joinToString(" · ")
            ))
        }
    }

    private fun label(text: String) = TextView(requireContext()).apply {
        this.text = text
        setPadding(8, 16, 8, 16)
    }

    override fun onDestroyView() {
        scope.cancel()
        super.onDestroyView()
    }
}
