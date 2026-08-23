package com.tigeruserwallet.fragments

import android.os.Bundle
import android.text.Editable
import android.text.TextWatcher
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
import java.util.Locale

class SendFragment : Fragment() {
    private lateinit var walletSpinner: Spinner
    private lateinit var chainSpinner: Spinner
    private lateinit var recipientInput: EditText
    private lateinit var amountInput: EditText
    private lateinit var passwordInput: EditText
    private lateinit var passcodeInput: EditText
    private lateinit var maxFeeInput: EditText
    private lateinit var maxPriorityInput: EditText
    private lateinit var sendButton: Button
    private lateinit var autoSendButton: Button
    private lateinit var simulateButton: Button
    private lateinit var unlockButton: Button
    private lateinit var ensStatusText: TextView
    private lateinit var simulateResultText: TextView
    private lateinit var statusTextView: TextView

    private val chains = arrayOf("Ethereum (1)", "BNB Chain (56)", "Polygon (137)")
    private val chainIds = intArrayOf(1, 56, 137)

    private var wallets: List<UserWalletApiService.Wallet> = emptyList()

    // Short-lived unlock token issued by /wallets/:id/unlock. While present, the
    // wallet password is optional and the token authorizes send/auto-send.
    private var unlockToken: String? = null

    // Resolved 0x recipient when the user typed an ENS name (mirror web Send.tsx).
    private var resolvedEnsName: String? = null
    private var resolvedEnsAddress: String? = null

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
        maxFeeInput = view.findViewById(R.id.maxFeeInput)
        maxPriorityInput = view.findViewById(R.id.maxPriorityInput)
        sendButton = view.findViewById(R.id.sendButton)
        autoSendButton = view.findViewById(R.id.autoSendButton)
        simulateButton = view.findViewById(R.id.simulateButton)
        unlockButton = view.findViewById(R.id.unlockButton)
        ensStatusText = view.findViewById(R.id.ensStatusText)
        simulateResultText = view.findViewById(R.id.simulateResultText)
        statusTextView = view.findViewById(R.id.statusTextView)

        chainSpinner.adapter =
            ArrayAdapter(requireContext(), android.R.layout.simple_spinner_dropdown_item, chains)

        sendButton.setOnClickListener { performSend(autoFirst = true) }
        autoSendButton.setOnClickListener { performSend(autoFirst = true) }
        simulateButton.setOnClickListener { performSimulate() }
        unlockButton.setOnClickListener { unlockWallet() }

        recipientInput.addTextChangedListener(object : TextWatcher {
            override fun beforeTextChanged(s: CharSequence?, start: Int, count: Int, after: Int) {}
            override fun onTextChanged(s: CharSequence?, start: Int, before: Int, count: Int) {}
            override fun afterTextChanged(s: Editable?) {
                onRecipientChanged(s?.toString()?.trim() ?: "")
            }
        })

