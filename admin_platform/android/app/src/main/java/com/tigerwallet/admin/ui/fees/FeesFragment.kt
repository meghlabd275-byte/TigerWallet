package com.tigerwallet.admin.ui.fees

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.lifecycle.ViewModelProvider
import androidx.recyclerview.widget.LinearLayoutManager
import com.tigerwallet.admin.databinding.FragmentFeesBinding

class FeesFragment : Fragment() {
    private var _binding: FragmentFeesBinding? = null
    private val binding get() = _binding!!
    private lateinit var viewModel: FeesViewModel
    private lateinit var feesAdapter: FeesAdapter
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentFeesBinding.inflate(inflater, container, false)
        return binding.root
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        viewModel = ViewModelProvider(this)[FeesViewModel::class.java]
        
        setupRecyclerView()
        observeViewModel()
        
        viewModel.loadFees()
    }
    
    private fun setupRecyclerView() {
        feesAdapter = FeesAdapter { fee ->
            viewModel.toggleFeeActive(fee.id, !fee.isActive)
        }
        
        binding.rvFees.apply {
            layoutManager = LinearLayoutManager(context)
            adapter = feesAdapter
        }
        
        binding.swipeRefresh.setOnRefreshListener {
            viewModel.loadFees()
        }
    }
    
    private fun observeViewModel() {
        viewModel.fees.observe(viewLifecycleOwner) { fees ->
            binding.swipeRefresh.isRefreshing = false
            feesAdapter.submitList(fees)
            
            if (fees.isEmpty()) {
                binding.tvEmpty.visibility = View.VISIBLE
                binding.rvFees.visibility = View.GONE
            } else {
                binding.tvEmpty.visibility = View.GONE
                binding.rvFees.visibility = View.VISIBLE
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
