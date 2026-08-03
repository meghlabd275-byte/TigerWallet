package com.tigerwallet.admin.ui.chains

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.lifecycle.ViewModelProvider
import androidx.recyclerview.widget.LinearLayoutManager
import com.tigerwallet.admin.databinding.FragmentChainsBinding

class ChainsFragment : Fragment() {
    private var _binding: FragmentChainsBinding? = null
    private val binding get() = _binding!!
    private lateinit var viewModel: ChainsViewModel
    private lateinit var chainsAdapter: ChainsAdapter
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentChainsBinding.inflate(inflater, container, false)
        return binding.root
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        viewModel = ViewModelProvider(this)[ChainsViewModel::class.java]
        
        setupRecyclerView()
        observeViewModel()
        
        viewModel.loadChains()
    }
    
    private fun setupRecyclerView() {
        chainsAdapter = ChainsAdapter { chain ->
            viewModel.toggleChainActive(chain.id, !chain.isActive)
        }
        
        binding.rvChains.apply {
            layoutManager = LinearLayoutManager(context)
            adapter = chainsAdapter
        }
        
        binding.swipeRefresh.setOnRefreshListener {
            viewModel.loadChains()
        }
    }
    
    private fun observeViewModel() {
        viewModel.chains.observe(viewLifecycleOwner) { chains ->
            binding.swipeRefresh.isRefreshing = false
            chainsAdapter.submitList(chains)
            
            if (chains.isEmpty()) {
                binding.tvEmpty.visibility = View.VISIBLE
                binding.rvChains.visibility = View.GONE
            } else {
                binding.tvEmpty.visibility = View.GONE
                binding.rvChains.visibility = View.VISIBLE
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
