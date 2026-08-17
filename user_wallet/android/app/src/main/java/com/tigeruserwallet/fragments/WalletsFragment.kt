package com.tigeruserwallet.fragments

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import android.widget.Button
import android.widget.CheckBox
import android.widget.EditText
import android.widget.Spinner
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
import androidx.fragment.app.Fragment
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.tigeruserwallet.R
import com.tigeruserwallet.adapters.WalletAdapter
import com.tigeruserwallet.api.UserWalletApiService
import com.tigeruserwallet.util.CredentialManagerHelper
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class WalletsFragment : Fragment() {
    private lateinit var walletsRecyclerView: RecyclerView
    private lateinit var addWalletButton: Button
    private lateinit var importWalletButton: Button
    private lateinit var passkeyCreateWalletButton: Button

    private val chains = arrayOf("Ethereum (1)", "BNB Chain (56)", "Polygon (137)")
    private val chainIds = intArrayOf(1, 56, 137)

    // Set by StartFragment after guestAuth so the first wallet action (create
    // vs import) opens automatically. Reset to NONE once consumed.
    var entryMode: Mode = Mode.NONE

    enum class Mode { NONE, CREATE, IMPORT }

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_wallets, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        walletsRecyclerView = view.findViewById(R.id.walletsRecyclerView)
        addWalletButton = view.findViewById(R.id.addWalletButton)
        importWalletButton = view.findViewById(R.id.importWalletButton)
        passkeyCreateWalletButton = view.findViewById(R.id.passkeyCreateWalletButton)
        walletsRecyclerView.layoutManager = LinearLayoutManager(requireContext())
        addWalletButton.setOnClickListener { showAddWalletDialog() }
        importWalletButton.setOnClickListener { showImportWalletDialog() }
        passkeyCreateWalletButton.setOnClickListener { createPasskeyWallet() }
        loadWallets()

        // If launched from the Start screen, surface the matching dialog once.
        view.post {
            when (entryMode) {
                Mode.IMPORT -> {
                    entryMode = Mode.NONE
                    showImportWalletDialog()
                }
                Mode.CREATE -> {
                    entryMode = Mode.NONE
                    showAddWalletDialog()
                }
                else -> { /* no auto-action */ }
            }
        }
    }

    private fun loadWallets() {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val wallets = UserWalletApiService.getWallets()
                withContext(Dispatchers.Main) {
                    walletsRecyclerView.adapter = WalletAdapter(wallets) { w -> showLockSetupDialog(w) }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), "Failed to load wallets", Toast.LENGTH_SHORT).show()
                }
            }
        }
    }

    private fun showAddWalletDialog() {
        val dialogView = layoutInflater.inflate(R.layout.dialog_add_wallet, null)
        val nameInput = dialogView.findViewById<EditText>(R.id.walletNameInput)
        val passwordInput = dialogView.findViewById<EditText>(R.id.walletPasswordInput)
        val chainSpinner = dialogView.findViewById<Spinner>(R.id.chainSpinner)

        chainSpinner.adapter = ArrayAdapter(requireContext(), android.R.layout.simple_spinner_dropdown_item, chains)

        AlertDialog.Builder(requireContext())
            .setTitle("Create Wallet")
            .setView(dialogView)
            .setPositiveButton("Create") { _, _ ->
                val name = nameInput.text.toString()
                val password = passwordInput.text.toString()
                val chainId = chainIds[chainSpinner.selectedItemPosition]
                if (password.length < 8) {
                    Toast.makeText(requireContext(), "Password must be at least 8 chars", Toast.LENGTH_SHORT).show()
                    return@setPositiveButton
                }
                createWallet(name, password, chainId)
            }
            .setNegativeButton("Cancel", null)
            .show()
    }

    private fun createWallet(name: String, password: String, chainId: Int) {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val wallet = UserWalletApiService.createWallet(name, password, chainId)
                withContext(Dispatchers.Main) {
                    if (wallet.mnemonic != null) {
                        showMnemonicDialog(wallet.mnemonic)
                    }
                    loadWallets()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), e.message ?: "Failed to create wallet", Toast.LENGTH_SHORT).show()
                }
            }
        }
    }

    // Shows the freshly-generated recovery phrase with a Copy button.
    private fun showMnemonicDialog(mnemonic: String) {
        AlertDialog.Builder(requireContext())
            .setTitle("Save your recovery phrase")
            .setMessage("Store this securely — it controls your funds:\n\n$mnemonic")
            .setPositiveButton("I've saved it", null)
            .setNeutralButton("Copy") { _, _ ->
                val cm = requireContext()
                    .getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                cm.setPrimaryClip(ClipData.newPlainText("mnemonic", mnemonic))
                Toast.makeText(requireContext(), "Recovery phrase copied", Toast.LENGTH_SHORT).show()
            }
            .show()
    }

    private fun showImportWalletDialog() {
        val dialogView = layoutInflater.inflate(R.layout.dialog_import_wallet, null)
        val nameInput = dialogView.findViewById<EditText>(R.id.importNameInput)
        val mnemonicInput = dialogView.findViewById<EditText>(R.id.importMnemonicInput)
        val passwordInput = dialogView.findViewById<EditText>(R.id.importPasswordInput)
        val chainSpinner = dialogView.findViewById<Spinner>(R.id.importChainSpinner)

        chainSpinner.adapter = ArrayAdapter(requireContext(), android.R.layout.simple_spinner_dropdown_item, chains)

        AlertDialog.Builder(requireContext())
            .setTitle("Import Wallet")
            .setView(dialogView)
            .setPositiveButton("Import") { _, _ ->
                val name = nameInput.text.toString()
                val mnemonic = mnemonicInput.text.toString().trim()
                val password = passwordInput.text.toString()
                val chainId = chainIds[chainSpinner.selectedItemPosition]
                if (mnemonic.isEmpty() || password.length < 8) {
                    Toast.makeText(requireContext(), "Enter a mnemonic and a password (min 8 chars)", Toast.LENGTH_SHORT).show()
                    return@setPositiveButton
                }
                importWallet(name, password, mnemonic, chainId)
            }
            .setNegativeButton("Cancel", null)
            .show()
    }

    private fun importWallet(label: String, password: String, mnemonic: String, chainId: Int) {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                UserWalletApiService.importWallet(label, password, mnemonic, chainId, null)
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), "Wallet imported", Toast.LENGTH_SHORT).show()
                    loadWallets()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), e.message ?: "Failed to import wallet", Toast.LENGTH_SHORT).show()
                }
            }
        }
    }

    // --- App-lock setup -------------------------------------------------------

    /**
     * Per-wallet "Setup App Lock" dialog. Implements the passcode path for real
     * (`setupLock(walletId, {"passcode": ...})`) and surfaces a genuine passkey
     * attempt via [CredentialManagerHelper] when the gradle dependency is present.
     * The passkey path is never faked.
     */
    private fun showLockSetupDialog(wallet: UserWalletApiService.Wallet) {
        val dialogView = layoutInflater.inflate(R.layout.dialog_lock_setup, null)
        val passcodeInput = dialogView.findViewById<EditText>(R.id.lockPasscodeInput)
        val confirmInput = dialogView.findViewById<EditText>(R.id.lockPasscodeConfirmInput)
        val usePasskeyCheckBox = dialogView.findViewById<CheckBox>(R.id.usePasskeyCheckBox)
        val passkeyNote = dialogView.findViewById<TextView>(R.id.passkeyUnavailableNote)

        if (!CredentialManagerHelper.isAvailable) {
            passkeyNote.visibility = View.VISIBLE
            usePasskeyCheckBox.isEnabled = false
        }

        AlertDialog.Builder(requireContext())
            .setTitle("Setup App Lock — ${wallet.label}")
            .setView(dialogView)
            .setPositiveButton("Save") { _, _ ->
                if (usePasskeyCheckBox.isChecked && CredentialManagerHelper.isAvailable) {
                    setupLockWithPasskey(wallet)
                } else {
                    val passcode = passcodeInput.text.toString()
                    val confirm = confirmInput.text.toString()
                    if (passcode.length < 4 || passcode != confirm) {
                        Toast.makeText(requireContext(), "Passcodes must match and be ≥ 4 digits", Toast.LENGTH_SHORT).show()
                        return@setPositiveButton
                    }
                    setupLockWithPasscode(wallet, passcode)
                }
            }
            .setNegativeButton("Cancel", null)
            .show()
    }

    private fun setupLockWithPasscode(wallet: UserWalletApiService.Wallet, passcode: String) {
        val params = UserWalletApiService.LockParams(passcode = passcode)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                UserWalletApiService.setupLock(wallet.id, params)
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), "App lock set (passcode)", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), e.message ?: "Failed to set lock", Toast.LENGTH_SHORT).show()
                }
            }
        }
    }

    private fun setupLockWithPasskey(wallet: UserWalletApiService.Wallet) {
        // Real Credential Manager passkey registration; runs only when the
        // androidx.credentials gradle dep is wired. Wrapped in try/catch so the
        // app never crashes if the platform refuses or the dep is missing.
        try {
            CredentialManagerHelper.createPasskey(this, wallet) { credential ->
                CoroutineScope(Dispatchers.IO).launch {
                    try {
                        val params = UserWalletApiService.LockParams(
                            passkeyCredentialId = credential.credentialId,
                            passkeyPublicKey = credential.publicKey
                        )
                        UserWalletApiService.setupLock(wallet.id, params)
                        withContext(Dispatchers.Main) {
                            Toast.makeText(requireContext(), "App lock set (passkey)", Toast.LENGTH_SHORT).show()
                        }
                    } catch (e: Exception) {
                        withContext(Dispatchers.Main) {
                            Toast.makeText(requireContext(), e.message ?: "Failed to set passkey lock", Toast.LENGTH_SHORT).show()
                        }
                    }
                }
            }
        } catch (e: Throwable) {
            Toast.makeText(requireContext(), e.message ?: "Passkey unavailable", Toast.LENGTH_SHORT).show()
        }
    }

    // --- Passkey wallet creation ----------------------------------------------

    /**
     * "Create with Passkey": drives a real Credential Manager passkey creation
     * (when the gradle dep is present) and posts the credential to
     * [UserWalletApiService.passkeyCreateWallet], then shows the returned
     * mnemonic with a Copy action. No mock fallback.
     */
    private fun createPasskeyWallet() {
        if (!CredentialManagerHelper.isAvailable) {
            Toast.makeText(
                requireContext(),
                "Passkey wallet creation requires the androidx.credentials gradle dependency.",
                Toast.LENGTH_LONG
            ).show()
            return
        }
        try {
            CredentialManagerHelper.createPasskeyForWallet(this) { credential ->
                val params = UserWalletApiService.PasskeyWalletParams(
                    label = "Passkey Wallet",
                    chainId = 1,
                    accountIndex = 0,
                    credentialId = credential.credentialId,
                    publicKey = credential.publicKey
                )
                CoroutineScope(Dispatchers.IO).launch {
                    try {
                        val res = UserWalletApiService.passkeyCreateWallet(params)
                        withContext(Dispatchers.Main) {
                            val mnemonic = res.mnemonic
                            if (!mnemonic.isNullOrEmpty()) {
                                showMnemonicDialog(mnemonic)
                            } else {
                                Toast.makeText(requireContext(), "Wallet created", Toast.LENGTH_SHORT).show()
                                loadWallets()
                            }
                        }
                    } catch (e: Exception) {
                        withContext(Dispatchers.Main) {
                            Toast.makeText(requireContext(), e.message ?: "Passkey wallet creation failed", Toast.LENGTH_SHORT).show()
                        }
                    }
                }
            }
        } catch (e: Throwable) {
            Toast.makeText(requireContext(), e.message ?: "Passkey unavailable", Toast.LENGTH_SHORT).show()
        }
    }

    private fun showMnemonicDialog(mnemonic: String) {
        val dialogView = layoutInflater.inflate(R.layout.dialog_mnemonic, null)
        val mnemonicView = dialogView.findViewById<TextView>(R.id.mnemonicTextView)
        mnemonicView.text = mnemonic
        AlertDialog.Builder(requireContext())
            .setTitle("Wallet created — save your mnemonic")
            .setView(dialogView)
            .setPositiveButton("Copy") { _, _ ->
                val clipboard = requireContext().getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                clipboard.setPrimaryClip(ClipData.newPlainText("mnemonic", mnemonic))
                Toast.makeText(requireContext(), "Mnemonic copied", Toast.LENGTH_SHORT).show()
                loadWallets()
            }
            .setNegativeButton("Done", null)
            .show()
    }
}
