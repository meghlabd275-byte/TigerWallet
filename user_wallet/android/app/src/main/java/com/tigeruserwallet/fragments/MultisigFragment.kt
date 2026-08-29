package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.TextView
import android.widget.Toast
import androidx.fragment.app.Fragment
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONArray

/**
 * Multisig: create multisig wallets, list them, create/sign/execute multisig
 * transactions. All calls go through the wallet_api multisig proxy
 * (/wallet/multisig/* -> MasterWallet) — real backend state, fail-closed
 * error display, no fabricated data.
 */
class MultisigFragment : Fragment() {

    private lateinit var container: LinearLayout

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_multisig, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        container = view.findViewById(R.id.multisigList)
        buildUi()
        loadWallets()
    }

    private fun buildUi() {
        val ctx = requireContext()
        val nameInput = EditText(ctx).apply { hint = "Multisig name" }
        val ownersInput = EditText(ctx).apply { hint = "Owners (comma-separated 0x…)" }
        val thresholdInput = EditText(ctx).apply {
            hint = "Threshold (required signatures)"
            inputType = android.text.InputType.TYPE_CLASS_NUMBER
        }
        val createBtn = Button(ctx).apply { text = "Create Multisig" }
        val walletsHeader = TextView(ctx).apply { text = "Wallets"; textSize = 18f; setPadding(0, 24, 0, 8) }
        val walletsBox = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; tag = "wallets" }
        val txWalletInput = EditText(ctx).apply { hint = "Multisig wallet ID" }
        val txToInput = EditText(ctx).apply { hint = "To address (0x…)" }
        val txValueInput = EditText(ctx).apply { hint = "Value (wei)" }
        val txDataInput = EditText(ctx).apply { hint = "Data (hex, optional)" }
        val txCreateBtn = Button(ctx).apply { text = "Create Transaction" }
        val txRefreshBtn = Button(ctx).apply { text = "Load Transactions" }
        val txBox = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; tag = "txs" }

        container.addView(nameInput)
        container.addView(ownersInput)
        container.addView(thresholdInput)
        container.addView(createBtn)
        container.addView(walletsHeader)
        container.addView(walletsBox)
        container.addView(TextView(ctx).apply { text = "Transactions"; textSize = 18f; setPadding(0, 24, 0, 8) })
        container.addView(txWalletInput)
        container.addView(txToInput)
        container.addView(txValueInput)
        container.addView(txDataInput)
        container.addView(txCreateBtn)
        container.addView(txRefreshBtn)
        container.addView(txBox)

        createBtn.setOnClickListener {
            val name = nameInput.text.toString().trim()
            val owners = ownersInput.text.toString().split(',').map { it.trim() }.filter { it.isNotEmpty() }
            val threshold = thresholdInput.text.toString().toIntOrNull() ?: 0
            if (name.isEmpty() || owners.isEmpty() || threshold < 1) {
                Toast.makeText(ctx, "Enter name, owners and threshold", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    UserWalletApiService.createMultisigWallet(name, owners, threshold, 1)
                    withContext(Dispatchers.Main) {
                        Toast.makeText(ctx, "Multisig wallet created", Toast.LENGTH_SHORT).show()
                        loadWallets()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) {
                        Toast.makeText(ctx, "Create failed: ${e.message}", Toast.LENGTH_LONG).show()
                    }
                }
            }
        }

        txCreateBtn.setOnClickListener {
            val wid = txWalletInput.text.toString().trim()
            val to = txToInput.text.toString().trim()
            val value = txValueInput.text.toString().trim()
            val data = txDataInput.text.toString().trim()
            if (wid.isEmpty() || to.isEmpty() || value.isEmpty()) {
                Toast.makeText(ctx, "Enter wallet id, to address and value", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    UserWalletApiService.createMultisigTransaction(wid, to, value, data)
                    withContext(Dispatchers.Main) {
                        Toast.makeText(ctx, "Multisig transaction created — pending signatures", Toast.LENGTH_SHORT).show()
                        loadTransactions(wid, txBox)
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) {
                        Toast.makeText(ctx, "Create tx failed: ${e.message}", Toast.LENGTH_LONG).show()
                    }
                }
            }
        }

        txRefreshBtn.setOnClickListener {
            val wid = txWalletInput.text.toString().trim()
            if (wid.isEmpty()) {
                Toast.makeText(ctx, "Enter a multisig wallet id", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            loadTransactions(wid, txBox)
        }
    }

    private fun loadWallets() {
        val walletsBox = container.findViewWithTag<LinearLayout>("wallets") ?: return
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val json = UserWalletApiService.listMultisigWallets()
                val list = json.optJSONArray("multisig_wallets") ?: json.optJSONArray("wallets") ?: JSONArray()
                withContext(Dispatchers.Main) {
                    walletsBox.removeAllViews()
                    if (list.length() == 0) {
                        walletsBox.addView(TextView(requireContext()).apply { text = "No multisig wallets" })
                    }
                    for (i in 0 until list.length()) {
                        val w = list.getJSONObject(i)
                        walletsBox.addView(TextView(requireContext()).apply {
                            text = "${w.optString("name", w.optString("id"))} · ${w.optString("id")} · chain ${w.optInt("chain_id")} · ${w.optInt("threshold")}-of-${w.optJSONArray("owners")?.length() ?: 0}"
                            setPadding(0, 12, 0, 12)
                        })
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    walletsBox.removeAllViews()
                    walletsBox.addView(TextView(requireContext()).apply { text = "Multisig unavailable: ${e.message}" })
                }
            }
        }
    }

    private fun loadTransactions(walletId: String, txBox: LinearLayout) {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val json = UserWalletApiService.listMultisigTransactions(walletId)
                val list = json.optJSONArray("transactions") ?: json.optJSONArray("multisig_transactions") ?: JSONArray()
                withContext(Dispatchers.Main) {
                    txBox.removeAllViews()
                    if (list.length() == 0) {
                        txBox.addView(TextView(requireContext()).apply { text = "No multisig transactions" })
                    }
                    for (i in 0 until list.length()) {
                        val t = list.getJSONObject(i)
                        val tid = t.optString("id")
                        val row = LinearLayout(requireContext()).apply { orientation = LinearLayout.HORIZONTAL }
                        row.addView(TextView(requireContext()).apply {
                            text = "$tid → ${t.optString("to_address")} · ${t.optString("status")} · sigs ${t.optInt("signatures_collected")}"
                            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
                        })
                        row.addView(Button(requireContext()).apply {
                            text = "Sign"
                            setOnClickListener { multisigAction(tid, "sign", walletId, txBox) }
                        })
                        row.addView(Button(requireContext()).apply {
                            text = "Execute"
                            setOnClickListener { multisigAction(tid, "execute", walletId, txBox) }
                        })
                        txBox.addView(row)
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    txBox.removeAllViews()
                    txBox.addView(TextView(requireContext()).apply { text = "Failed: ${e.message}" })
                }
            }
        }
    }

    private fun multisigAction(txId: String, action: String, walletId: String, txBox: LinearLayout) {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val result = if (action == "sign") {
                    UserWalletApiService.signMultisigTransaction(txId)
                } else {
                    UserWalletApiService.executeMultisigTransaction(txId)
                }
                withContext(Dispatchers.Main) {
                    val msg = if (action == "execute") {
                        "Transaction submitted to the blockchain network: ${result.optString("tx_hash", result.optString("status", "broadcast"))}"
                    } else {
                        "Multisig transaction signed"
                    }
                    Toast.makeText(requireContext(), msg, Toast.LENGTH_LONG).show()
                    loadTransactions(walletId, txBox)
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), "$action failed: ${e.message}", Toast.LENGTH_LONG).show()
                }
            }
        }
    }
}
