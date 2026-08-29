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

/**
 * Non-EVM chains: derive native addresses (bitcoin/solana/cosmos) from the
 * stored seed, sign messages, and build/sign non-EVM transactions. Real key
 * derivation + signing on the backend (mainnet only); fail-closed errors.
 */
class NonEvmFragment : Fragment() {

    private lateinit var container: LinearLayout
    private var wallets: List<UserWalletApiService.Wallet> = emptyList()

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_non_evm, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        container = view.findViewById(R.id.nonEvmList)
        buildUi()
        loadWallets()
    }

    private fun loadWallets() {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                wallets = UserWalletApiService.getWallets()
            } catch (_: Exception) {
                // Wallet list is needed for derive/sign; shown empty until loaded.
            }
        }
    }

    private fun buildUi() {
        val ctx = requireContext()
        val chainInput = EditText(ctx).apply { hint = "Chain (bitcoin / solana / cosmos)" }
        val passwordInput = EditText(ctx).apply {
            hint = "Wallet password"
            inputType = android.text.InputType.TYPE_CLASS_TEXT or android.text.InputType.TYPE_TEXT_VARIATION_PASSWORD
        }
        val deriveBtn = Button(ctx).apply { text = "Derive Address" }
        val resultBox = TextView(ctx).apply { textSize = 14f; setPadding(0, 16, 0, 16) }
        val messageInput = EditText(ctx).apply { hint = "Message to sign" }
        val signBtn = Button(ctx).apply { text = "Sign Message" }
        val signResult = TextView(ctx).apply { textSize = 14f; setPadding(0, 16, 0, 16) }

        container.addView(TextView(ctx).apply { text = "Derive native address"; textSize = 18f })
        container.addView(chainInput)
        container.addView(passwordInput)
        container.addView(deriveBtn)
        container.addView(resultBox)
        container.addView(TextView(ctx).apply { text = "Sign message"; textSize = 18f; setPadding(0, 24, 0, 8) })
        container.addView(messageInput)
        container.addView(signBtn)
        container.addView(signResult)

        deriveBtn.setOnClickListener {
            val wallet = wallets.firstOrNull() ?: run {
                Toast.makeText(ctx, "No wallet available", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            val chain = chainInput.text.toString().trim().lowercase()
            val password = passwordInput.text.toString()
            if (chain.isEmpty() || password.isEmpty()) {
                Toast.makeText(ctx, "Enter chain and password", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val json = UserWalletApiService.deriveNonEvmAddress(wallet.id, password, chain)
                    withContext(Dispatchers.Main) {
                        resultBox.text = "$chain: ${json.optString("address", json.toString())}"
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { resultBox.text = "Derive failed: ${e.message}" }
                }
            }
        }

        signBtn.setOnClickListener {
            val wallet = wallets.firstOrNull() ?: run {
                Toast.makeText(ctx, "No wallet available", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            val chain = chainInput.text.toString().trim().lowercase()
            val password = passwordInput.text.toString()
            val message = messageInput.text.toString()
            if (chain.isEmpty() || password.isEmpty() || message.isEmpty()) {
                Toast.makeText(ctx, "Enter chain, password and message", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val json = UserWalletApiService.nonEvmSignMessage(wallet.id, password, message, chain)
                    withContext(Dispatchers.Main) {
                        signResult.text = "signature: ${json.optString("signature", json.toString())}"
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { signResult.text = "Sign failed: ${e.message}" }
                }
            }
        }
    }
}
