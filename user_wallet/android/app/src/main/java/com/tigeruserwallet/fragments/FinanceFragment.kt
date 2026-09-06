package com.tigeruserwallet.fragments

import android.os.Bundle
import android.content.ClipboardManager
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import android.widget.Button
import android.widget.EditText
import android.widget.ImageView
import android.widget.LinearLayout
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
 * Wallet & finance plane: multi-chain ledger accounts, deterministic deposit
 * addresses with QR + copy, signed withdrawals, instant convert, KYC-gated
 * internal transfers, escrowed P2P marketplace, full ledger history.
 * Every value comes from the canonical backend.
 */
class FinanceFragment : Fragment() {

    private val assets = listOf("BTC", "ETH", "USDT", "USDC", "BNB", "SOL", "TRX", "MATIC", "LTC", "DOGE")

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_finance, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val accountsText = view.findViewById<TextView>(R.id.finAccountsText)
        val depositsList = view.findViewById<LinearLayout>(R.id.finDepositsList)
        val ratesText = view.findViewById<TextView>(R.id.finRatesText)
        val escrowList = view.findViewById<LinearLayout>(R.id.finEscrowList)
        val historyText = view.findViewById<TextView>(R.id.finHistoryText)
        val statusText = view.findViewById<TextView>(R.id.finStatusText)
        val currencySpinner = view.findViewById<Spinner>(R.id.finCurrencySpinner)
        val toInput = view.findViewById<EditText>(R.id.finToInput)
        val amountInput = view.findViewById<EditText>(R.id.finAmountInput)
        val addressInput = view.findViewById<EditText>(R.id.finWAddressInput)

        currencySpinner.adapter = ArrayAdapter(requireContext(), android.R.layout.simple_spinner_item, assets)

        fun currency(): String = currencySpinner.selectedItem?.toString() ?: assets[0]

        fun toast(msg: String) = Toast.makeText(requireContext(), msg, Toast.LENGTH_SHORT).show()

