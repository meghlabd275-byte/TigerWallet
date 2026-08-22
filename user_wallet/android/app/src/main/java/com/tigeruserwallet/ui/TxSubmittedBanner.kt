package com.tigeruserwallet.ui

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.LinearLayout
import android.widget.TextView
import android.widget.Toast
import com.google.android.material.snackbar.Snackbar
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService

/**
 * TxSubmittedBanner — "Transaction submitted to the blockchain network"
 * banner shown after every successful send (mirrors web TxSubmittedBanner).
 *
 * Two real surfaces:
 *  - [showSnackbar] — a Material Snackbar (dismissable, with copy-hash action).
 *  - [makeBanner]   — a banner Card inserted into a LinearLayout container
 *    (persistent until the user dismisses it), with the real tx hash + a real
 *    explorer link (per chainId) that opens the system browser.
 *
 * No fabricated hashes/links: the hash comes from the backend's
 * transaction_hash; the explorer URL is the real block explorer for that
 * chainId (UserWalletApiService.explorerFor).
 */
object TxSubmittedBanner {

    fun showSnackbar(view: View, txHash: String, chainId: Int) {
        val msg = view.context.getString(R.string.tx_submitted_title) + "\n" + shortHash(txHash)
        Snackbar.make(view, msg, Snackbar.LENGTH_INDEFINITE).apply {
            setAction(R.string.tx_copy_hash) {
                copyToClipboard(view.context, txHash)
                Toast.makeText(view.context, "Hash copied", Toast.LENGTH_SHORT).show()
            }
            show()
        }
    }

    /**
     * Build a persistent banner card and add it to [container]. Includes the
     * full hash (monospace), the explorer link (if the chain has one), and a
     * copy-hash button. Tapping the explorer row opens the real URL.
     */
    fun makeBanner(container: LinearLayout, txHash: String, chainId: Int): View {
        val ctx = container.context
        val card = LayoutInflater.from(ctx)
            .inflate(R.layout.item_tx_banner, container, false) as com.google.android.material.card.MaterialCardView

        val title = card.findViewById<TextView>(R.id.txBannerTitle)
        val hashView = card.findViewById<TextView>(R.id.txBannerHash)
        val explorerRow = card.findViewById<View>(R.id.txBannerExplorerRow)
        val explorerText = card.findViewById<TextView>(R.id.txBannerExplorer)
        val copyBtn = card.findViewById<View>(R.id.txBannerCopy)
        val dismissBtn = card.findViewById<View>(R.id.txBannerDismiss)

        title.text = ctx.getString(R.string.tx_submitted_title)
        hashView.text = txHash

        val explorerBase = UserWalletApiService.explorerFor(chainId)
        if (explorerBase.isNotEmpty() && txHash.isNotEmpty()) {
            val url = explorerBase + txHash
            explorerText.text = url
            explorerRow.setOnClickListener {
                try {
                    ctx.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                } catch (e: Exception) {
                    Toast.makeText(ctx, "No browser available", Toast.LENGTH_SHORT).show()
                }
            }
        } else {
            explorerText.text = "Explorer unavailable for chain $chainId"
            explorerRow.setOnClickListener(null)
            explorerRow.alpha = 0.5f
        }

        copyBtn.setOnClickListener {
            copyToClipboard(ctx, txHash)
            Toast.makeText(ctx, "Hash copied", Toast.LENGTH_SHORT).show()
        }
        dismissBtn.setOnClickListener { container.removeView(card) }

        container.addView(card)
        return card
    }

    private fun shortHash(hash: String): String =
        if (hash.length <= 14) hash else "${hash.take(8)}…${hash.takeLast(6)}"

    private fun copyToClipboard(context: Context, text: String) {
        val cm = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        cm.setPrimaryClip(ClipData.newPlainText("tx_hash", text))
    }
}
