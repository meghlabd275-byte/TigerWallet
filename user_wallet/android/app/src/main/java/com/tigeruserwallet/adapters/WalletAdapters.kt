package com.tigeruserwallet.adapters

import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import androidx.recyclerview.widget.RecyclerView
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService

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
        h.usd.text = String.format("$%.2f", b.usdValue)
    }
}

class WalletAdapter(private val items: List<UserWalletApiService.Wallet>) :
    RecyclerView.Adapter<WalletAdapter.VH>() {
    class VH(v: View) : RecyclerView.ViewHolder(v) {
        val label: TextView = v.findViewById(R.id.walletLabel)
        val address: TextView = v.findViewById(R.id.walletAddress)
    }
    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int) =
        VH(LayoutInflater.from(parent.context).inflate(R.layout.item_wallet, parent, false))
    override fun getItemCount() = items.size
    override fun onBindViewHolder(h: VH, position: Int) {
        val w = items[position]
        h.label.text = w.label
        h.address.text = "Chain #${w.chainId} · ${w.address}"
    }
}

class TransactionAdapter(private val items: List<UserWalletApiService.Transaction>) :
    RecyclerView.Adapter<TransactionAdapter.VH>() {
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
        h.hash.text = t.hash.take(18) + "..."
        h.value.text = t.value
        h.status.text = if (t.isError == "0") "Success" else "Failed"
    }
}
