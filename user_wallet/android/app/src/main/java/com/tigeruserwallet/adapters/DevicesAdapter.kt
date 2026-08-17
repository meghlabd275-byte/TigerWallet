package com.tigeruserwallet.adapters

import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.TextView
import androidx.recyclerview.widget.RecyclerView
import com.tigeruserwallet.R
import org.json.JSONObject

/** Adapter for device rows with per-row Sync and Delete actions. */
class DevicesAdapter(
    private val items: MutableList<JSONObject>,
    private val onSync: (JSONObject) -> Unit,
    private val onDelete: (JSONObject) -> Unit
) : RecyclerView.Adapter<DevicesAdapter.VH>() {

    class VH(view: View) : RecyclerView.ViewHolder(view) {
        val name: TextView = view.findViewById(R.id.deviceNameText)
        val type: TextView = view.findViewById(R.id.deviceTypeText)
        val sync: Button = view.findViewById(R.id.deviceSyncButton)
        val delete: Button = view.findViewById(R.id.deviceDeleteButton)
    }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): VH {
        val v = LayoutInflater.from(parent.context).inflate(R.layout.item_device, parent, false)
        return VH(v)
    }

    override fun onBindViewHolder(holder: VH, position: Int) {
        val d = items[position]
        holder.name.text = d.optString("name", "(unnamed)")
        holder.type.text = d.optString("device_type", d.optString("type", ""))
        holder.sync.setOnClickListener { onSync(d) }
        holder.delete.setOnClickListener { onDelete(d) }
    }

    override fun getItemCount(): Int = items.size

    fun update(newItems: List<JSONObject>) {
        items.clear()
        items.addAll(newItems)
        notifyDataSetChanged()
    }
}
