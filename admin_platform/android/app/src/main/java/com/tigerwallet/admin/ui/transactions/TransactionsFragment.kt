package com.tigerwallet.admin.ui.transactions

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.lifecycle.ViewModelProvider
import androidx.recyclerview.widget.LinearLayoutManager
import com.tigerwallet.admin.databinding.FragmentTransactionsBinding

class TransactionsFragment : Fragment() {
    private var _binding: FragmentTransactionsBinding? = null
    private val binding get() = _binding!!
    private lateinit var viewModel: TransactionsViewModel
    private lateinit var transactionsAdapter: TransactionsAdapter
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentTransactionsBinding.inflate(inflater, container, false)
        return binding.root
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        viewModel = ViewModelProvider(this)[TransactionsViewModel::class.java]
        
        setupRecyclerView()
        setupFilters()
        observeViewModel()
        
        viewModel.loadTransactions()
    }
    
    private fun setupRecyclerView() {
        transactionsAdapter = TransactionsAdapter { transaction ->
            // Show transaction details
        }
        
        binding.rvTransactions.apply {
            layoutManager = LinearLayoutManager(context)
            adapter = transactionsAdapter
        }
        
        binding.swipeRefresh.setOnRefreshListener {
            viewModel.loadTransactions()
        }
    }
    
    private fun setupFilters() {
        binding.btnFilter.setOnClickListener {
            val type = binding.spinnerType.selectedItem?.toString()
            val status = binding.spinnerStatus.selectedItem?.toString()
            viewModel.loadTransactions(type = type, status = status)
        }
    }
    
    private fun observeViewModel() {
        viewModel.transactions.observe(viewLifecycleOwner) { transactions ->
            binding.swipeRefresh.isRefreshing = false
            transactionsAdapter.submitList(transactions)
            
            if (transactions.isEmpty()) {
                binding.tvEmpty.visibility = View.VISIBLE
                binding.rvTransactions.visibility = View.GONE
            } else {
                binding.tvEmpty.visibility = View.GONE
                binding.rvTransactions.visibility = View.VISIBLE
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
    
    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
