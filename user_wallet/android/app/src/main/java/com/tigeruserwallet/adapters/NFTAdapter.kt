package com.tigeruserwallet.adapters

import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.TextView
import androidx.recyclerview.widget.RecyclerView
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService

/**
 * Adapter for the NFT grid. Each row exposes its own Transfer action so the
 * fragment can open a destination prompt for that specific token.
 */
class NFTAdapter(
    private val items: MutableList<UserWalletApiService.NFT>,
    private val onTransfer: (UserWalletApiService.NFT) -> Unit
) : RecyclerView.Adapter<NFTAdapter.VH>() {

    class VH(view: View) : RecyclerView.ViewHolder(view) {
        val name: TextView = view.findViewById(R.id.nftNameText)
        val symbol: TextView = view.findViewById(R.id.nftSymbolText)
        val tokenId: TextView = view.findViewById(R.id.nftTokenIdText)
        val transfer: Button = view.findViewById(R.id.nftTransferButton)
    }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): VH {
        val v = LayoutInflater.from(parent.context).inflate(R.layout.item_nft, parent, false)
        return VH(v)
    }

    override fun onBindViewHolder(holder: VH, position: Int) {
        val nft = items[position]
        holder.name.text = nft.name.ifEmpty { "Unnamed NFT" }
        holder.symbol.text = nft.symbol.ifEmpty { nft.contractAddress.take(10) + "..." }
        holder.tokenId.text = "Token #${nft.tokenId}"
        holder.transfer.setOnClickListener { onTransfer(nft) }
    }

    override fun getItemCount(): Int = items.size

    fun update(newItems: List<UserWalletApiService.NFT>) {
        items.clear()
        items.addAll(newItems)
        notifyDataSetChanged()
    }
}
