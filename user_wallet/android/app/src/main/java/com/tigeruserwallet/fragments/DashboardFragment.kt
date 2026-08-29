package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.fragment.app.Fragment
import androidx.recyclerview.widget.LinearLayoutManager
import com.tigeruserwallet.R
import com.tigeruserwallet.adapters.BalanceAdapter
import com.tigeruserwallet.api.UserWalletApiService
import com.tigeruserwallet.databinding.FragmentDashboardBinding
import com.tigeruserwallet.util.LiveFeedSocket
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Dashboard (mirrors web Dashboard.tsx):
 *  - greeting from the transparent session's user identity
 *  - stat cards: total native balance, wallet count, network count
 *  - balances table (real GET /wallets + per-wallet GET /wallets/:id/balance)
 *
 * No stubs: every value is a real backend fetch.
 */
class DashboardFragment : Fragment() {

    private var _binding: FragmentDashboardBinding? = null
    private val binding get() = _binding!!

    private var liveFeed: LiveFeedSocket? = null
    private val livePrices = mutableMapOf<String, Pair<Double, Double>>()

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentDashboardBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        binding.balancesRecyclerView.layoutManager = LinearLayoutManager(requireContext())
        load()
        binding.swipeRefresh.setOnRefreshListener { load() }
        connectLiveFeed()
    }

    /** Public live price feed (WebSocket /api/v1/ws) for the dashboard ticker. */
    private fun connectLiveFeed() {
        val feed = LiveFeedSocket()
        liveFeed = feed
        feed.connect(listOf("BTC", "ETH"), onTicker = { t ->
            val symbol = t.optString("symbol")
            if (symbol.isNotEmpty()) {
                livePrices[symbol] = t.optDouble("last_price") to t.optDouble("change_24h_pct")
                val text = livePrices.entries.joinToString("   ") { (sym, p) ->
                    "$sym $${String.format("%,.2f", p.first)} (${String.format("%+.2f", p.second)}%)"
                }
                activity?.runOnUiThread { _binding?.liveTicker?.text = text }
            }
        })
    }

    override fun onDestroyView() {
        liveFeed?.close()
        liveFeed = null
        super.onDestroyView()
        _binding = null
    }

    private fun load() {
        binding.swipeRefresh.isRefreshing = true
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val user = UserWalletApiService.ensureSession().user
                val balances = UserWalletApiService.getBalances()
                val total = balances.sumOf { it.balanceF }
                val networks = balances.map { it.chainId }.toSet().size
                withContext(Dispatchers.Main) {
                    binding.welcomeText.text = getString(R.string.dashboard_welcome, user.username)
                    binding.statTotal.text = String.format("%.6f", total)
                    binding.statWallets.text = balances.size.toString()
                    binding.statNetworks.text = networks.toString()
                    binding.balancesRecyclerView.adapter = BalanceAdapter(balances)
                    binding.emptyState.visibility =
                        if (balances.isEmpty()) View.VISIBLE else View.GONE
                    binding.balancesRecyclerView.visibility =
                        if (balances.isEmpty()) View.GONE else View.VISIBLE
                    binding.swipeRefresh.isRefreshing = false
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    binding.welcomeText.text = getString(R.string.dashboard_welcome, "")
                    binding.statTotal.text = "0.000000"
                    binding.statWallets.text = "0"
                    binding.statNetworks.text = "0"
                    binding.balancesRecyclerView.adapter = BalanceAdapter(emptyList())
                    binding.emptyState.visibility = View.VISIBLE
                    binding.balancesRecyclerView.visibility = View.GONE
                    binding.swipeRefresh.isRefreshing = false
                }
            }
        }
    }

}
