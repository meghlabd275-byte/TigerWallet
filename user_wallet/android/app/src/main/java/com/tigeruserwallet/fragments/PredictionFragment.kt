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
 * Prediction markets: list markets (GET /prediction/markets) and place bets
 * (POST /prediction/markets/:id/bet).
 */
class PredictionFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_prediction, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val marketsText = view.findViewById<TextView>(R.id.predictionMarketsText)
        val statusText = view.findViewById<TextView>(R.id.predictionStatusText)

        fun load() {
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val markets = UserWalletApiService.getPredictionMarkets()
                    withContext(Dispatchers.Main) {
                        marketsText.text = if (markets.isEmpty()) "No active markets"
                        else markets.joinToString("\n") {
                            "\u2022 ${it.optString("id", "?")}: ${it.optString("question", it.optString("title", "?"))} (${it.optString("status", "?")})"
                        }
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { marketsText.text = "Markets unavailable" }
                }
            }
        }
        load()

        view.findViewById<Button>(R.id.predictionBetButton).setOnClickListener {
            val marketId = view.findViewById<EditText>(R.id.predictionMarketIdInput).text.toString().trim()
            val side = view.findViewById<EditText>(R.id.predictionSideInput).text.toString().trim()
            val amount = view.findViewById<EditText>(R.id.predictionAmountInput).text.toString().trim()
            if (marketId.isEmpty() || side.isEmpty() || amount.isEmpty()) {
                Toast.makeText(requireContext(), "Enter market ID, side and amount", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            statusText.text = "Placing bet…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    UserWalletApiService.placePredictionBet(marketId, side, amount)
                    withContext(Dispatchers.Main) {
                        statusText.text = "Bet submitted"
                        load()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { statusText.text = "Bet failed: ${e.message}" }
                }
            }
        }
    }
}
