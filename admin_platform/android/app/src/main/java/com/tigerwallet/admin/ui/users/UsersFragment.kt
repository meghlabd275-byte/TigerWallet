package com.tigerwallet.admin.ui.users

import android.app.AlertDialog
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.lifecycle.ViewModelProvider
import androidx.recyclerview.widget.LinearLayoutManager
import com.tigerwallet.admin.databinding.FragmentUsersBinding

class UsersFragment : Fragment() {
    private var _binding: FragmentUsersBinding? = null
    private val binding get() = _binding!!
    private lateinit var viewModel: UsersViewModel
    private lateinit var usersAdapter: UsersAdapter
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentUsersBinding.inflate(inflater, container, false)
        return binding.root
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        viewModel = ViewModelProvider(this)[UsersViewModel::class.java]
        
        setupRecyclerView()
        setupSearch()
        observeViewModel()
        
        viewModel.loadUsers()
    }
    
    private fun setupRecyclerView() {
        usersAdapter = UsersAdapter(
            onUserClick = { user -> showUserDetails(user) },
            onSuspendClick = { user -> confirmSuspend(user.id, user.username) },
            onBanClick = { user -> confirmBan(user.id, user.username) }
        )
        
        binding.rvUsers.apply {
            layoutManager = LinearLayoutManager(context)
            adapter = usersAdapter
        }
        
        binding.swipeRefresh.setOnRefreshListener {
            viewModel.loadUsers()
        }
    }
    
    private fun setupSearch() {
        binding.btnSearch.setOnClickListener {
            val query = binding.etSearch.text.toString()
            val status = binding.spinnerStatus.selectedItem?.toString()
            viewModel.loadUsers(search = query, status = status)
        }
    }
    
    private fun observeViewModel() {
        viewModel.users.observe(viewLifecycleOwner) { users ->
            binding.swipeRefresh.isRefreshing = false
            usersAdapter.submitList(users)
            
            if (users.isEmpty()) {
                binding.tvEmpty.visibility = View.VISIBLE
                binding.rvUsers.visibility = View.GONE
            } else {
                binding.tvEmpty.visibility = View.GONE
                binding.rvUsers.visibility = View.VISIBLE
            }
        }
        
        viewModel.isLoading.observe(viewLifecycleOwner) { isLoading ->
            binding.swipeRefresh.isRefreshing = isLoading
        }
        
        viewModel.error.observe(viewLifecycleOwner) { error ->
            error?.let {
                Toast.makeText(context, it, Toast.LENGTH_LONG).show()
            }
        }
    }
    
    private fun showUserDetails(user: com.tigerwallet.admin.data.model.User) {
        // Show user details dialog
        val message = """
            Username: ${user.username}
            Email: ${user.email}
            Phone: ${user.phone ?: "N/A"}
            Status: ${user.status}
            Tier: ${user.tier}
            KYC Status: ${user.kycStatus}
            KYC Level: ${user.kycLevel}
            Email Verified: ${user.isEmailVerified}
            Phone Verified: ${user.isPhoneVerified}
            Created: ${user.createdAt}
            Last Login: ${user.lastLogin ?: "N/A"}
        """.trimIndent()
        
        AlertDialog.Builder(requireContext())
            .setTitle("User Details")
            .setMessage(message)
            .setPositiveButton("Close", null)
            .show()
    }
    
    private fun confirmSuspend(userId: String, username: String) {
        AlertDialog.Builder(requireContext())
            .setTitle("Suspend User")
            .setMessage("Are you sure you want to suspend $username?")
            .setPositiveButton("Suspend") { _, _ ->
                viewModel.suspendUser(userId, "Suspended by admin")
            }
            .setNegativeButton("Cancel", null)
            .show()
    }
    
    private fun confirmBan(userId: String, username: String) {
        AlertDialog.Builder(requireContext())
            .setTitle("Ban User")
            .setMessage("Are you sure you want to ban $username? This action cannot be undone.")
            .setPositiveButton("Ban") { _, _ ->
                viewModel.banUser(userId, "Banned by admin")
            }
            .setNegativeButton("Cancel", null)
            .show()
    }
    
    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
