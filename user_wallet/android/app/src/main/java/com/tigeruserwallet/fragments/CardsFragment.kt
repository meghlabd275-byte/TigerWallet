package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import androidx.fragment.app.Fragment
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Crypto card: live per-user card balance (GET /card/balance) and card
 * transactions (GET /card/transactions). All values are real backend fetches.
 */
class CardsFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_cards, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val balanceText = view.findViewById<TextView>(R.id.cardBalanceText)
        val txText = view.findViewById<TextView>(R.id.cardTransactionsText)

        CoroutineScope(Dispatchers.IO).launch {
            try {
                val balance = UserWalletApiService.getCryptoCardBalance()
                withContext(Dispatchers.Main) { balanceText.text = balance.toString(2) }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) { balanceText.text = "Card balance unavailable" }
            }
            try {
                val txs = UserWalletApiService.getCardTransactions()
                withContext(Dispatchers.Main) {
                    txText.text = if (txs.isEmpty()) "No card transactions"
                    else txs.joinToString("\n") { it.toString() }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) { txText.text = "Card transactions unavailable" }
            }
        }
    }
}
