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
 * Fee transparency: public fee tier schedule (GET /public/fees) and recent
 * settled fee transactions (GET /public/fees/transactions). Read-only.
 */
class FeesFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_fees, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val tiersText = view.findViewById<TextView>(R.id.feesTiersText)
        val txText = view.findViewById<TextView>(R.id.feesTxText)
        val statusText = view.findViewById<TextView>(R.id.feesStatusText)

        CoroutineScope(Dispatchers.IO).launch {
            try {
                val tiers = UserWalletApiService.getPublicFees()
                val txs = UserWalletApiService.getPublicFeeTransactions()
                val tierArr = tiers.optJSONArray("fees") ?: tiers.optJSONArray("data")
                val txArr = txs.optJSONArray("transactions") ?: txs.optJSONArray("data")
                withContext(Dispatchers.Main) {
                    tiersText.text = if (tierArr == null || tierArr.length() == 0) "No fee tiers configured"
                    else (0 until tierArr.length()).joinToString("\n") { i ->
                        val t = tierArr.getJSONObject(i)
                        "\u2022 ${t.optString("name", t.optString("tier", "?"))}: ${t.optString("rate_bps", t.optString("rate", "?"))} bps"
                    }
                    txText.text = if (txArr == null || txArr.length() == 0) "No settled fee transactions"
                    else "Recent settled fees:\n" + (0 until minOf(txArr.length(), 20)).joinToString("\n") { i ->
                        val t = txArr.getJSONObject(i)
                        "\u2022 ${t.optString("tx_hash", "?").take(14)}… ${t.optString("fee_amount", t.optString("amount", "?"))} ${t.optString("token", "")}"
                    }
                    statusText.text = ""
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) { statusText.text = "Fee data unavailable: ${e.message}" }
            }
        }
    }
}
