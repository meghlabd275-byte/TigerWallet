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
 * Perpetual + margin trading: positions list, open and close.
 * GET/POST /perpetual/positions, POST /perpetual/positions/:id/close,
 * GET/POST /margin/positions, POST /margin/positions/:id/close.
 */
class TradingFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_trading, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val perpText = view.findViewById<TextView>(R.id.tradingPerpText)
        val marginText = view.findViewById<TextView>(R.id.tradingMarginText)
        val statusText = view.findViewById<TextView>(R.id.tradingStatusText)

        fun render(list: List<org.json.JSONObject>): String =
            if (list.isEmpty()) "No open positions"
            else list.joinToString("\n") {
                "\u2022 ${it.optString("id", "?")}: ${it.optString("pair", "?")} ${it.optString("side", "?")} size:${it.optString("size", "?")} ${it.optString("leverage", "?")}x pnl:${it.optString("pnl", it.optString("unrealized_pnl", "?"))}"
            }

        fun load() {
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val perps = UserWalletApiService.getPerpetualPositions()
                    val margins = UserWalletApiService.getMarginPositions()
                    withContext(Dispatchers.Main) {
                        perpText.text = "Perpetual:\n" + render(perps)
                        marginText.text = "Margin:\n" + render(margins)
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { perpText.text = "Positions unavailable" }
                }
            }
        }
        load()

        fun open(perp: Boolean) {
            val pair = view.findViewById<EditText>(R.id.tradingPairInput).text.toString().trim()
            val side = view.findViewById<EditText>(R.id.tradingSideInput).text.toString().trim()
            val size = view.findViewById<EditText>(R.id.tradingSizeInput).text.toString().trim()
            val leverage = view.findViewById<EditText>(R.id.tradingLeverageInput).text.toString().trim().toIntOrNull() ?: 1
            if (pair.isEmpty() || side.isEmpty() || size.isEmpty()) {
                Toast.makeText(requireContext(), "Enter pair, side and size", Toast.LENGTH_SHORT).show()
                return
            }
            statusText.text = "Opening position…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val res = if (perp) UserWalletApiService.createPerpetualPosition(pair, side, size, leverage, 1)
                              else UserWalletApiService.createMarginPosition(pair, side, size, leverage, 1)
                    withContext(Dispatchers.Main) {
                        val tx = res.optString("tx_hash", "")
                        statusText.text = if (tx.isNotEmpty()) "Transaction submitted to the blockchain network: $tx"
                                          else "Position opened: ${res.optString("id", "ok")}"
                        load()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { statusText.text = "Open failed: ${e.message}" }
                }
            }
        }
        view.findViewById<Button>(R.id.tradingOpenPerpButton).setOnClickListener { open(true) }
        view.findViewById<Button>(R.id.tradingOpenMarginButton).setOnClickListener { open(false) }

        fun close(perp: Boolean) {
            val id = view.findViewById<EditText>(R.id.tradingPositionIdInput).text.toString().trim()
            if (id.isEmpty()) {
                Toast.makeText(requireContext(), "Enter position ID", Toast.LENGTH_SHORT).show()
                return
            }
            statusText.text = "Closing position…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    if (perp) UserWalletApiService.closePerpetualPosition(id)
                    else UserWalletApiService.closeMarginPosition(id)
                    withContext(Dispatchers.Main) {
                        statusText.text = "Position close submitted"
                        load()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { statusText.text = "Close failed: ${e.message}" }
                }
            }
        }
        view.findViewById<Button>(R.id.tradingClosePerpButton).setOnClickListener { close(true) }
        view.findViewById<Button>(R.id.tradingCloseMarginButton).setOnClickListener { close(false) }
    }
}
