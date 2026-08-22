package com.tigeruserwallet.fragments

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.CheckBox
import android.widget.LinearLayout
import android.widget.TextView
import android.widget.Toast
import androidx.activity.result.ActivityResultLauncher
import androidx.activity.result.contract.ActivityResultContracts
import androidx.fragment.app.Fragment
import com.google.android.material.button.MaterialButton
import com.google.android.material.progressindicator.CircularProgressIndicator
import com.tigeruserwallet.MainActivity
import com.tigeruserwallet.R
import com.tigeruserwallet.crypto.EncryptedBackup
import com.tigeruserwallet.crypto.GoogleDriveBackup
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Backup screen (mirrors web BackupMnemonic.tsx): shown only after a fresh
 * create (the backend returns the mnemonic once).
 *
 *  - numbered grid of mnemonic words
 *  - copy-to-clipboard (real ClipboardManager)
 *  - Google Drive backup (real Google Sign-In + Drive REST API v3 via
 *    GoogleDriveBackup; honestly DISABLED with a message when no Google web
 *    client ID is configured — never fakes success)
 *  - download encrypted backup (real AES-256-GCM + PBKDF2 via EncryptedBackup,
 *    saved to app external files dir + share intent)
 *  - confirm checkbox + Continue -> wallet ready (MainActivity.enterApp)
 */
class BackupMnemonicFragment : Fragment() {

    private lateinit var gridContainer: LinearLayout
    private lateinit var copyButton: MaterialButton
    private lateinit var gdriveButton: MaterialButton
    private lateinit var gdriveHint: TextView
    private lateinit var downloadButton: MaterialButton
    private lateinit var confirmCheckbox: CheckBox
    private lateinit var continueButton: MaterialButton
    private lateinit var progress: CircularProgressIndicator

    private var mnemonic: String = ""
    private var walletId: String = ""
    private var chainId: Int = 1
    private var walletLabel: String = ""
    private var walletPassword: String = ""

