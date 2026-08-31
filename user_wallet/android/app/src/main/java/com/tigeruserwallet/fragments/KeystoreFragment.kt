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
import android.widget.LinearLayout
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
 * Keystore export/import. Export calls [exportKeystore] and displays the
 * resulting JSON (with a Copy button using ClipboardManager). Import takes a
 * pasted keystore + password + optional label and calls [importKeystore].
 */
class KeystoreFragment : Fragment() {
    private lateinit var exportWalletSpinner: Spinner
    private lateinit var exportPasswordInput: EditText
    private lateinit var exportButton: Button
    private lateinit var copyButton: Button
    private lateinit var exportResultText: TextView
    private lateinit var importKeystoreInput: EditText
    private lateinit var importPasswordInput: EditText
    private lateinit var importLabelInput: EditText
    private lateinit var importButton: Button
    private lateinit var importResultText: TextView
    private lateinit var progressBar: ProgressBar

    private var wallets: List<UserWalletApiService.Wallet> = emptyList()
    private var exportedKeystore: String = ""

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_keystore, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        exportWalletSpinner = view.findViewById(R.id.ksExportWalletSpinner)
        exportPasswordInput = view.findViewById(R.id.ksExportPasswordInput)
        exportButton = view.findViewById(R.id.ksExportButton)
        copyButton = view.findViewById(R.id.ksCopyButton)
        exportResultText = view.findViewById(R.id.ksExportResultText)
        importKeystoreInput = view.findViewById(R.id.ksImportKeystoreInput)
        importPasswordInput = view.findViewById(R.id.ksImportPasswordInput)
        importLabelInput = view.findViewById(R.id.ksImportLabelInput)
        importButton = view.findViewById(R.id.ksImportButton)
        importResultText = view.findViewById(R.id.ksImportResultText)
        progressBar = view.findViewById(R.id.ksProgressBar)

        exportButton.setOnClickListener { exportKeystore() }
        copyButton.setOnClickListener { copyExported() }
        importButton.setOnClickListener { importKeystore() }
        addEncryptedSeedImport(view)
        loadWallets()
    }

    /**
     * Encrypted-seed restore (POST /wallets/import-encrypted-seed) — restores a
     * wallet from an AES-256-GCM blob previously produced by
     * /wallets/:id/export-encrypted-seed.
     */
    private fun addEncryptedSeedImport(view: View) {
        val root = view as? LinearLayout ?: return
        val ctx = requireContext()
        val header = TextView(ctx).apply {
            text = "Import Encrypted Seed"
            textSize = 18f
            setPadding(0, 32, 0, 8)
        }
        val seedInput = EditText(ctx).apply { hint = "Encrypted seed blob" }
        val pwInput = EditText(ctx).apply {
            hint = "Password"
            inputType = android.text.InputType.TYPE_CLASS_TEXT or android.text.InputType.TYPE_TEXT_VARIATION_PASSWORD
        }
        val labelInput = EditText(ctx).apply { hint = "Label (optional)" }
        val importBtn = Button(ctx).apply { text = "Import Encrypted Seed" }
        val resultText = TextView(ctx).apply { setPadding(0, 8, 0, 8) }
        root.addView(header)
        root.addView(seedInput)
        root.addView(pwInput)
        root.addView(labelInput)
        root.addView(importBtn)
        root.addView(resultText)

        importBtn.setOnClickListener {
            val blob = seedInput.text.toString().trim()
            val pw = pwInput.text.toString()
            val label = labelInput.text.toString().trim()
            if (blob.isEmpty() || pw.length < 8) {
                Toast.makeText(ctx, "Enter the blob and a password (8+ chars)", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            resultText.text = "Importing…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val res = UserWalletApiService.importEncryptedSeed(blob, pw, label.ifEmpty { null })
                    withContext(Dispatchers.Main) {
                        resultText.text = "Wallet restored: ${res.optString("address", res.optString("wallet_id", "ok"))}"
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) {
                        resultText.text = "Import failed: ${e.message}"
                    }
                }
            }
        }
    }

    private fun loadWallets() {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                wallets = UserWalletApiService.getWallets()
                withContext(Dispatchers.Main) {
                    exportWalletSpinner.adapter = ArrayAdapter(
                        requireContext(),
                        android.R.layout.simple_spinner_dropdown_item,
                        wallets.map { "${it.label} \u00b7 ${it.address.take(10)}..." })
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    exportResultText.text = "\u2717 ${e.message ?: "Failed to load wallets"}"
                }
            }
        }
    }

    private fun exportKeystore() {
        val wallet = wallets.getOrNull(exportWalletSpinner.selectedItemPosition) ?: run {
            Toast.makeText(requireContext(), "Select a wallet", Toast.LENGTH_SHORT).show()
            return
        }
        val password = exportPasswordInput.text.toString()
        if (password.isEmpty()) {
            Toast.makeText(requireContext(), "Enter password", Toast.LENGTH_SHORT).show()
            return
        }
        setLoading(true)
        exportButton.isEnabled = false
        exportResultText.text = "Exporting..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val result = UserWalletApiService.exportKeystore(wallet.id, password)
                exportedKeystore = result.toString(2)
                withContext(Dispatchers.Main) {
                    exportResultText.text = exportedKeystore
                    setLoading(false)
                    exportButton.isEnabled = true
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    exportResultText.text = "\u2717 ${e.message ?: "Export failed"}"
                    exportedKeystore = ""
                    setLoading(false)
                    exportButton.isEnabled = true
                }
            }
        }
    }

    private fun copyExported() {
        if (exportedKeystore.isEmpty()) {
            Toast.makeText(requireContext(), "Nothing to copy yet", Toast.LENGTH_SHORT).show()
            return
        }
        val clipboard = requireContext().getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        clipboard.setPrimaryClip(ClipData.newPlainText("keystore", exportedKeystore))
        Toast.makeText(requireContext(), "\u2713 Keystore copied to clipboard", Toast.LENGTH_SHORT).show()
    }

    private fun importKeystore() {
        val keystore = importKeystoreInput.text.toString().trim()
        val password = importPasswordInput.text.toString()
        val label = importLabelInput.text.toString().trim().ifEmpty { null }
        if (keystore.isEmpty() || password.isEmpty()) {
            Toast.makeText(requireContext(), "Paste keystore and enter password", Toast.LENGTH_SHORT).show()
            return
        }
        setLoading(true)
        importButton.isEnabled = false
        importResultText.text = "Importing..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val wallet = UserWalletApiService.importKeystore(keystore, password, label)
                withContext(Dispatchers.Main) {
                    importResultText.text =
                        "\u2713 Imported wallet: ${wallet.label} (${wallet.address.take(12)}...)"
                    setLoading(false)
                    importButton.isEnabled = true
                    loadWallets()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    importResultText.text = "\u2717 ${e.message ?: "Import failed"}"
                    setLoading(false)
                    importButton.isEnabled = true
                }
            }
        }
    }

    private fun setLoading(loading: Boolean) {
        progressBar.visibility = if (loading) View.VISIBLE else View.GONE
    }
}
