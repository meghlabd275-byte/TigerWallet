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
import android.widget.EditText
import android.widget.Spinner
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
import androidx.fragment.app.Fragment
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.tigeruserwallet.R
import com.tigeruserwallet.adapters.WalletAdapter
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class WalletsFragment : Fragment() {
    private lateinit var walletsRecyclerView: RecyclerView
    private lateinit var addWalletButton: Button
    private lateinit var importWalletButton: Button

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
        walletsRecyclerView.layoutManager = LinearLayoutManager(requireContext())
        addWalletButton.setOnClickListener { showAddWalletDialog() }
        importWalletButton.setOnClickListener { showImportWalletDialog() }
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
                    walletsRecyclerView.adapter = WalletAdapter(wallets)
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
}
