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
 * Fiat on/off ramp. Lists live providers (GET /ramp/providers) and fetches real
 * on-ramp / off-ramp quotes (POST /ramp/quote, /ramp/offramp-quote). No
 * fabricated rates — every value comes from the backend.
 */
class RampFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_ramp, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val providersText = view.findViewById<TextView>(R.id.rampProvidersText)
        val quoteText = view.findViewById<TextView>(R.id.rampQuoteText)
        val providerId = view.findViewById<EditText>(R.id.rampProviderIdInput)
        val amount = view.findViewById<EditText>(R.id.rampAmountInput)
        val fiat = view.findViewById<EditText>(R.id.rampFiatInput)
        val crypto = view.findViewById<EditText>(R.id.rampCryptoInput)

        CoroutineScope(Dispatchers.IO).launch {
            try {
                val data = UserWalletApiService.getFiatProviders()
                val arr = data.optJSONArray("providers")
                withContext(Dispatchers.Main) {
                    if (arr == null || arr.length() == 0) {
                        providersText.text = "No fiat providers configured"
                    } else {
                        providersText.text = (0 until arr.length()).joinToString("\n") {
                            val p = arr.getJSONObject(it)
                            "• ${p.optString("id", p.optString("name", "?"))}: ${p.optString("name", "")}"
                        }
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) { providersText.text = "Providers unavailable" }
            }
        }

        fun quote(offramp: Boolean) {
            val pid = providerId.text.toString().trim()
            val amt = amount.text.toString().trim()
            if (pid.isEmpty() || amt.isEmpty()) {
                Toast.makeText(requireContext(), "Enter provider ID and amount", Toast.LENGTH_SHORT).show()
                return
            }
            val fiatC = fiat.text.toString().trim().uppercase().ifEmpty { "USD" }
            val cryptoC = crypto.text.toString().trim().uppercase().ifEmpty { "ETH" }
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val res = if (offramp) UserWalletApiService.getFiatOfframpQuote(pid, amt, fiatC, cryptoC)
                        else UserWalletApiService.getFiatQuote(pid, amt, fiatC, cryptoC, "card")
                    withContext(Dispatchers.Main) { quoteText.text = res.toString(2) }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { quoteText.text = "Quote failed: ${e.message}" }
                }
            }
        }

        view.findViewById<Button>(R.id.rampBuyButton).setOnClickListener { quote(false) }
        view.findViewById<Button>(R.id.rampSellButton).setOnClickListener { quote(true) }
    }
}
