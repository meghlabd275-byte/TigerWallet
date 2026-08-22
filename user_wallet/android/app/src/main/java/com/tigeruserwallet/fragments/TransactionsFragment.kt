package com.tigeruserwallet.fragments

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import android.widget.Spinner
import android.widget.TextView
import androidx.appcompat.widget.AppCompatSpinner
import androidx.fragment.app.Fragment
import androidx.recyclerview.widget.LinearLayoutManager
import com.google.android.material.button.MaterialButton
import com.google.android.material.progressindicator.CircularProgressIndicator
import com.google.android.material.textfield.TextInputEditText
import com.google.android.material.textfield.TextInputLayout
import com.tigeruserwallet.R
import com.tigeruserwallet.adapters.TransactionAdapter
import com.tigeruserwallet.api.UserWalletApiService
import com.tigeruserwallet.databinding.FragmentTransactionsBinding
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Transactions (mirrors web Transactions.tsx):
 *  - wallet selector + network/token filters
 *  - Send form (POST /wallets/:id/send) -> TxSubmittedBanner with the real
 *    transaction hash + explorer link (per chainId)
 *  - Sign message form (POST /wallets/:id/sign)
 *  - transactions table (GET /wallets/:id/transactions)
 *
 * No stubs: every value is a real backend fetch.
 */
class TransactionsFragment : Fragment() {

    private var _binding: FragmentTransactionsBinding? = null
    private val binding get() = _binding!!

    private lateinit var walletSpinner: AppCompatSpinner
    private lateinit var networkSpinner: AppCompatSpinner
    private lateinit var tokenSpinner: AppCompatSpinner
    private lateinit var applyButton: MaterialButton

    private lateinit var toInput: TextInputEditText
    private lateinit var amountInput: TextInputEditText
    private lateinit var sendPwInput: TextInputEditText
    private lateinit var sendPwLayout: TextInputLayout
    private lateinit var sendButton: MaterialButton
    private lateinit var sendProgress: CircularProgressIndicator

    private lateinit var msgInput: TextInputEditText
    private lateinit var signPwInput: TextInputEditText
    private lateinit var signPwLayout: TextInputLayout
    private lateinit var signButton: MaterialButton
    private lateinit var signProgress: CircularProgressIndicator

    private var wallets: List<UserWalletApiService.Wallet> = emptyList()
    private var walletId: String = ""
    private var bannerHandler: Handler? = null
    private var bannerRunnable: Runnable? = null

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentTransactionsBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        walletSpinner = binding.walletSpinner
        networkSpinner = binding.networkSpinner
        tokenSpinner = binding.tokenSpinner
        applyButton = binding.applyFiltersButton
        toInput = binding.toInput
        amountInput = binding.amountInput
        sendPwInput = binding.sendPwInput
        sendPwLayout = binding.sendPwLayout
        sendButton = binding.sendButton
        sendProgress = binding.sendProgress
        msgInput = binding.msgInput
        signPwInput = binding.signPwInput
        signPwLayout = binding.signPwLayout
        signButton = binding.signButton
        signProgress = binding.signProgress

        binding.transactionsRecyclerView.layoutManager = LinearLayoutManager(requireContext())
        binding.swipeRefresh.setOnRefreshListener { loadTransactions() }
        applyButton.setOnClickListener { loadTransactions() }
        sendButton.setOnClickListener { onSend() }
        signButton.setOnClickListener { onSign() }
        hideBanner()

        // Network filter spinner (chain names + "All networks").
        val networkNames = listOf(getString(R.string.tx_all_networks)) +
            UserWalletApiService.CHAINS.map { it.name }
        networkSpinner.adapter = ArrayAdapter(
            requireContext(),
            android.R.layout.simple_spinner_dropdown_item,
            networkNames
        )
        // Token filter spinner.
        val tokenNames = listOf(
            getString(R.string.tx_all_tokens), "ETH", "BNB", "MATIC"
        )
        tokenSpinner.adapter = ArrayAdapter(
            requireContext(),
            android.R.layout.simple_spinner_dropdown_item,
            tokenNames
        )

