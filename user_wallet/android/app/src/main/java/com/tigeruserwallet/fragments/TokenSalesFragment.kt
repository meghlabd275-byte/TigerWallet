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
 * Token sales: list active sales (GET /token-sales) and participate
 * (POST /token-sales/:id/participate).
 */
class TokenSalesFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_token_sales, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val listText = view.findViewById<TextView>(R.id.tokenSalesListText)
        val statusText = view.findViewById<TextView>(R.id.tokenSalesStatusText)

        fun load() {
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val sales = UserWalletApiService.getTokenSales()
                    withContext(Dispatchers.Main) {
                        listText.text = if (sales.isEmpty()) "No active token sales"
                        else sales.joinToString("\n") {
                            "\u2022 ${it.optString("id", "?")}: ${it.optString("name", it.optString("token", "?"))} @ ${it.optString("price", "?")} (${it.optString("status", "?")})"
                        }
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { listText.text = "Token sales unavailable" }
                }
            }
        }
        load()

        view.findViewById<Button>(R.id.tokenSaleParticipateButton).setOnClickListener {
            val saleId = view.findViewById<EditText>(R.id.tokenSaleIdInput).text.toString().trim()
            val amount = view.findViewById<EditText>(R.id.tokenSaleAmountInput).text.toString().trim()
            if (saleId.isEmpty() || amount.isEmpty()) {
                Toast.makeText(requireContext(), "Enter sale ID and amount", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            statusText.text = "Submitting…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val res = UserWalletApiService.participateTokenSale(saleId, amount)
                    withContext(Dispatchers.Main) {
                        val tx = res.optString("tx_hash", "")
                        statusText.text = if (tx.isNotEmpty()) "Transaction submitted to the blockchain network: $tx"
                                          else "Participation submitted"
                        load()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { statusText.text = "Failed: ${e.message}" }
                }
            }
        }
    }
}
