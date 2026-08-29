package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import android.widget.Button
import android.widget.EditText
import android.widget.Spinner
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
 * Price alerts: list / create / delete via /price-alerts (backend watch_alerts
 * engine evaluates them against live prices).
 */
class PriceAlertsFragment : Fragment() {

    private val directions = arrayOf("above", "below")

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_price_alerts, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val symbol = view.findViewById<EditText>(R.id.alertSymbolInput)
        val target = view.findViewById<EditText>(R.id.alertTargetInput)
        val directionSpinner = view.findViewById<Spinner>(R.id.alertDirectionSpinner)
        val listText = view.findViewById<TextView>(R.id.alertsListText)
        val deleteId = view.findViewById<EditText>(R.id.alertDeleteIdInput)

        directionSpinner.adapter =
            ArrayAdapter(requireContext(), android.R.layout.simple_spinner_dropdown_item, directions)

        fun loadAlerts() {
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val alerts = UserWalletApiService.getPriceAlerts()
                    withContext(Dispatchers.Main) {
                        listText.text = if (alerts.isEmpty()) "No price alerts"
                        else alerts.joinToString("\n") {
                            "• ${it.optString("id", "?")}: ${it.optString("symbol", "?")} " +
                                "${it.optString("direction", "above")} $${it.optString("target_price", "?")}"
                        }
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { listText.text = "Alerts unavailable" }
                }
            }
        }

        view.findViewById<Button>(R.id.alertCreateButton).setOnClickListener {
            val sym = symbol.text.toString().trim().uppercase()
            val tgt = target.text.toString().trim()
            if (sym.isEmpty() || tgt.isEmpty()) {
                Toast.makeText(requireContext(), "Enter symbol and target", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            val dir = directions[directionSpinner.selectedItemPosition]
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    UserWalletApiService.createPriceAlert(sym, tgt, dir)
                    withContext(Dispatchers.Main) {
                        Toast.makeText(requireContext(), "✓ Alert created", Toast.LENGTH_SHORT).show()
                        loadAlerts()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) {
                        Toast.makeText(requireContext(), "Create failed: ${e.message}", Toast.LENGTH_SHORT).show()
                    }
                }
            }
        }

        view.findViewById<Button>(R.id.alertDeleteButton).setOnClickListener {
            val id = deleteId.text.toString().trim()
            if (id.isEmpty()) {
                Toast.makeText(requireContext(), "Enter alert ID", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    UserWalletApiService.deletePriceAlert(id)
                    withContext(Dispatchers.Main) {
                        Toast.makeText(requireContext(), "✓ Alert deleted", Toast.LENGTH_SHORT).show()
                        loadAlerts()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) {
                        Toast.makeText(requireContext(), "Delete failed: ${e.message}", Toast.LENGTH_SHORT).show()
                    }
                }
            }
        }

        loadAlerts()
    }
}
