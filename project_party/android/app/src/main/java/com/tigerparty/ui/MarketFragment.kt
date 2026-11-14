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
 * Market: live pricing, 24h analytics and the market-making order book.
 *
 * The figures come from the pricing and analytics routes, which read the
 * platform's own records; nothing is computed locally.
 */
class MarketFragment : Fragment() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private lateinit var api: ProjectPartyApiService

    private lateinit var analyticsView: TextView
    private lateinit var ordersView: LinearLayout
    private lateinit var errorView: TextView

    override fun onCreateView(
        inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?,
    ): View = inflater.inflate(R.layout.fragment_market, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        api = ProjectPartyApiService.getInstance(requireContext())
        analyticsView = view.findViewById(R.id.marketAnalytics)
        ordersView = view.findViewById(R.id.marketOrders)
        errorView = view.findViewById(R.id.marketError)

        view.findViewById<Button>(R.id.btnRefreshMarket).setOnClickListener { load() }
        load()
    }

    private fun load() {
        errorView.visibility = View.GONE
        scope.launch {
            try {
                val volume = withContext(Dispatchers.IO) { runCatching { api.getTradingVolume() }.getOrNull() }
                val liquidity = withContext(Dispatchers.IO) { runCatching { api.getLiquidity() }.getOrNull() }
                val holders = withContext(Dispatchers.IO) { runCatching { api.getHolderCount() }.getOrNull() }
                val txCount = withContext(Dispatchers.IO) { runCatching { api.getTransactionCount() }.getOrNull() }
                analyticsView.text = renderAnalytics(volume, liquidity, holders, txCount)

                val orders = withContext(Dispatchers.IO) { runCatching { api.getMakerOrders() }.getOrNull() }
                renderOrders(orders)
            } catch (t: Throwable) {
                showError(t.message ?: getString(R.string.load_failed))
            }
        }
    }

    private fun renderAnalytics(
        volume: JSONObject?,
        liquidity: JSONObject?,
        holders: JSONObject?,
        txCount: JSONObject?,
    ): String = buildString {
        append("24h volume: ").append(firstValue(volume, "volume_24h", "volume", "total"))
        append("\nLiquidity: ").append(firstValue(liquidity, "liquidity", "total_liquidity", "total"))
        append("\nHolders: ").append(firstValue(holders, "holders", "count", "total"))
        append("\nTransactions: ").append(firstValue(txCount, "transactions", "count", "total"))
    }

    private fun firstValue(o: JSONObject?, vararg keys: String): String {
        if (o == null) return "—"
        for (k in keys) {
            val v = Fmt.str(o, k)
            if (v.isNotBlank()) return v
        }
        return "—"
    }

    private fun renderOrders(payload: JSONObject?) {
        ordersView.removeAllViews()
        val arr = Fmt.array(payload, "orders", "items", "data")
        if (arr.length() == 0) {
            ordersView.addView(label(getString(R.string.empty_state)))
            return
        }
        for (i in 0 until arr.length()) {
            val order = arr.optJSONObject(i) ?: continue
            val side = Fmt.str(order, "side")
            val price = Fmt.str(order, "price")
            val qty = Fmt.str(order, "quantity").ifBlank { Fmt.str(order, "amount") }
            val status = Fmt.str(order, "status")
            ordersView.addView(label(
                listOf(side, price, qty, status).filter { it.isNotBlank() }.joinToString(" · ")
            ))
        }
    }

    private fun label(text: String) = TextView(requireContext()).apply {
        this.text = text
        setPadding(8, 12, 8, 12)
    }

    private fun showError(message: String) {
        errorView.text = message
        errorView.visibility = View.VISIBLE
    }

    override fun onDestroyView() {
        scope.cancel()
        super.onDestroyView()
    }
}
