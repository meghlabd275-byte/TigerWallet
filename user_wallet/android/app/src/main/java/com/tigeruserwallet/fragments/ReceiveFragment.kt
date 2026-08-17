package com.tigeruserwallet.fragments

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import android.widget.Button
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
 * Receive screen: pick one of the caller's real wallets and expose its deposit
 * address with Copy/Share actions. No QR library is bundled in this build, so we
 * honestly show the address text + clipboard/share instead of faking a QR code.
 */
class ReceiveFragment : Fragment() {
    private lateinit var progressBar: ProgressBar
    private lateinit var walletSpinner: Spinner
    private lateinit var addressTextView: TextView
    private lateinit var copyButton: Button
    private lateinit var shareButton: Button

    private var wallets: List<UserWalletApiService.Wallet> = emptyList()

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_receive, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        progressBar = view.findViewById(R.id.receiveProgressBar)
        walletSpinner = view.findViewById(R.id.receiveWalletSpinner)
        addressTextView = view.findViewById(R.id.receiveAddressTextView)
        copyButton = view.findViewById(R.id.receiveCopyButton)
        shareButton = view.findViewById(R.id.receiveShareButton)

        copyButton.setOnClickListener { copySelectedAddress() }
        shareButton.setOnClickListener { shareSelectedAddress() }
        loadWallets()
    }

    private fun loadWallets() {
        setLoading(true)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                wallets = UserWalletApiService.getWallets()
                withContext(Dispatchers.Main) {
                    walletSpinner.adapter = ArrayAdapter(
                        requireContext(),
                        android.R.layout.simple_spinner_dropdown_item,
                        wallets.map { "${it.label} · Chain #${it.chainId}" }
                    )
                    renderSelected()
                    setLoading(false)
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    addressTextView.text = "[x] ${e.message ?: "Failed to load wallets"}"
                    setLoading(false)
                }
            }
        }
    }

    private fun selectedWallet(): UserWalletApiService.Wallet? =
        wallets.getOrNull(walletSpinner.selectedItemPosition)

    private fun renderSelected() {
        val w = selectedWallet()
        addressTextView.text = w?.address ?: "No wallet selected"
    }

    private fun copySelectedAddress() {
        val w = selectedWallet() ?: run {
            Toast.makeText(requireContext(), "Select a wallet", Toast.LENGTH_SHORT).show()
            return
        }
        val cm = requireContext().getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        cm.setPrimaryClip(ClipData.newPlainText("address", w.address))
        Toast.makeText(requireContext(), "Address copied", Toast.LENGTH_SHORT).show()
    }

    private fun shareSelectedAddress() {
        val w = selectedWallet() ?: run {
            Toast.makeText(requireContext(), "Select a wallet", Toast.LENGTH_SHORT).show()
            return
        }
        val intent = Intent(Intent.ACTION_SEND).apply {
            type = "text/plain"
            putExtra(Intent.EXTRA_TEXT, w.address)
        }
        startActivity(Intent.createChooser(intent, "Share address"))
    }

    private fun setLoading(loading: Boolean) {
        progressBar.visibility = if (loading) View.VISIBLE else View.GONE
    }
}
