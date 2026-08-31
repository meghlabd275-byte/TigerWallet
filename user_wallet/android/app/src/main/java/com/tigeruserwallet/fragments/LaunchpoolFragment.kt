package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.TextView
import android.widget.Toast
import androidx.fragment.app.Fragment
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Launchpool staking: pool info + user stakes, stake/unstake with wallet
 * password (POST /launchpool/stake|unstake — broadcast server-side).
 */
class LaunchpoolFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_launchpool, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val infoText = view.findViewById<TextView>(R.id.launchpoolInfoText)
        val stakesText = view.findViewById<TextView>(R.id.launchpoolStakesText)
        val statusText = view.findViewById<TextView>(R.id.launchpoolStatusText)

        fun load() {
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val pool = UserWalletApiService.getLaunchpool()
                    val stakes = UserWalletApiService.getLaunchpoolStakes()
                    withContext(Dispatchers.Main) {
                        infoText.text = "Pool: ${pool.optString("token", pool.optString("asset", "?"))} | APY: ${pool.optString("apy", "?")}% | TVL: ${pool.optString("tvl", pool.optString("total_staked", "?"))}"
                        stakesText.text = if (stakes.isEmpty()) "No active stakes"
                        else stakes.joinToString("\n") {
                            "\u2022 ${it.optString("id", "?")}: ${it.optString("amount", "?")} ${it.optString("token", "")} (${it.optString("status", "?")})"
                        }
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { infoText.text = "Launchpool data unavailable" }
                }
            }
        }
        load()

        fun act(stake: Boolean) {
            val amount = view.findViewById<EditText>(R.id.launchpoolAmountInput).text.toString().trim()
            val password = view.findViewById<EditText>(R.id.launchpoolPasswordInput).text.toString()
            if (amount.isEmpty() || password.isEmpty()) {
                Toast.makeText(requireContext(), "Enter amount and password", Toast.LENGTH_SHORT).show()
                return
            }
            statusText.text = "Submitting…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val walletId = UserWalletApiService.getWallets().firstOrNull()?.id
                    if (walletId == null) {
                        withContext(Dispatchers.Main) { statusText.text = "No wallet available" }
                        return@launch
                    }
                    val res = if (stake) UserWalletApiService.launchpoolStake(walletId, password, amount)
                              else UserWalletApiService.launchpoolUnstake(walletId, password, amount)
                    withContext(Dispatchers.Main) {
                        val tx = res.optString("tx_hash", "")
                        statusText.text = if (tx.isNotEmpty()) "Transaction submitted to the blockchain network: $tx"
                                          else "${if (stake) "Stake" else "Unstake"} submitted"
                        load()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { statusText.text = "Failed: ${e.message}" }
                }
            }
        }
        view.findViewById<Button>(R.id.launchpoolStakeButton).setOnClickListener { act(true) }
        view.findViewById<Button>(R.id.launchpoolUnstakeButton).setOnClickListener { act(false) }
    }
}
