package com.tigerwallet.admin.ui.withdrawals

import android.app.AlertDialog
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.EditText
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.lifecycle.ViewModelProvider
import androidx.recyclerview.widget.LinearLayoutManager
import com.tigerwallet.admin.databinding.FragmentWithdrawalsBinding

class WithdrawalsFragment : Fragment() {
    private var _binding: FragmentWithdrawalsBinding? = null
    private val binding get() = _binding!!
    private lateinit var viewModel: WithdrawalsViewModel
    private lateinit var withdrawalsAdapter: WithdrawalsAdapter
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentWithdrawalsBinding.inflate(inflater, container, false)
        return binding.root
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        viewModel = ViewModelProvider(this)[WithdrawalsViewModel::class.java]
        
        setupRecyclerView()
        setupFilters()
        observeViewModel()
        
        viewModel.loadWithdrawals()
    }
    
    private fun setupRecyclerView() {
        withdrawalsAdapter = WithdrawalsAdapter(
            onApproveClick = { withdrawal -> viewModel.approveWithdrawal(withdrawal.id) },
            onRejectClick = { withdrawal -> showRejectDialog(withdrawal.id) }
        )
        
        binding.rvWithdrawals.apply {
            layoutManager = LinearLayoutManager(context)
            adapter = withdrawalsAdapter
        }
        
        binding.swipeRefresh.setOnRefreshListener {
            viewModel.loadWithdrawals()
        }
    }
    
    private fun setupFilters() {
        binding.btnFilter.setOnClickListener {
            val status = binding.spinnerStatus.selectedItem?.toString()
            viewModel.loadWithdrawals(status = status)
        }
    }
    
    private fun observeViewModel() {
        viewModel.withdrawals.observe(viewLifecycleOwner) { withdrawals ->
            binding.swipeRefresh.isRefreshing = false
            withdrawalsAdapter.submitList(withdrawals)
            
            if (withdrawals.isEmpty()) {
                binding.tvEmpty.visibility = View.VISIBLE
                binding.rvWithdrawals.visibility = View.GONE
            } else {
                binding.tvEmpty.visibility = View.GONE
                binding.rvWithdrawals.visibility = View.VISIBLE
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
    
    private fun showRejectDialog(withdrawalId: String) {
        val editText = EditText(requireContext())
        editText.hint = "Enter rejection reason"
        
        AlertDialog.Builder(requireContext())
            .setTitle("Reject Withdrawal")
            .setView(editText)
            .setPositiveButton("Reject") { _, _ ->
                val reason = editText.text.toString()
                if (reason.isNotEmpty()) {
                    viewModel.rejectWithdrawal(withdrawalId, reason)
                } else {
                    Toast.makeText(context, "Please enter a reason", Toast.LENGTH_SHORT).show()
                }
            }
            .setNegativeButton("Cancel", null)
            .show()
    }
    
    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