        walletSpinner.onItemSelectedListener = object : Spinner.OnItemSelectedListener {
            override fun onItemSelected(p: Spinner, v: View?, pos: Int, id: Long) {
                walletId = wallets.getOrNull(pos)?.id.orEmpty()
                loadTransactions()
            }
            override fun onNothingSelected(p: Spinner) {}
        }

        loadWallets()
    }

    private fun loadWallets() {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val ws = UserWalletApiService.getWallets()
                wallets = ws
                withContext(Dispatchers.Main) {
                    walletSpinner.adapter = ArrayAdapter(
                        requireContext(),
                        android.R.layout.simple_spinner_dropdown_item,
                        ws.map { "${it.label.ifEmpty { it.address.take(8) }} (#${it.chainId})" }
                    )
                    if (ws.isNotEmpty()) {
                        walletSpinner.setSelection(0)
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    walletSpinner.adapter = ArrayAdapter(
                        requireContext(),
                        android.R.layout.simple_spinner_dropdown_item,
                        listOf(getString(R.string.tx_select_wallet))
                    )
                }
            }
        }
    }

    private fun loadTransactions() {
        if (walletId.isEmpty()) {
            binding.transactionsRecyclerView.adapter = TransactionAdapter(emptyList())
            binding.emptyState.visibility = View.VISIBLE
            binding.transactionsRecyclerView.visibility = View.GONE
            return
        }
        binding.swipeRefresh.isRefreshing = true
        val network = networkSpinner.selectedItem?.toString().orEmpty()
        val token = tokenSpinner.selectedItem?.toString().orEmpty()
        val netParam = network.takeIf { it.isNotEmpty() && it != getString(R.string.tx_all_networks) }
        val tokParam = token.takeIf { it.isNotEmpty() && it != getString(R.string.tx_all_tokens) }
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val txs = UserWalletApiService.getTransactions(
                    walletId, network = netParam, token = tokParam
                )
                withContext(Dispatchers.Main) {
                    binding.transactionsRecyclerView.adapter = TransactionAdapter(txs)
                    binding.emptyState.visibility =
                        if (txs.isEmpty()) View.VISIBLE else View.GONE
                    binding.transactionsRecyclerView.visibility =
                        if (txs.isEmpty()) View.GONE else View.VISIBLE
                    binding.swipeRefresh.isRefreshing = false
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    binding.transactionsRecyclerView.adapter = TransactionAdapter(emptyList())
                    binding.emptyState.visibility = View.VISIBLE
                    binding.transactionsRecyclerView.visibility = View.GONE
                    binding.swipeRefresh.isRefreshing = false
                }
            }
        }
    }

    private fun onSend() {
        sendPwLayout.error = null
        binding.sendErrorText.visibility = View.GONE
        if (walletId.isEmpty()) {
            showSendError(getString(R.string.tx_select_first))
            return
        }
        val to = toInput.text?.toString().orEmpty().trim()
        val amount = amountInput.text?.toString().orEmpty().trim()
        val pw = sendPwInput.text?.toString().orEmpty()
        if (to.isEmpty()) { showSendError(getString(R.string.tx_enter_recipient)); return }
        if (amount.isEmpty()) { showSendError(getString(R.string.tx_enter_amount)); return }
        if (pw.length < 8) {
            showSendError(getString(R.string.err_password_short))
            return
        }
        setSendBusy(true)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val res = UserWalletApiService.sendTransaction(walletId, pw, to, amount)
                val chainId = wallets.find { it.id == walletId }?.chainId ?: 1
                withContext(Dispatchers.Main) {
                    setSendBusy(false)
                    toInput.setText("")
                    amountInput.setText("")
                    sendPwInput.setText("")
                    showBanner(res.transactionHash, chainId)
                    loadTransactions()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    setSendBusy(false)
                    showSendError(e.message ?: getString(R.string.tx_send_failed))
                }
            }
        }
    }

    private fun onSign() {
        signPwLayout.error = null
        binding.signErrorText.visibility = View.GONE
        binding.signResultText.visibility = View.GONE
        if (walletId.isEmpty()) { showSignError(getString(R.string.tx_select_first)); return }
        val msg = msgInput.text?.toString().orEmpty().trim()
        val pw = signPwInput.text?.toString().orEmpty()
        if (msg.isEmpty()) { showSignError(getString(R.string.tx_enter_message)); return }
        if (pw.length < 8) { showSignError(getString(R.string.err_password_short)); return }
        setSignBusy(true)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val res = UserWalletApiService.signMessage(walletId, pw, msg)
                withContext(Dispatchers.Main) {
                    setSignBusy(false)
                    binding.signResultText.text = getString(R.string.tx_signature_label, res.signature)
                    binding.signResultText.visibility = View.VISIBLE
                    msgInput.setText("")
                    signPwInput.setText("")
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    setSignBusy(false)
                    showSignError(e.message ?: getString(R.string.tx_sign_failed))
                }
            }
        }
    }

    // ---- TxSubmittedBanner (mirrors web TxSubmittedBanner) ----

    private fun showBanner(txHash: String, chainId: Int) {
        val explorer = UserWalletApiService.explorerFor(chainId)
        val root = binding.txBannerRoot
        val hashTv = root.findViewById<TextView>(R.id.txBannerHash)
        val explorerRow = root.findViewById<View>(R.id.txBannerExplorerRow)
        val explorerBtn = root.findViewById<TextView>(R.id.txBannerExplorer)
        val copyBtn = root.findViewById<View>(R.id.txBannerCopy)
        val dismissBtn = root.findViewById<View>(R.id.txBannerDismiss)
        hashTv.text = if (explorer.isNotEmpty()) {
            "${txHash.take(10)}\u2026${txHash.takeLast(8)} \u2197"
        } else {
            txHash.take(16) + "\u2026"
        }
        explorerRow.visibility = if (explorer.isNotEmpty()) View.VISIBLE else View.GONE
        explorerBtn.setOnClickListener {
            if (explorer.isNotEmpty()) {
                startActivity(
                    Intent(Intent.ACTION_VIEW, Uri.parse(explorer + txHash))
                        .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                )
            }
        }
        copyBtn.setOnClickListener {
            val cm = requireContext()
                .getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
            cm.setPrimaryClip(ClipData.newPlainText("tx", txHash))
        }
        dismissBtn.setOnClickListener { hideBanner() }
        root.visibility = View.VISIBLE
        // Auto-dismiss after 30s (mirror web).
        bannerHandler?.removeCallbacks(bannerRunnable ?: Runnable {})
        val r = Runnable { hideBanner() }
        bannerRunnable = r
        bannerHandler = Handler(Looper.getMainLooper())
        bannerHandler?.postDelayed(r, 30_000L)
    }

    private fun hideBanner() {
        binding.txBannerRoot.visibility = View.GONE
        bannerHandler?.removeCallbacks(bannerRunnable ?: Runnable {})
    }

    private fun showSendError(msg: String) {
        binding.sendErrorText.text = msg
        binding.sendErrorText.visibility = View.VISIBLE
    }

    private fun showSignError(msg: String) {
        binding.signErrorText.text = msg
        binding.signErrorText.visibility = View.VISIBLE
    }

    private fun setSendBusy(b: Boolean) {
        sendProgress.visibility = if (b) View.VISIBLE else View.GONE
        sendButton.isEnabled = !b
    }

    private fun setSignBusy(b: Boolean) {
        signProgress.visibility = if (b) View.VISIBLE else View.GONE
        signButton.isEnabled = !b
    }

    override fun onDestroyView() {
        super.onDestroyView()
        bannerHandler?.removeCallbacks(bannerRunnable ?: Runnable {})
        _binding = null
    }
}
