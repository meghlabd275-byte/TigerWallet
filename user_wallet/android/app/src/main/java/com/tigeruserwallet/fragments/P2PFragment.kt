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
 * P2P trading: browse adverts (GET /p2p/adverts) and create orders
 * (POST /p2p/orders — KYC-gated backend-side; 403 surfaces to the user).
 */
class P2PFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_p2p, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val advertsText = view.findViewById<TextView>(R.id.p2pAdvertsText)
        val advertId = view.findViewById<EditText>(R.id.p2pAdvertIdInput)
        val amount = view.findViewById<EditText>(R.id.p2pAmountInput)
        val statusText = view.findViewById<TextView>(R.id.p2pStatusText)

        CoroutineScope(Dispatchers.IO).launch {
            try {
                val adverts = UserWalletApiService.getP2PAdverts()
                withContext(Dispatchers.Main) {
                    advertsText.text = if (adverts.isEmpty()) "No P2P adverts"
                    else adverts.joinToString("\n") {
                        "• ${it.optString("id", it.optString("advert_id", "?"))}: " +
                            "${it.optString("asset", it.optString("token", "?"))} @ ${it.optString("price", "?")}"
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) { advertsText.text = "Adverts unavailable" }
            }
        }

        view.findViewById<Button>(R.id.p2pCreateOrderButton).setOnClickListener {
            val id = advertId.text.toString().trim()
            val amt = amount.text.toString().trim()
            if (id.isEmpty() || amt.isEmpty()) {
                Toast.makeText(requireContext(), "Enter advert ID and amount", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            statusText.text = "Submitting…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val res = UserWalletApiService.createP2POrder(id, amt)
                    withContext(Dispatchers.Main) {
                        statusText.text = "✓ Order submitted: ${res}"
                        Toast.makeText(requireContext(), "✓ Order submitted", Toast.LENGTH_SHORT).show()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { statusText.text = "Order failed: ${e.message}" }
                }
            }
        }
    }
}
