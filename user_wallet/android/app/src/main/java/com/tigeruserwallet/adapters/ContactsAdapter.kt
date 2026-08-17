package com.tigeruserwallet.adapters

import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.TextView
import androidx.recyclerview.widget.RecyclerView
import com.tigeruserwallet.R
import org.json.JSONObject

/** Adapter for address-book rows. Surfaces delete per row and item selection. */
class ContactsAdapter(
    private val items: MutableList<JSONObject>,
    private val onDelete: (JSONObject) -> Unit,
    private var onItemClick: ((JSONObject) -> Unit)? = null
) : RecyclerView.Adapter<ContactsAdapter.VH>() {

    class VH(view: View) : RecyclerView.ViewHolder(view) {
        val name: TextView = view.findViewById(R.id.contactNameText)
        val address: TextView = view.findViewById(R.id.contactAddressText)
        val chain: TextView = view.findViewById(R.id.contactChainText)
        val delete: Button = view.findViewById(R.id.contactDeleteButton)
    }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): VH {
        val v = LayoutInflater.from(parent.context).inflate(R.layout.item_contact, parent, false)
        return VH(v)
    }

    override fun onBindViewHolder(holder: VH, position: Int) {
        val c = items[position]
        holder.name.text = c.optString("name", "(unnamed)")
        holder.address.text = c.optString("address", "")
        val cid = c.optInt("chain_id", -1)
        holder.chain.text = if (cid > 0) "Chain #$cid" else "Any chain"
        holder.delete.setOnClickListener { onDelete(c) }
        holder.itemView.setOnClickListener { onItemClick?.invoke(c) }
    }

    override fun getItemCount(): Int = items.size

    fun setOnItemClickListener(listener: (JSONObject) -> Unit) {
        onItemClick = listener
    }

    fun update(newItems: List<JSONObject>) {
        items.clear()
        items.addAll(newItems)
        notifyDataSetChanged()
    }
}
