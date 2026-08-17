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

class SendFragment : Fragment() {
    private lateinit var walletSpinner: Spinner
    private lateinit var chainSpinner: Spinner
    private lateinit var recipientInput: EditText
    private lateinit var amountInput: EditText
    private lateinit var passwordInput: EditText
    private lateinit var sendButton: Button
    private lateinit var autoSendButton: Button
    private lateinit var statusTextView: TextView

    private val chains = arrayOf("Ethereum (1)", "BNB Chain (56)", "Polygon (137)")
    private val chainIds = intArrayOf(1, 56, 137)

    private var wallets: List<UserWalletApiService.Wallet> = emptyList()

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_send, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        walletSpinner = view.findViewById(R.id.walletSpinner)
        chainSpinner = view.findViewById(R.id.chainSpinner)
        recipientInput = view.findViewById(R.id.recipientInput)
        amountInput = view.findViewById(R.id.amountInput)
        passwordInput = view.findViewById(R.id.passwordInput)
        sendButton = view.findViewById(R.id.sendButton)
        autoSendButton = view.findViewById(R.id.autoSendButton)
        statusTextView = view.findViewById(R.id.statusTextView)

        chainSpinner.adapter =
            ArrayAdapter(requireContext(), android.R.layout.simple_spinner_dropdown_item, chains)

        sendButton.setOnClickListener { performSend(auto = false) }
        autoSendButton.setOnClickListener { performSend(auto = true) }

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

    private fun performSend(auto: Boolean) {
        val wallet = wallets.getOrNull(walletSpinner.selectedItemPosition) ?: run {
            Toast.makeText(requireContext(), "Select a wallet", Toast.LENGTH_SHORT).show()
            return
        }
        val to = recipientInput.text.toString().trim()
        val value = amountInput.text.toString().trim()
        val password = passwordInput.text.toString()
        val chainId = chainIds[chainSpinner.selectedItemPosition]

        if (to.isEmpty() || value.isEmpty() || password.isEmpty()) {
            Toast.makeText(requireContext(), "Fill all fields", Toast.LENGTH_SHORT).show()
            return
        }
        if (password.length < 8) {
            Toast.makeText(requireContext(), "Password must be at least 8 chars", Toast.LENGTH_SHORT).show()
            return
        }

        sendButton.isEnabled = false
        autoSendButton.isEnabled = false
        statusTextView.text = "Submitting..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val message = if (auto) {
                    val r = UserWalletApiService.autoSendTransaction(wallet.id, password, to, value, chainId)
                    buildMessage(r.txHash, r.autoApproved, r.autoApprovalReason)
                } else {
                    val r = UserWalletApiService.sendTransaction(wallet.id, password, to, value, chainId)
                    buildMessage(r.txHash, null, null)
                }
                withContext(Dispatchers.Main) {
                    statusTextView.text = message
                    sendButton.isEnabled = true
                    autoSendButton.isEnabled = true
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "✗ ${e.message ?: "Send failed"}"
                    sendButton.isEnabled = true
                    autoSendButton.isEnabled = true
                }
            }
        }
    }

    private fun buildMessage(txHash: String, autoApproved: Boolean?, reason: String?): String {
        val sb = StringBuilder()
        sb.append("✓ Transaction submitted to the blockchain network")
        if (txHash.isNotEmpty()) sb.append("\nHash: ").append(txHash)
        if (autoApproved != null) {
            sb.append("\nAuto-approved: ").append(if (autoApproved) "yes" else "no")
            if (reason.isNotEmpty()) sb.append(" (").append(reason).append(")")
        }
        return sb.toString()
    }
}