    /** Launcher for the Google Sign-In intent (real GoogleAccountCredential). */
    private lateinit var googleSignInLauncher: ActivityResultLauncher<Intent>

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        arguments?.let {
            mnemonic = it.getString(ARG_MNEMONIC).orEmpty()
            walletId = it.getString(ARG_WALLET_ID).orEmpty()
            chainId = it.getInt(ARG_CHAIN_ID, 1)
            walletLabel = it.getString(ARG_LABEL).orEmpty()
            walletPassword = it.getString(ARG_PASSWORD).orEmpty()
        }
        googleSignInLauncher =
            registerForActivityResult(ActivityResultContracts.StartActivityForResult()) { result ->
                if (result.resultCode != android.app.Activity.RESULT_OK || result.data == null) {
                    Toast.makeText(
                        requireContext(),
                        "Google Sign-In canceled",
                        Toast.LENGTH_SHORT
                    ).show()
                    setBusy(false)
                    return@registerForActivityResult
                }
                val (account, err) = GoogleDriveBackup.accountFromResult(result.data)
                if (account == null) {
                    Toast.makeText(
                        requireContext(),
                        err ?: "Google Sign-In failed",
                        Toast.LENGTH_LONG
                    ).show()
                    setBusy(false)
                    return@registerForActivityResult
                }
                // Got a real account — encrypt locally, then upload via Drive v3.
                val ctx = requireContext()
                CoroutineScope(Dispatchers.IO).launch {
                    var localFile: java.io.File? = null
                    try {
                        val enc = EncryptedBackup.writeEncrypted(
                            ctx, walletId, walletPassword, mnemonic
                        )
                        localFile = enc.file
                        val res = GoogleDriveBackup.uploadBackup(
                            ctx, account, enc.file,
                            EncryptedBackup.backupFileName(walletId)
                        )
                        withContext(Dispatchers.Main) {
                            setBusy(false)
                            when (res) {
                                is GoogleDriveBackup.Result.Success ->
                                    Toast.makeText(
                                        requireContext(),
                                        "Backed up to Google Drive (${res.fileId})",
                                        Toast.LENGTH_LONG
                                    ).show()
                                is GoogleDriveBackup.Result.Failure ->
                                    Toast.makeText(
                                        requireContext(),
                                        res.message,
                                        Toast.LENGTH_LONG
                                    ).show()
                                is GoogleDriveBackup.Result.Canceled ->
                                    Toast.makeText(
                                        requireContext(),
                                        res.message,
                                        Toast.LENGTH_SHORT
                                    ).show()
                            }
                        }
                    } catch (e: Exception) {
                        withContext(Dispatchers.Main) {
                            setBusy(false)
                            Toast.makeText(
                                requireContext(),
                                e.message ?: "Google Drive backup failed",
                                Toast.LENGTH_LONG
                            ).show()
                        }
                    }
                }
            }
    }

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        return inflater.inflate(R.layout.fragment_backup_mnemonic, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        gridContainer = view.findViewById(R.id.mnemonicGrid)
        copyButton = view.findViewById(R.id.copyButton)
        gdriveButton = view.findViewById(R.id.gdriveButton)
        gdriveHint = view.findViewById(R.id.gdriveHint)
        downloadButton = view.findViewById(R.id.downloadButton)
        confirmCheckbox = view.findViewById(R.id.confirmCheckbox)
        continueButton = view.findViewById(R.id.continueButton)
        progress = view.findViewById(R.id.backupProgress)

        renderMnemonicGrid(mnemonic)

        copyButton.setOnClickListener {
            val cm = requireContext().getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
            cm.setPrimaryClip(ClipData.newPlainText("mnemonic", mnemonic))
            Toast.makeText(requireContext(), "Copied to clipboard", Toast.LENGTH_SHORT).show()
        }

        // Honest disable when no Google client ID configured — never fake success.
        if (!GoogleDriveBackup.isConfigured()) {
            gdriveButton.isEnabled = false
            gdriveButton.alpha = 0.5f
            gdriveHint.visibility = View.VISIBLE
            gdriveHint.text = getString(R.string.backup_gdrive_disabled)
        } else {
            gdriveHint.visibility = View.GONE
            gdriveButton.setOnClickListener { onGoogleDriveBackup() }
        }

        downloadButton.setOnClickListener { onDownloadBackup() }

        continueButton.isEnabled = false
        confirmCheckbox.setOnCheckedChangeListener { _, isChecked ->
            continueButton.isEnabled = isChecked
        }
        continueButton.setOnClickListener {
            (activity as? MainActivity)?.enterApp()
        }
    }

    private fun renderMnemonicGrid(mnemonic: String) {
        gridContainer.removeAllViews()
        val words = mnemonic.trim().split(Regex("\\s+")).filter { it.isNotEmpty() }
        // 3 columns × ceil(n/3) rows of numbered word chips.
        val columns = 3
        var row: LinearLayout? = null
        for ((i, word) in words.withIndex()) {
            if (i % columns == 0) {
                row = LinearLayout(requireContext()).apply {
                    orientation = LinearLayout.HORIZONTAL
                    layoutParams = LinearLayout.LayoutParams(
                        LinearLayout.LayoutParams.MATCH_PARENT,
                        LinearLayout.LayoutParams.WRAP_CONTENT
                    ).apply { bottomMargin = 8 }
                }
                gridContainer.addView(row)
            }
            val chip = layoutInflater.inflate(R.layout.item_mnemonic_word, row, false)
            chip.findViewById<TextView>(R.id.wordNumber).text = "${i + 1}."
            chip.findViewById<TextView>(R.id.wordText).text = word
            row?.addView(chip)
        }
    }

    private fun onGoogleDriveBackup() {
        if (walletPassword.isEmpty()) {
            Toast.makeText(requireContext(), "Password required to encrypt backup", Toast.LENGTH_SHORT).show()
            return
        }
        setBusy(true)
        // Kick off the real Google Sign-In intent (Play Services). The result
        // is handled in [googleSignInLauncher]'s callback, which encrypts the
        // mnemonic locally (AES-256-GCM) and uploads via Drive REST API v3.
        val intent = GoogleDriveBackup.signInClient(requireContext()).signInIntent
        googleSignInLauncher.launch(intent)
    }

    private fun onDownloadBackup() {
        if (walletPassword.isEmpty()) {
            Toast.makeText(requireContext(), "Password required to encrypt backup", Toast.LENGTH_SHORT).show()
            return
        }
        setBusy(true)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val enc = EncryptedBackup.writeEncrypted(
                    requireContext(), walletId, walletPassword, mnemonic
                )
                val shareIntent = EncryptedBackup.shareIntent(enc.uri)
                withContext(Dispatchers.Main) {
                    setBusy(false)
                    Toast.makeText(
                        requireContext(),
                        "Encrypted backup saved. Use share to store it safely.",
                        Toast.LENGTH_LONG
                    ).show()
                    startActivity(Intent.createChooser(shareIntent, "Save encrypted backup"))
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    setBusy(false)
                    Toast.makeText(
                        requireContext(),
                        e.message ?: "Backup failed",
                        Toast.LENGTH_LONG
                    ).show()
                }
            }
        }
    }

    private fun setBusy(busy: Boolean) {
        progress.visibility = if (busy) View.VISIBLE else View.GONE
        copyButton.isEnabled = !busy
        gdriveButton.isEnabled = !busy && GoogleDriveBackup.isConfigured()
        downloadButton.isEnabled = !busy
    }

    companion object {
        private const val ARG_MNEMONIC = "mnemonic"
        private const val ARG_WALLET_ID = "wallet_id"
        private const val ARG_CHAIN_ID = "chain_id"
        private const val ARG_LABEL = "label"
        private const val ARG_PASSWORD = "password"

        fun newInstance(walletId: String, mnemonic: String, chainId: Int, label: String, password: String) =
            BackupMnemonicFragment().apply {
                arguments = Bundle().apply {
                    putString(ARG_WALLET_ID, walletId)
                    putString(ARG_MNEMONIC, mnemonic)
                    putInt(ARG_CHAIN_ID, chainId)
                    putString(ARG_LABEL, label)
                    putString(ARG_PASSWORD, password)
                }
            }
    }
}
