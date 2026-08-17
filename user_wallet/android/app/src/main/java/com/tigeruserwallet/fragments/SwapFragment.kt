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
 * AMM swap screen: from/to/amount inputs fetch a real [UserWalletApiService.getAmmQuote]
 * (and a fallback [getSwapQuote]); the "Swap" button broadcasts the real on-chain
 * [ammSwap]. On success the user is told the transaction was submitted to the
 * blockchain network, exactly like SendFragment.
 */
class SwapFragment : Fragment() {
    private lateinit var walletSpinner: Spinner
    private lateinit var chainSpinner: Spinner
    private lateinit var fromTokenInput: EditText
    private lateinit var toTokenInput: EditText
    private lateinit var amountInput: EditText
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
        return inflater.inflate(R.layout.fragment_swap, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        walletSpinner = view.findViewById(R.id.swapWalletSpinner)
        chainSpinner = view.findViewById(R.id.swapChainSpinner)
        fromTokenInput = view.findViewById(R.id.swapFromTokenInput)
        toTokenInput = view.findViewById(R.id.swapToTokenInput)
        amountInput = view.findViewById(R.id.swapAmountInput)
        passwordInput = view.findViewById(R.id.swapPasswordInput)
        quoteButton = view.findViewById(R.id.swapQuoteButton)
        executeButton = view.findViewById(R.id.swapExecuteButton)
        quoteTextView = view.findViewById(R.id.swapQuoteTextView)
        statusTextView = view.findViewById(R.id.swapStatusTextView)
        progressBar = view.findViewById(R.id.swapProgressBar)

        chainSpinner.adapter =
            ArrayAdapter(requireContext(), android.R.layout.simple_spinner_dropdown_item, chains)

        quoteButton.setOnClickListener { loadQuote() }
        executeButton.setOnClickListener { executeSwap() }
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
                        wallets.map { "${it.label} · ${it.address.take(10)}..." })
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), "Failed to load wallets", Toast.LENGTH_SHORT).show()
                }
            }
        }
    }

    private fun readInputs(): Triple<String, String, String>? {
        val from = fromTokenInput.text.toString().trim()
        val to = toTokenInput.text.toString().trim()
        val amount = amountInput.text.toString().trim()
        if (from.isEmpty() || to.isEmpty() || amount.isEmpty()) {
            Toast.makeText(requireContext(), "Fill token and amount fields", Toast.LENGTH_SHORT).show()
            return null
        }
        return Triple(from, to, amount)
    }

    private fun loadQuote() {
        val (from, to, amount) = readInputs() ?: return
        val chainId = chainIds[chainSpinner.selectedItemPosition]
        setLoading(true)
        quoteTextView.text = "Fetching quote..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val quote = try {
                    UserWalletApiService.getAmmQuote(from, to, amount, chainId)
                } catch (e: Exception) {
                    UserWalletApiService.getSwapQuote(from, to, amount, chainId)
                }
                withContext(Dispatchers.Main) {
                    quoteTextView.text = formatQuote(quote)
                    setLoading(false)
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    quoteTextView.text = "✗ ${e.message ?: "Quote failed"}"
                    setLoading(false)
                }
            }
        }
    }

    private fun executeSwap() {
        val wallet = wallets.getOrNull(walletSpinner.selectedItemPosition) ?: run {
            Toast.makeText(requireContext(), "Select a wallet", Toast.LENGTH_SHORT).show()
            return
        }
        val (from, to, amount) = readInputs() ?: return
        val password = passwordInput.text.toString()
        val chainId = chainIds[chainSpinner.selectedItemPosition]
        if (password.isEmpty()) {
            Toast.makeText(requireContext(), "Enter wallet password", Toast.LENGTH_SHORT).show()
            return
        }

        executeButton.isEnabled = false
        statusTextView.text = "Submitting..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val r = UserWalletApiService.ammSwap(wallet.id, password, from, to, amount, chainId)
                val hash = r.optString("tx_hash")
                withContext(Dispatchers.Main) {
                    statusTextView.text = buildMessage(hash)
                    Toast.makeText(
                        requireContext(),
                        "Transaction submitted to the blockchain network",
                        Toast.LENGTH_LONG
                    ).show()
                    executeButton.isEnabled = true
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "✗ ${e.message ?: "Swap failed"}"
                    executeButton.isEnabled = true
                }
            }
        }
    }

    private fun formatQuote(q: UserWalletApiService.SwapQuote): String {
        val sb = StringBuilder()
        sb.append("${q.fromAmount} ${q.fromToken} -> ${q.toAmount} ${q.toToken}")
        if (q.priceImpact != 0.0) sb.append("\nPrice impact: ").append(q.priceImpact)
        if (q.route.isNotEmpty()) sb.append("\nRoute: ").append(q.route)
        return sb.toString()
    }

    private fun buildMessage(txHash: String): String {
        val sb = StringBuilder()
        sb.append("✓ Transaction submitted to the blockchain network")
        if (txHash.isNotEmpty()) sb.append("\nHash: ").append(txHash)
        return sb.toString()
    }

    private fun setLoading(loading: Boolean) {
        progressBar.visibility = if (loading) View.VISIBLE else View.GONE
    }
}
