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
    private lateinit var passcodeInput: EditText
    private lateinit var sendButton: Button
    private lateinit var autoSendButton: Button
    private lateinit var unlockButton: Button
    private lateinit var statusTextView: TextView

    private val chains = arrayOf("Ethereum (1)", "BNB Chain (56)", "Polygon (137)")
    private val chainIds = intArrayOf(1, 56, 137)

    private var wallets: List<UserWalletApiService.Wallet> = emptyList()

    // Short-lived unlock token issued by /wallets/:id/unlock. While present, the
    // wallet password is optional and the token authorizes send/auto-send.
    private var unlockToken: String? = null

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
        passcodeInput = view.findViewById(R.id.passcodeInput)
        sendButton = view.findViewById(R.id.sendButton)
        autoSendButton = view.findViewById(R.id.autoSendButton)
        unlockButton = view.findViewById(R.id.unlockButton)
        statusTextView = view.findViewById(R.id.statusTextView)

        chainSpinner.adapter =
            ArrayAdapter(requireContext(), android.R.layout.simple_spinner_dropdown_item, chains)

        sendButton.setOnClickListener { performSend(autoFirst = true) }
        autoSendButton.setOnClickListener { performSend(autoFirst = true) }
        unlockButton.setOnClickListener { unlockWallet() }

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

    /**
     * Primary send path. Both the primary [sendButton] and the explicit
     * [autoSendButton] route here with `autoFirst = true`: the request first
     * attempts `autoSendTransaction` (auto sign + auto approval from
     * superAdmin / MasterWallet owner / Admin panel). If auto-send fails, it
     * transparently falls back to the manual `sendTransaction` so a wallet send
     * still goes through when the wallet is unlocked. Either path surfaces the
     * "Transaction submitted to the blockchain network" success message via
     * [buildMessage].
     */
    private fun performSend(autoFirst: Boolean) {
        val wallet = wallets.getOrNull(walletSpinner.selectedItemPosition) ?: run {
            Toast.makeText(requireContext(), "Select a wallet", Toast.LENGTH_SHORT).show()
            return
        }
        val to = recipientInput.text.toString().trim()
        val value = amountInput.text.toString().trim()
        val password = passwordInput.text.toString()
        val chainId = chainIds[chainSpinner.selectedItemPosition]

        if (to.isEmpty() || value.isEmpty()) {
            Toast.makeText(requireContext(), "Enter recipient and amount", Toast.LENGTH_SHORT).show()
            return
        }
        // Password is optional once we hold an unlock token; otherwise require a
        // strong password exactly as the original flow did.
        if (unlockToken == null) {
            if (password.isEmpty()) {
                Toast.makeText(requireContext(), "Enter password or unlock passwordlessly", Toast.LENGTH_SHORT).show()
                return
            }
            if (password.length < 8) {
                Toast.makeText(requireContext(), "Password must be at least 8 chars", Toast.LENGTH_SHORT).show()
                return
            }
        }

        sendButton.isEnabled = false
        autoSendButton.isEnabled = false
        statusTextView.text = "Submitting..."
        CoroutineScope(Dispatchers.IO).launch {
            val message = try {
                // Primary: auto-send (auto sign + auto approval).
                val r = UserWalletApiService.autoSendTransaction(
                    wallet.id, password, to, value, chainId, null, unlockToken
                )
                buildMessage(r.txHash, r.autoApproved, r.autoApprovalReason)
            } catch (autoErr: Exception) {
                if (!autoFirst) throw autoErr
                // Fallback: manual on-chain send when auto-send is unavailable
                // (e.g. no auto-approval policy / Admin panel offline).
                val r = UserWalletApiService.sendTransaction(
                    wallet.id, password, to, value, chainId, unlockToken
                )
                buildMessage(r.txHash, null, null)
            }
            withContext(Dispatchers.Main) {
                statusTextView.text = message
                sendButton.isEnabled = true
                autoSendButton.isEnabled = true
            }
        }
    }

    /**
     * Passwordless unlock: posts the app-lock passcode to /wallets/:id/unlock and
     * stores the returned [unlock_token]. The token is then forwarded as the
     * [unlockToken] param to [sendTransaction] / [autoSendTransaction], making the
     * wallet password field optional.
     */
    private fun unlockWallet() {
        val wallet = wallets.getOrNull(walletSpinner.selectedItemPosition) ?: run {
            Toast.makeText(requireContext(), "Select a wallet", Toast.LENGTH_SHORT).show()
            return
        }
        val passcode = passcodeInput.text.toString()
        if (passcode.length < 4) {
            Toast.makeText(requireContext(), "Enter the app-lock passcode", Toast.LENGTH_SHORT).show()
            return
        }

        unlockButton.isEnabled = false
        statusTextView.text = "Unlocking..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val params = UserWalletApiService.UnlockParams(passcode = passcode)
                val res = UserWalletApiService.unlockWallet(wallet.id, params)
                val token = res.optString("unlock_token", "")
                withContext(Dispatchers.Main) {
                    if (token.isNotEmpty()) {
                        unlockToken = token
                        statusTextView.text = "✓ Wallet unlocked — password optional now"
                        Toast.makeText(requireContext(), "Unlocked", Toast.LENGTH_SHORT).show()
                    } else {
                        statusTextView.text = "✗ No unlock token returned"
                    }
                    unlockButton.isEnabled = true
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "✗ ${e.message ?: "Unlock failed"}"
                    unlockButton.isEnabled = true
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
