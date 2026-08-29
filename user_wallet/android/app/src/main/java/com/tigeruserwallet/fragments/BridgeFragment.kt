package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import android.widget.Button
import android.widget.EditText
import android.widget.ProgressBar
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
 * Cross-chain bridge — real bridge_service integration:
 * GET /bridge/routes (available routes), POST /bridge/quote (quote),
 * POST /bridge/transfer (initiate the cross-chain transfer). The user is told
 * the transaction was submitted to the blockchain network.
 */
class BridgeFragment : Fragment() {
    private lateinit var walletSpinner: Spinner
    private lateinit var fromChainSpinner: Spinner
    private lateinit var toChainSpinner: Spinner
    private lateinit var tokenInput: EditText
    private lateinit var amountInput: EditText
    private lateinit var recipientInput: EditText
    private lateinit var passwordInput: EditText
    private lateinit var quoteButton: Button
    private lateinit var executeButton: Button
    private lateinit var quoteTextView: TextView
    private lateinit var statusTextView: TextView
    private lateinit var progressBar: ProgressBar

    private val chains = arrayOf("Ethereum (1)", "BNB Chain (56)", "Polygon (137)")
    private val chainIds = intArrayOf(1, 56, 137)

    private var wallets: List<UserWalletApiService.Wallet> = emptyList()

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_bridge, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        walletSpinner = view.findViewById(R.id.bridgeWalletSpinner)
        fromChainSpinner = view.findViewById(R.id.bridgeFromChainSpinner)
        toChainSpinner = view.findViewById(R.id.bridgeToChainSpinner)
        tokenInput = view.findViewById(R.id.bridgeTokenInput)
        amountInput = view.findViewById(R.id.bridgeAmountInput)
        recipientInput = view.findViewById(R.id.bridgeRecipientInput)
        passwordInput = view.findViewById(R.id.bridgePasswordInput)
        quoteButton = view.findViewById(R.id.bridgeQuoteButton)
        executeButton = view.findViewById(R.id.bridgeExecuteButton)
        quoteTextView = view.findViewById(R.id.bridgeQuoteTextView)
        statusTextView = view.findViewById(R.id.bridgeStatusTextView)
        progressBar = view.findViewById(R.id.bridgeProgressBar)

        fromChainSpinner.adapter =
            ArrayAdapter(requireContext(), android.R.layout.simple_spinner_dropdown_item, chains)
        toChainSpinner.adapter =
            ArrayAdapter(requireContext(), android.R.layout.simple_spinner_dropdown_item, chains)

        quoteButton.setOnClickListener { loadQuote() }
        executeButton.setOnClickListener { executeBridge() }
        loadWallets()
    }

    private fun loadWallets() {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                wallets = UserWalletApiService.getWallets()
                withContext(Dispatchers.Main) {
                    walletSpinner.adapter = ArrayAdapter(
                        requireContext(),
                        android.R.layout.simple_spinner_dropdown_item,
                        wallets.map { "${it.label} \u00b7 ${it.address.take(10)}..." })
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), "Failed to load wallets", Toast.LENGTH_SHORT).show()
                }
            }
        }
    }

    private fun readInputs(): Triple<String, String, String>? {
        val token = tokenInput.text.toString().trim()
        val amount = amountInput.text.toString().trim()
        if (token.isEmpty() || amount.isEmpty()) {
            Toast.makeText(requireContext(), "Fill token and amount", Toast.LENGTH_SHORT).show()
            return null
        }
        return Triple(token, amount, "")
    }

    private fun loadQuote() {
        val (token, amount, _) = readInputs() ?: return
        val fromChain = chainIds[fromChainSpinner.selectedItemPosition]
        val toChain = chainIds[toChainSpinner.selectedItemPosition]
        setLoading(true)
        quoteTextView.text = "Fetching bridge quote..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val q = UserWalletApiService.getBridgeQuote(fromChain, toChain, token, amount)
                withContext(Dispatchers.Main) {
                    quoteTextView.text = q.toString(2)
                    setLoading(false)
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    quoteTextView.text = "\u2717 ${e.message ?: "Quote failed"}"
                    setLoading(false)
                }
            }
        }
    }

    private fun executeBridge() {
        val wallet = wallets.getOrNull(walletSpinner.selectedItemPosition) ?: run {
            Toast.makeText(requireContext(), "Select a wallet", Toast.LENGTH_SHORT).show()
            return
        }
        val (token, amount, _) = readInputs() ?: return
        val fromChain = chainIds[fromChainSpinner.selectedItemPosition]
        val toChain = chainIds[toChainSpinner.selectedItemPosition]
        executeButton.isEnabled = false
        statusTextView.text = "Submitting..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val res = UserWalletApiService.executeBridgeTransfer(
                    fromChain, toChain, token, amount, wallet.address
                )
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2713 Bridge transfer submitted to the blockchain network" +
                        "\n" + res.optString("id", res.optString("tx_hash", res.toString()))
                    Toast.makeText(
                        requireContext(),
                        "Transaction submitted to the blockchain network",
                        Toast.LENGTH_LONG
                    ).show()
                    executeButton.isEnabled = true
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Bridge failed"}"
                    executeButton.isEnabled = true
                }
            }
        }
    }

    private fun setLoading(loading: Boolean) {
        progressBar.visibility = if (loading) View.VISIBLE else View.GONE
    }
}
