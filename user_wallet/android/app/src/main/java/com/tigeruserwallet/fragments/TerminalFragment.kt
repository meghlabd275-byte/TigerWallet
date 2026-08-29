package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.TextView
import androidx.fragment.app.Fragment
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService
import com.tigeruserwallet.ui.CandleChartView
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject

/**
 * Trading Terminal: live 24h ticker (/terminal/ticker) + OHLC candles
 * (/terminal/kline) rendered in a real CandleChartView.
 */
class TerminalFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_terminal, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val symbolInput = view.findViewById<EditText>(R.id.terminalSymbolInput)
        val daysInput = view.findViewById<EditText>(R.id.terminalDaysInput)
        val tickerText = view.findViewById<TextView>(R.id.terminalTickerText)
        val chart = view.findViewById<CandleChartView>(R.id.terminalChart)

        view.findViewById<Button>(R.id.terminalLoadButton).setOnClickListener {
            val symbol = symbolInput.text.toString().trim().uppercase().ifEmpty { "ETH" }
            val days = daysInput.text.toString().trim().toIntOrNull() ?: 1
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val ticker = UserWalletApiService.getTerminalTicker(symbol)
                    withContext(Dispatchers.Main) { tickerText.text = ticker.toString() }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { tickerText.text = "Ticker unavailable" }
                }
                try {
                    val raw = UserWalletApiService.getTerminalKline(symbol, days)
                    val candles = parseCandles(raw)
                    withContext(Dispatchers.Main) { chart.setCandles(candles) }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { chart.setCandles(emptyList()) }
                }
            }
        }
    }

    private fun parseCandles(raw: JSONObject): List<CandleChartView.Candle> {
        val arr = raw.optJSONArray("candles") ?: raw.optJSONArray("kline")
            ?: raw.optJSONArray("data")
        return parseArrayCandles(arr)
    }

    private fun parseArrayCandles(arr: JSONArray?): List<CandleChartView.Candle> {
        if (arr == null) return emptyList()
        return (0 until arr.length()).mapNotNull { i ->
            val el = arr.opt(i)
            when (el) {
                is JSONArray -> {
                    if (el.length() < 5) null
                    else CandleChartView.Candle(
                        el.optDouble(1, 0.0).toFloat(), // open
                        el.optDouble(2, 0.0).toFloat(), // high
                        el.optDouble(3, 0.0).toFloat(), // low
                        el.optDouble(4, 0.0).toFloat()  // close
                    )
                }
                is JSONObject -> {
                    val o = el.optDouble("open", el.optDouble("o", 0.0))
                    val h = el.optDouble("high", el.optDouble("h", 0.0))
                    val l = el.optDouble("low", el.optDouble("l", 0.0))
                    val c = el.optDouble("close", el.optDouble("c", 0.0))
                    CandleChartView.Candle(o.toFloat(), h.toFloat(), l.toFloat(), c.toFloat())
                }
                else -> null
            }
        }
    }
}
