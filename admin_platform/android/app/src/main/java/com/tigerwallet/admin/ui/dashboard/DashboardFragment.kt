package com.tigerwallet.admin.ui.dashboard

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.fragment.app.Fragment
import androidx.lifecycle.ViewModelProvider
import androidx.recyclerview.widget.GridLayoutManager
import com.tigerwallet.admin.databinding.FragmentDashboardBinding

class DashboardFragment : Fragment() {
    private var _binding: FragmentDashboardBinding? = null
    private val binding get() = _binding!!
    private lateinit var viewModel: DashboardViewModel
    private lateinit var statsAdapter: StatsAdapter
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentDashboardBinding.inflate(inflater, container, false)
        return binding.root
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        viewModel = ViewModelProvider(this)[DashboardViewModel::class.java]
        
        setupRecyclerView()
        observeViewModel()
        
        viewModel.loadDashboard()
    }
    
    private fun setupRecyclerView() {
        statsAdapter = StatsAdapter()
        binding.rvStats.apply {
            layoutManager = GridLayoutManager(context, 2)
            adapter = statsAdapter
        }
        
        binding.swipeRefresh.setOnRefreshListener {
            viewModel.loadDashboard()
        }
    }
    
    private fun observeViewModel() {
        viewModel.dashboardStats.observe(viewLifecycleOwner) { stats ->
            binding.swipeRefresh.isRefreshing = false
            stats?.let {
                binding.tvTotalUsers.text = it.totalUsers.toString()
                binding.tvActiveUsers.text = it.activeUsers.toString()
                binding.tvKycPending.text = it.kycPending.toString()
                binding.tvTotalTransactions.text = it.totalTransactions.toString()
                binding.tvVolume24h.text = "$${String.format("%.2f", it.volume24h)}"
                binding.tvRevenue24h.text = "$${String.format("%.2f", it.revenue24h)}"
                binding.tvNewUsers24h.text = it.newUsers24h.toString()
                binding.tvNewTransactions24h.text = it.newTransactions24h.toString()
            }
        }
        
        viewModel.isLoading.observe(viewLifecycleOwner) { isLoading ->
            binding.swipeRefresh.isRefreshing = isLoading
        }
        
        viewModel.error.observe(viewLifecycleOwner) { error ->
            error?.let {
                binding.tvError.text = it
                binding.tvError.visibility = View.VISIBLE
            } ?: run {
                binding.tvError.visibility = View.GONE
            }
        }
    }
    
    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
