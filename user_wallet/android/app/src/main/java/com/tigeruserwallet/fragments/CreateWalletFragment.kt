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
 * Create Wallet flow (mirrors web CreateWallet.tsx):
 *  name + network select + password + confirm ->
 *  UserWalletApiService.ensureSession() (transparent no-registration) ->
 *  createWalletTyped(label, password, chainId) ->
 *  BackupMnemonicFragment (mnemonic display + backup options).
 *
 * No stubs: the mnemonic comes from the backend; backup uses real AES-256-GCM
 * + Google Drive REST API v3 (or is honestly disabled if unconfigured).
 */
class CreateWalletFragment : Fragment() {

    private lateinit var nameInput: TextInputEditText
    private lateinit var nameLayout: TextInputLayout
    private lateinit var networkSpinner: AppCompatSpinner
    private lateinit var passwordInput: TextInputEditText
    private lateinit var passwordLayout: TextInputLayout
    private lateinit var confirmInput: TextInputEditText
    private lateinit var confirmLayout: TextInputLayout
    private lateinit var createButton: MaterialButton
    private lateinit var progress: CircularProgressIndicator

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        return inflater.inflate(R.layout.fragment_create_wallet, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        nameInput = view.findViewById(R.id.nameInput)
        nameLayout = view.findViewById(R.id.nameLayout)
        networkSpinner = view.findViewById(R.id.networkSpinner)
        passwordInput = view.findViewById(R.id.passwordInput)
        passwordLayout = view.findViewById(R.id.passwordLayout)
        confirmInput = view.findViewById(R.id.confirmInput)
        confirmLayout = view.findViewById(R.id.confirmLayout)
        createButton = view.findViewById(R.id.createButton)
        progress = view.findViewById(R.id.createProgress)

        val chains = UserWalletApiService.CHAINS
        networkSpinner.adapter = ArrayAdapter(
            requireContext(),
            android.R.layout.simple_spinner_dropdown_item,
            chains.map { "${it.name} (${it.symbol} · ${it.id})" }
        )

        createButton.setOnClickListener { onSubmit() }
    }

    private fun onSubmit() {
        nameLayout.error = null
        passwordLayout.error = null
        confirmLayout.error = null

        val label = nameInput.text?.toString().orEmpty().trim()
        val password = passwordInput.text?.toString().orEmpty()
        val confirm = confirmInput.text?.toString().orEmpty()
        val chainId = UserWalletApiService.CHAINS[networkSpinner.selectedItemPosition].id

        if (label.isEmpty()) {
            nameLayout.error = getString(R.string.err_name)
            return
        }
        if (password.length < 8) {
            passwordLayout.error = getString(R.string.err_password_short)
            return
        }
        if (password != confirm) {
            confirmLayout.error = getString(R.string.err_password_match)
            return
        }

        setBusy(true)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                // Transparent no-registration session (auto-provisions a random
                // device-bound identity; the user never sees a login form).
                UserWalletApiService.ensureSession()
                val wallet = UserWalletApiService.createWalletTyped(
                    label = label,
                    password = password,
                    chainId = chainId
                )
                UserWalletApiService.rememberWallet(wallet.id)

                withContext(Dispatchers.Main) {
                    setBusy(false)
                    if (wallet.mnemonic != null) {
                        // Fresh create -> show the backup screen with the real mnemonic.
                        parentFragmentManager.beginTransaction()
                            .replace(
                                R.id.fragmentContainer,
                                BackupMnemonicFragment.newInstance(
                                    wallet.id,
                                    wallet.mnemonic,
                                    wallet.chainId,
                                    label,
                                    password
                                )
                            )
                            .addToBackStack(null)
                            .commit()
                    } else {
                        // Defensive: backend didn't return a mnemonic — proceed to app.
                        (activity as? MainActivity)?.enterApp()
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    setBusy(false)
                    Toast.makeText(
                        requireContext(),
                        e.message ?: getString(R.string.err_create_failed),
                        Toast.LENGTH_LONG
                    ).show()
                }
            }
        }
    }

    private fun setBusy(busy: Boolean) {
        progress.visibility = if (busy) View.VISIBLE else View.GONE
        createButton.isEnabled = !busy
    }
}
