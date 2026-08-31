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
 * Copy trading: list top traders (GET /copytrading/traders), follow and stop.
 */
class CopyTradingFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_copy_trading, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val tradersText = view.findViewById<TextView>(R.id.copyTradersText)
        val statusText = view.findViewById<TextView>(R.id.copyStatusText)

        fun load() {
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val traders = UserWalletApiService.getCopyTraders()
                    withContext(Dispatchers.Main) {
                        tradersText.text = if (traders.isEmpty()) "No traders available"
                        else traders.joinToString("\n") {
                            "\u2022 ${it.optString("id", it.optString("trader_id", "?"))}: ${it.optString("name", it.optString("address", "?"))} roi:${it.optString("roi", "?")}%"
                        }
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { tradersText.text = "Traders unavailable" }
                }
            }
        }
        load()

        view.findViewById<Button>(R.id.copyFollowButton).setOnClickListener {
            val traderId = view.findViewById<EditText>(R.id.copyTraderIdInput).text.toString().trim()
            val allocation = view.findViewById<EditText>(R.id.copyAllocationInput).text.toString().trim()
            if (traderId.isEmpty()) {
                Toast.makeText(requireContext(), "Enter trader ID", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            statusText.text = "Following…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    UserWalletApiService.followTrader(traderId, allocation.ifEmpty { null })
                    withContext(Dispatchers.Main) {
                        statusText.text = "Now copying trader $traderId"
                        load()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { statusText.text = "Follow failed: ${e.message}" }
                }
            }
        }

        view.findViewById<Button>(R.id.copyStopButton).setOnClickListener {
            val copierId = view.findViewById<EditText>(R.id.copyCopierIdInput).text.toString().trim()
            if (copierId.isEmpty()) {
                Toast.makeText(requireContext(), "Enter copier ID", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            statusText.text = "Stopping…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    UserWalletApiService.stopCopyTrader(copierId)
                    withContext(Dispatchers.Main) {
                        statusText.text = "Stopped copying"
                        load()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { statusText.text = "Stop failed: ${e.message}" }
                }
            }
        }
    }
}