        view.findViewById<Button>(R.id.finTransferBtn).setOnClickListener {
            val to = toInput.text.toString().trim()
            val amount = amountInput.text.toString().trim()
            if (to.isEmpty() || amount.isEmpty()) { toast("Enter recipient and amount"); return@setOnClickListener }
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    UserWalletApiService.financeTransfer(to, currency(), amount)
                    withContext(Dispatchers.Main) { toast("Transfer completed"); toInput.text.clear(); amountInput.text.clear(); reload(accountsText, historyText) }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { toast("Transfer failed: ${e.message}") }
                }
            }
        }

        view.findViewById<Button>(R.id.finWithdrawBtn).setOnClickListener {
            val addr = addressInput.text.toString().trim()
            val amount = amountInput.text.toString().trim()
            if (addr.isEmpty() || amount.isEmpty()) { toast("Enter address and amount"); return@setOnClickListener }
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val res = UserWalletApiService.createWithdrawal(currency(), amount, addr)
                    val msg = if (res.optString("status") == "auto_approved") "Auto-approved" else "Queued for superadmin sign-off"
                    withContext(Dispatchers.Main) { toast(msg); addressInput.text.clear(); amountInput.text.clear(); reload(accountsText, historyText) }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { toast("Withdrawal failed: ${e.message}") }
                }
            }
        }

        view.findViewById<Button>(R.id.finConvertBtn).setOnClickListener {
            val amount = amountInput.text.toString().trim()
            if (amount.isEmpty()) { toast("Enter an amount"); return@setOnClickListener }
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val res = UserWalletApiService.financeConvert(currency(), "USDC", amount)
                    withContext(Dispatchers.Main) {
                        toast("Converted ${res.optString("from_amount")} ${res.optString("from_currency")} → ${res.optString("to_amount")} USDC")
                        amountInput.text.clear()
                        reload(accountsText, historyText)
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { toast("Convert failed: ${e.message}") }
                }
            }
        }

        fun reload(accountsTv: TextView, historyTv: TextView) {
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val accounts = UserWalletApiService.getFinanceAccounts()
                    val history = UserWalletApiService.getFinanceHistory()
                    val convertHistory = UserWalletApiService.getConvertHistory()
                    withContext(Dispatchers.Main) {
                        val arr = accounts.optJSONArray("accounts")
                        accountsTv.text = if (arr == null || arr.length() == 0) "No accounts yet"
                        else (0 until arr.length()).joinToString("\n") { i ->
                            val a = arr.getJSONObject(i)
                            "${a.optString("currency")}: ${a.optString("balance")} (available ${a.optString("available")})"
                        }
                        val h = history.optJSONArray("history")
                        historyTv.text = if (h == null || h.length() == 0) "No ledger history yet"
                        else (0 until minOf(h.length(), 30)).joinToString("\n") { i ->
                            val e = h.getJSONObject(i)
                            "${e.optString("kind")} ${if (e.optString("direction") == "debit") "−" else "+"}${e.optString("amount")} ${e.optString("currency")}"
                        }
                        val ch = convertHistory.optJSONArray("conversions")
                        val convTv = view.findViewById<TextView>(R.id.finConvertHistoryText)
                        convTv.text = if (ch == null || ch.length() == 0) "No conversions yet"
                        else (0 until minOf(ch.length(), 20)).joinToString("\n") { i ->
                            val c = ch.getJSONObject(i)
                            "${c.optString("from_currency")} ${c.optString("from_amount")} → ${c.optString("to_currency")} ${c.optString("to_amount")} @ ${c.optString("rate")}"
                        }
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { accountsTv.text = "Accounts unavailable: ${e.message}" }
                }
            }
        }

        reload(accountsText, historyText)

        // Deposit addresses with QR bitmaps (server-rendered, fetched with auth).
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val res = UserWalletApiService.getDepositAddresses()
                val arr = res.optJSONArray("addresses")
                withContext(Dispatchers.Main) {
                    depositsList.removeAllViews()
                    if (arr == null || arr.length() == 0) {
                        val tv = TextView(requireContext())
                        tv.text = "Deposit addresses unavailable on this node"
                        depositsList.addView(tv)
                    } else {
                        (0 until arr.length()).forEach { i ->
                            val d = arr.getJSONObject(i)
                            val tv = TextView(requireContext())
                            tv.text = "${d.optString("asset")}: ${d.optString("address")}"
                            tv.textSize = 11f
                            tv.setOnClickListener {
                                val cm = requireContext().getSystemService(android.content.Context.CLIPBOARD_SERVICE) as ClipboardManager
                                cm.text = d.optString("address")
                                toast("Address copied")
                            }
                            depositsList.addView(tv)
                            // QR bitmap (server-rendered).
                            CoroutineScope(Dispatchers.IO).launch {
                                try {
                                    val bytes = UserWalletApiService.getDepositQr(d.optString("asset"))
                                    if (bytes.isNotEmpty()) {
                                        val bmp = android.graphics.BitmapFactory.decodeByteArray(bytes, 0, bytes.size)
                                        if (bmp != null) withContext(Dispatchers.Main) {
                                            val iv = ImageView(requireContext())
                                            iv.setImageBitmap(bmp)
                                            depositsList.addView(iv)
                                        }
                                    }
                                } catch (_: Exception) { }
                            }
                        }
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) { statusText.text = "Deposits unavailable: ${e.message}" }
            }
        }

        // Convert rates.
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val res = UserWalletApiService.getConvertRates()
                val arr = res.optJSONArray("rates")
                withContext(Dispatchers.Main) {
                    ratesText.text = if (arr == null || arr.length() == 0) "No rates configured"
                    else (0 until arr.length()).joinToString("\n") { i ->
                        val r = arr.getJSONObject(i)
                        "${r.optString("from_currency")}/${r.optString("to_currency")}: ${r.optString("rate")}"
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) { ratesText.text = "Rates unavailable" }
            }
        }

        // Escrow marketplace: list + action buttons per order.
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val res = UserWalletApiService.getEscrowOrders()
                val arr = res.optJSONArray("orders")
                withContext(Dispatchers.Main) {
                    escrowList.removeAllViews()
                    if (arr == null || arr.length() == 0) {
                        val tv = TextView(requireContext())
                        tv.text = "No open orders"
                        escrowList.addView(tv)
                    } else {
                        (0 until arr.length()).forEach { i ->
                            val o = arr.getJSONObject(i)
                            val status = o.optString("status")
                            val tv = TextView(requireContext())
                            tv.text = "${o.optString("amount")} ${o.optString("currency")} @ ${o.optString("fiat_amount")} ${o.optString("fiat_currency")} (${o.optString("payment_method_name", o.optString("payment_method_code"))}, $status)"
                            tv.textSize = 11f
                            escrowList.addView(tv)
                            val actions = when (status) {
                                "open" -> listOf("accept" to "Buy", "cancel" to "Cancel")
                                "escrowed" -> listOf("paid" to "Mark paid", "dispute" to "Dispute")
                                "paid" -> listOf("release" to "Release", "dispute" to "Dispute")
                                else -> emptyList()
                            }
                            actions.forEach { (act, label) ->
                                val btn = Button(requireContext())
                                btn.text = label
                                btn.textSize = 11f
                                btn.setOnClickListener {
                                    CoroutineScope(Dispatchers.IO).launch {
                                        try {
                                            UserWalletApiService.escrowAction(o.getString("id"), act, if (act == "dispute") "disputed" else null)
                                            withContext(Dispatchers.Main) { toast("Escrow $act done") }
                                        } catch (e: Exception) {
                                            withContext(Dispatchers.Main) { toast("Escrow failed: ${e.message}") }
                                        }
                                    }
                                }
                                escrowList.addView(btn)
                            }
                        }
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) { statusText.text = "Escrow unavailable: ${e.message}" }
            }
        }
    }
}