        loadWallets()
    }

    /**
     * Live ENS feedback while typing (mirror web resolveRecipient): a full 0x
     * address is accepted as-is; a name ending in .eth is resolved through the
     * backend and the resolved address is shown under the input.
     */
    private fun onRecipientChanged(raw: String) {
        resolvedEnsName = null
        resolvedEnsAddress = null
        simulateResultText.text = ""
        when {
            raw.isEmpty() -> ensStatusText.text = ""
            ADDRESS_REGEX.matches(raw) -> ensStatusText.text = ""
            raw.endsWith(".eth", ignoreCase = true) -> {
                ensStatusText.text = "Resolving ENS…"
                CoroutineScope(Dispatchers.IO).launch {
                    try {
                        val res = UserWalletApiService.resolveENS(raw)
                        withContext(Dispatchers.Main) {
                            // Ignore stale results if the user kept typing.
                            if (recipientInput.text.toString().trim() != raw) return@withContext
                            resolvedEnsName = res.name
                            resolvedEnsAddress = res.address
                            ensStatusText.text =
                                "✓ ${res.name} → ${shortenAddress(res.address)}"
                        }
                    } catch (e: Exception) {
                        withContext(Dispatchers.Main) {
                            if (recipientInput.text.toString().trim() != raw) return@withContext
                            ensStatusText.text = "✗ ${e.message ?: "ENS resolution failed"}"
                        }
                    }
                }
            }
            else -> ensStatusText.text = ""
        }
    }

    /**
     * Resolve the recipient field to a 0x address: returns the input directly
     * when it is already an address, resolves it via /ens/resolve when it ends
     * in .eth, and throws otherwise. Callers run on Dispatchers.IO.
     */
    private suspend fun resolveRecipientAddress(raw: String): String {
        if (ADDRESS_REGEX.matches(raw)) return raw
        resolvedEnsAddress?.let { cached ->
            if (resolvedEnsName.equals(raw, ignoreCase = true) && ADDRESS_REGEX.matches(cached)) {
                return cached
            }
        }
        if (raw.endsWith(".eth", ignoreCase = true)) {
            val res = UserWalletApiService.resolveENS(raw)
            withContext(Dispatchers.Main) {
                resolvedEnsName = res.name
                resolvedEnsAddress = res.address
                ensStatusText.text = "✓ ${res.name} → ${shortenAddress(res.address)}"
            }
            return res.address
        }
        throw IllegalArgumentException("Enter a valid recipient (0x address or .eth name)")
    }

    /**
     * Pre-sign simulation (mirror web handleSimulate): dry-runs the exact tx
     * the user is about to send — from the selected wallet address to the
     * resolved recipient with the entered amount on the selected chain — and
     * surfaces success / revert reason / gas estimate before signing.
     */
    private fun performSimulate() {
        val wallet = wallets.getOrNull(walletSpinner.selectedItemPosition) ?: run {
            Toast.makeText(requireContext(), "Select a wallet", Toast.LENGTH_SHORT).show()
            return
        }
        val rawTo = recipientInput.text.toString().trim()
        val value = amountInput.text.toString().trim()
        val chainId = chainIds[chainSpinner.selectedItemPosition]

        if (rawTo.isEmpty() || value.isEmpty()) {
            Toast.makeText(requireContext(), "Enter recipient and amount", Toast.LENGTH_SHORT).show()
            return
        }

        simulateButton.isEnabled = false
        simulateResultText.text = "Simulating…"
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val to = resolveRecipientAddress(rawTo)
                val sim = UserWalletApiService.simulateTransaction(
                    chainId = chainId,
                    from = wallet.address,
                    to = to,
                    value = value
                )
                withContext(Dispatchers.Main) {
                    simulateResultText.text = buildSimMessage(sim)
                    simulateButton.isEnabled = true
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    simulateResultText.text = "✗ ${e.message ?: "Simulation failed"}"
                    simulateButton.isEnabled = true
                }
            }
        }
    }

    private fun buildSimMessage(sim: UserWalletApiService.SimulationResult): String {
        val sb = StringBuilder()
        if (sim.success && !sim.willRevert) {
            sb.append("✓ Simulation succeeded")
            sb.append("\nGas estimate: ").append(sim.gasEstimate)
        } else {
            sb.append("✗ Transaction will revert")
            if (!sim.revertReason.isNullOrEmpty()) sb.append("\n").append(sim.revertReason)
            if (!sim.estimateError.isNullOrEmpty()) sb.append("\n").append(sim.estimateError)
        }
        sim.estimatedCostWei?.takeIf { it.isNotEmpty() }?.let { costWei ->
            val cost = UserWalletApiService.weiToFloat(costWei)
            sb.append("\nEstimated cost: ")
                .append(String.format(Locale.US, "%.6f", cost))
                .append(" ")
                .append(UserWalletApiService.symbolFor(sim.chainId))
        }
        return sb.toString()
    }

    private fun shortenAddress(address: String): String =
        if (address.length > 16) "${address.take(10)}…${address.takeLast(6)}" else address

    /** Optional EIP-1559 overrides; blank inputs fall back to backend auto. */
    private fun maxFeeGwei(): String? =
        maxFeeInput.text.toString().trim().ifEmpty { null }

    private fun maxPriorityGwei(): String? =
        maxPriorityInput.text.toString().trim().ifEmpty { null }

    companion object {
        private val ADDRESS_REGEX = Regex("^0x[a-fA-F0-9]{40}$")
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
        val rawTo = recipientInput.text.toString().trim()
        val value = amountInput.text.toString().trim()
        val password = passwordInput.text.toString()
        val chainId = chainIds[chainSpinner.selectedItemPosition]
        val maxFee = maxFeeGwei()
        val maxPriority = maxPriorityGwei()

        if (rawTo.isEmpty() || value.isEmpty()) {
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
            try {
                // Resolve ENS names (.eth) to a 0x address before signing.
                val to = resolveRecipientAddress(rawTo)
                val message = try {
                    // Primary: auto-send (auto sign + auto approval).
                    val r = UserWalletApiService.autoSendTransaction(
                        wallet.id, password, to, value, chainId, null, unlockToken,
                        maxFeeGwei = maxFee, maxPriorityGwei = maxPriority
                    )
                    buildMessage(r.txHash, r.autoApproved, r.autoApprovalReason)
                } catch (autoErr: Exception) {
                    if (!autoFirst) throw autoErr
                    // Fallback: manual on-chain send when auto-send is unavailable
                    // (e.g. no auto-approval policy / Admin panel offline).
                    val r = UserWalletApiService.sendTransaction(
                        wallet.id, password, to, value, chainId, unlockToken,
                        maxFeeGwei = maxFee, maxPriorityGwei = maxPriority
                    )
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
