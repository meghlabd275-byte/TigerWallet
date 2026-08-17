package com.tigeruserwallet.adapters

import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.TextView
import androidx.recyclerview.widget.RecyclerView
import com.tigeruserwallet.R
import org.json.JSONObject

/** Adapter for token-approval rows with a per-row Revoke action. */
class ApprovalsAdapter(
    private val items: MutableList<JSONObject>,
    private val onRevoke: (JSONObject) -> Unit
) : RecyclerView.Adapter<ApprovalsAdapter.VH>() {

    class VH(view: View) : RecyclerView.ViewHolder(view) {
        val spender: TextView = view.findViewById(R.id.approvalSpenderText)
        val token: TextView = view.findViewById(R.id.approvalTokenText)
        val amount: TextView = view.findViewById(R.id.approvalAmountText)
        val revoke: Button = view.findViewById(R.id.approvalRevokeButton)
    }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): VH {
        val v = LayoutInflater.from(parent.context).inflate(R.layout.item_approval, parent, false)
        return VH(v)
    }

    override fun onBindViewHolder(holder: VH, position: Int) {
        val a = items[position]
        holder.spender.text = "Spender: ${a.optString("spender", a.optString("contract", ""))}"
        holder.token.text = "Token: ${a.optString("token", a.optString("token_contract", ""))}"
        holder.amount.text = "Allowance: ${a.optString("amount", a.optString("allowance", "unlimited"))}"
        holder.revoke.setOnClickListener { onRevoke(a) }
    }

    override fun getItemCount(): Int = items.size

    fun update(newItems: List<JSONObject>) {
        items.clear()
        items.addAll(newItems)
        notifyDataSetChanged()
    }
}
