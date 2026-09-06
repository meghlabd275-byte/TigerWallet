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
 * Options engine: real /options/* backend (series list, live quote,
 * open buy/sell position, close position). No fabricated premiums or hashes.
 */
class OptionsFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_options, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val seriesText = view.findViewById<TextView>(R.id.optionsSeriesText)
        val quoteText = view.findViewById<TextView>(R.id.optionsQuoteText)
        val positionsText = view.findViewById<TextView>(R.id.optionsPositionsText)
        val statusText = view.findViewById<TextView>(R.id.optionsStatusText)

        fun renderSeries(list: List<org.json.JSONObject>): String =
            if (list.isEmpty()) "No active options series. An operator must add series first."
            else list.joinToString("\n") {
                "\u2022 ${it.optString("id", "?")}: ${it.optString("underlying", "?")}-${it.optString("strike", "?")} ${it.optString("style", "?")} exp ${it.optLong("expiry_unix", 0)}"
            }

        fun renderPositions(list: List<org.json.JSONObject>): String =
            if (list.isEmpty()) "No open options positions"
            else list.joinToString("\n") {
                "\u2022 ${it.optString("id", "?")}: ${it.optString("underlying", "?")}-${it.optString("strike", "?")} ${it.optString("side", "?")} x${it.optString("contracts", "?")} ${it.optString("status", "?")} pnl:${it.optString("pnl", "?")}"
            }

        fun load() {
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val series = UserWalletApiService.getOptionsSeries()
                    val positions = UserWalletApiService.getOptionsPositions()
                    withContext(Dispatchers.Main) {
                        seriesText.text = "Series:\n" + renderSeries(series)
                        positionsText.text = "Positions:\n" + renderPositions(positions)
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { seriesText.text = "Options unavailable" }
                }
            }
        }
        load()

        view.findViewById<Button>(R.id.optionsOpenButton).setOnClickListener {
            val seriesId = view.findViewById<EditText>(R.id.optionsSeriesIdInput).text.toString().trim()
            val side = view.findViewById<EditText>(R.id.optionsSideInput).text.toString().trim().lowercase()
            val contracts = view.findViewById<EditText>(R.id.optionsContractsInput).text.toString().trim()
            if (seriesId.isEmpty() || side.isEmpty() || contracts.isEmpty()) {
                Toast.makeText(requireContext(), "Enter series id, side and contracts", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            statusText.text = "Opening options position…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val res = UserWalletApiService.openOptionsPosition(seriesId, side, contracts)
                    withContext(Dispatchers.Main) {
                        statusText.text = if (res.has("tx_hash")) "Transaction submitted to the blockchain network: ${res.getString("tx_hash")}"
                                          else "Options position opened: ${res.optString("id", "ok")}"
                        load()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { statusText.text = "Open failed: ${e.message}" }
                }
            }
        }
        view.findViewById<Button>(R.id.optionsCloseButton).setOnClickListener {
            val id = view.findViewById<EditText>(R.id.optionsCloseIdInput).text.toString().trim()
            if (id.isEmpty()) {
                Toast.makeText(requireContext(), "Enter a position ID", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            statusText.text = "Closing options position…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    UserWalletApiService.closeOptionsPosition(id)
                    withContext(Dispatchers.Main) {
                        statusText.text = "Options position close submitted"
                        load()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { statusText.text = "Close failed: ${e.message}" }
                }
            }
        }
    }
}