package com.tigerwallet.admin.ui.users

import android.view.LayoutInflater
import android.view.ViewGroup
import androidx.recyclerview.widget.DiffUtil
import androidx.recyclerview.widget.ListAdapter
import androidx.recyclerview.widget.RecyclerView
import com.tigerwallet.admin.R
import com.tigerwallet.admin.data.model.User
import com.tigerwallet.admin.databinding.ItemUserBinding

class UsersAdapter(
    private val onUserClick: (User) -> Unit,
    private val onSuspendClick: (User) -> Unit,
    private val onBanClick: (User) -> Unit
) : ListAdapter<User, UsersAdapter.UserViewHolder>(UserDiffCallback()) {
    
    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): UserViewHolder {
        val binding = ItemUserBinding.inflate(LayoutInflater.from(parent.context), parent, false)
        return UserViewHolder(binding, onUserClick, onSuspendClick, onBanClick)
    }
    
    override fun onBindViewHolder(holder: UserViewHolder, position: Int) {
        holder.bind(getItem(position))
    }
    
    class UserViewHolder(
        private val binding: ItemUserBinding,
        private val onUserClick: (User) -> Unit,
        private val onSuspendClick: (User) -> Unit,
        private val onBanClick: (User) -> Unit
    ) : RecyclerView.ViewHolder(binding.root) {
        
        fun bind(user: User) {
            binding.tvUsername.text = user.username
            binding.tvEmail.text = user.email
            binding.tvStatus.text = user.status
            binding.tvKycStatus.text = "${user.kycStatus} (Lv.${user.kycLevel})"
            
            val statusColor = when (user.status) {
                "active" -> R.color.green
                "suspended" -> R.color.yellow
                "banned" -> R.color.red
                else -> R.color.gray
            }
            binding.tvStatus.setTextColor(binding.root.context.getColor(statusColor))
            
            binding.root.setOnClickListener { onUserClick(user) }
            
            if (user.status == "active") {
                binding.btnSuspend.visibility = android.view.View.VISIBLE
                binding.btnBan.visibility = android.view.View.VISIBLE
                binding.btnSuspend.setOnClickListener { onSuspendClick(user) }
                binding.btnBan.setOnClickListener { onBanClick(user) }
            } else {
                binding.btnSuspend.visibility = android.view.View.GONE
                binding.btnBan.visibility = android.view.View.GONE
            }
        }
    }
    
    class UserDiffCallback : DiffUtil.ItemCallback<User>() {
        override fun areItemsTheSame(oldItem: User, newItem: User) = oldItem.id == newItem.id
        override fun areContentsTheSame(oldItem: User, newItem: User) = oldItem == newItem
    }
}
