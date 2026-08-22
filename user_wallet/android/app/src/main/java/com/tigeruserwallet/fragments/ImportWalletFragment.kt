package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import android.widget.Toast
import androidx.appcompat.widget.AppCompatSpinner
import androidx.fragment.app.Fragment
import com.google.android.material.button.MaterialButton
import com.google.android.material.progressindicator.CircularProgressIndicator
import com.google.android.material.textfield.TextInputEditText
import com.google.android.material.textfield.TextInputLayout
import com.tigeruserwallet.MainActivity
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Import Wallet flow (mirrors web ImportWallet.tsx):
 *  paste seed (12/24 words) + name + password ->
 *  ensureSession() (transparent) ->
 *  createWalletTyped(label, password, chainId, mnemonic) ->
 *  wallet ready (NO backup screen — the user already has the seed).
 */
class ImportWalletFragment : Fragment() {

    private lateinit var seedInput: TextInputEditText
    private lateinit var seedLayout: TextInputLayout
    private lateinit var nameInput: TextInputEditText
    private lateinit var nameLayout: TextInputLayout
    private lateinit var networkSpinner: AppCompatSpinner
    private lateinit var passwordInput: TextInputEditText
    private lateinit var passwordLayout: TextInputLayout
    private lateinit var importButton: MaterialButton
    private lateinit var progress: CircularProgressIndicator

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        return inflater.inflate(R.layout.fragment_import_wallet, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        seedInput = view.findViewById(R.id.seedInput)
        seedLayout = view.findViewById(R.id.seedLayout)
        nameInput = view.findViewById(R.id.nameInput)
        nameLayout = view.findViewById(R.id.nameLayout)
        networkSpinner = view.findViewById(R.id.networkSpinner)
        passwordInput = view.findViewById(R.id.passwordInput)
        passwordLayout = view.findViewById(R.id.passwordLayout)
        importButton = view.findViewById(R.id.importButton)
        progress = view.findViewById(R.id.importProgress)

        networkSpinner.adapter = ArrayAdapter(
            requireContext(),
            android.R.layout.simple_spinner_dropdown_item,
            UserWalletApiService.CHAINS.map { "${it.name} (${it.symbol} · ${it.id})" }
        )

        importButton.setOnClickListener { onSubmit() }
    }

    private fun onSubmit() {
        seedLayout.error = null
        nameLayout.error = null
        passwordLayout.error = null

        val mnemonic = seedInput.text?.toString().orEmpty().trim()
        val label = nameInput.text?.toString().orEmpty().trim()
        val password = passwordInput.text?.toString().orEmpty()
        val chainId = UserWalletApiService.CHAINS[networkSpinner.selectedItemPosition].id

        val words = mnemonic.split(Regex("\\s+")).filter { it.isNotEmpty() }
        if (words.size != 12 && words.size != 24) {
            seedLayout.error = getString(R.string.err_seed_count, words.size)
            return
        }
        if (label.isEmpty()) {
            nameLayout.error = getString(R.string.err_name)
            return
        }
        if (password.length < 8) {
            passwordLayout.error = getString(R.string.err_password_short)
            return
        }

        setBusy(true)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                UserWalletApiService.ensureSession()
                val wallet = UserWalletApiService.createWalletTyped(
                    label = label,
                    password = password,
                    chainId = chainId,
                    mnemonic = mnemonic
                )
                UserWalletApiService.rememberWallet(wallet.id)
                withContext(Dispatchers.Main) {
                    setBusy(false)
                    (activity as? MainActivity)?.enterApp()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    setBusy(false)
                    Toast.makeText(
                        requireContext(),
                        e.message ?: getString(R.string.err_import_failed),
                        Toast.LENGTH_LONG
                    ).show()
                }
            }
        }
    }

    private fun setBusy(busy: Boolean) {
        progress.visibility = if (busy) View.VISIBLE else View.GONE
        importButton.isEnabled = !busy
    }
}
