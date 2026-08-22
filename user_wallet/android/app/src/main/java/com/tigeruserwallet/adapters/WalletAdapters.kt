package com.tigeruserwallet.adapters

import android.content.Intent
import android.net.Uri
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import androidx.recyclerview.widget.RecyclerView
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService

/**
 * Adapters for the Dashboard / Wallets / Transactions lists.
 *
 * Mirrors the web tables exactly: real Balance/Wallet/Transaction fields
 * straight from [UserWalletApiService] (no fabricated values), block-explorer
 * links via [UserWalletApiService.explorerFor], and human-readable addresses.
 */
class BalanceAdapter(private val items: List<UserWalletApiService.Balance>) :
    RecyclerView.Adapter<BalanceAdapter.VH>() {

    class VH(v: View) : RecyclerView.ViewHolder(v) {
        val symbol: TextView = v.findViewById(R.id.balanceSymbol)
        val amount: TextView = v.findViewById(R.id.balanceAmount)
        val usd: TextView = v.findViewById(R.id.balanceUsd)
    }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int) =
        VH(LayoutInflater.from(parent.context).inflate(R.layout.item_balance, parent, false))

    override fun getItemCount() = items.size

    override fun onBindViewHolder(h: VH, position: Int) {
        val b = items[position]
        h.symbol.text = "${b.symbol} · Chain #${b.chainId}"
        h.amount.text = String.format("%.6f", b.balanceF)
        // No price feed is wired (mirror web: usdValue stays 0.0 -- honest,
        // never fabricated). Surface the address instead of a fake USD value.
        h.usd.text = h.itemView.context.getString(
            R.string.balance_address_fmt, shortAddr(b.address)
        )
    }

    private fun shortAddr(a: String): String =
        if (a.length > 16) "${a.take(10)}…${a.takeLast(6)}" else a
}

class WalletAdapter(
    private val items: List<UserWalletApiService.Wallet>,
    private val balances: Map<String, UserWalletApiService.Balance?> = emptyMap()
) : RecyclerView.Adapter<WalletAdapter.VH>() {

    class VH(v: View) : RecyclerView.ViewHolder(v) {
        val label: TextView = v.findViewById(R.id.walletLabel)
        val address: TextView = v.findViewById(R.id.walletAddress)
        val detail: TextView = v.findViewById(R.id.walletDetail)
    }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int) =
        VH(LayoutInflater.from(parent.context).inflate(R.layout.item_wallet, parent, false))

    override fun getItemCount() = items.size

    override fun onBindViewHolder(h: VH, position: Int) {
        val w = items[position]
        h.label.text = w.label.ifEmpty {
            h.itemView.context.getString(R.string.wallet_untitled)
        }
        val bal = balances[w.id]
        h.address.text = w.address
        h.detail.text = if (bal != null) {
            "Chain #${w.chainId} · ${bal.symbol} · " +
                String.format("%.6f", bal.balanceF)
        } else {
            "Chain #${w.chainId}"
        }
    }
}

class TransactionAdapter(
    private val items: List<UserWalletApiService.Transaction>
) : RecyclerView.Adapter<TransactionAdapter.VH>() {

    class VH(v: View) : RecyclerView.ViewHolder(v) {
        val hash: TextView = v.findViewById(R.id.txHash)
        val value: TextView = v.findViewById(R.id.txValue)
        val status: TextView = v.findViewById(R.id.txStatus)
    }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int) =
        VH(LayoutInflater.from(parent.context).inflate(R.layout.item_transaction, parent, false))

    override fun getItemCount() = items.size

    override fun onBindViewHolder(h: VH, position: Int) {
        val t = items[position]
        val hash = t.txHash
        h.hash.text = if (hash.isNotEmpty()) "${hash.take(14)}…" else "—"
        h.hash.setOnClickListener {
            val base = UserWalletApiService.explorerFor(t.chainId)
            if (base.isNotEmpty() && hash.isNotEmpty()) {
                val ctx = h.itemView.context
                ctx.startActivity(
                    Intent(Intent.ACTION_VIEW, Uri.parse(base + hash))
                        .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                )
            }
        }
        h.value.text = buildString {
            append(t.amount.ifEmpty { "—" })
            if (t.token.isNotEmpty()) append(" ").append(t.token)
        }
        h.status.text = t.status.ifEmpty { "unknown" }
    }
}
