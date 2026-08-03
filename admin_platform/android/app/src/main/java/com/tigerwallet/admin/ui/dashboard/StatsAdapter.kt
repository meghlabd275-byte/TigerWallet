package com.tigerwallet.admin.ui.dashboard

import android.view.LayoutInflater
import android.view.ViewGroup
import androidx.recyclerview.widget.DiffUtil
import androidx.recyclerview.widget.ListAdapter
import androidx.recyclerview.widget.RecyclerView
import com.tigerwallet.admin.data.model.StatItem
import com.tigerwallet.admin.databinding.ItemStatBinding

class StatsAdapter : ListAdapter<StatItem, StatsAdapter.StatViewHolder>(StatDiffCallback()) {
    
    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): StatViewHolder {
        val binding = ItemStatBinding.inflate(LayoutInflater.from(parent.context), parent, false)
        return StatViewHolder(binding)
    }
    
    override fun onBindViewHolder(holder: StatViewHolder, position: Int) {
        holder.bind(getItem(position))
    }
    
    class StatViewHolder(private val binding: ItemStatBinding) : RecyclerView.ViewHolder(binding.root) {
        fun bind(item: StatItem) {
            binding.tvTitle.text = item.title
            binding.tvValue.text = item.value
            binding.ivIcon.setImageResource(item.icon)
        }
    }
    
    class StatDiffCallback : DiffUtil.ItemCallback<StatItem>() {
        override fun areItemsTheSame(oldItem: StatItem, newItem: StatItem) = oldItem.title == newItem.title
        override fun areContentsTheSame(oldItem: StatItem, newItem: StatItem) = oldItem == newItem
    }
}
