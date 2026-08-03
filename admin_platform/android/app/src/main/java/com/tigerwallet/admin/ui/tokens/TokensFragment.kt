package com.tigerwallet.admin.ui.tokens

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.lifecycle.ViewModelProvider
import androidx.recyclerview.widget.LinearLayoutManager
import com.tigerwallet.admin.databinding.FragmentTokensBinding

class TokensFragment : Fragment() {
    private var _binding: FragmentTokensBinding? = null
    private val binding get() = _binding!!
    private lateinit var viewModel: TokensViewModel
    private lateinit var tokensAdapter: TokensAdapter
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentTokensBinding.inflate(inflater, container, false)
        return binding.root
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        viewModel = ViewModelProvider(this)[TokensViewModel::class.java]
        
        setupRecyclerView()
        observeViewModel()
        
        viewModel.loadTokens()
    }
    
    private fun setupRecyclerView() {
        tokensAdapter = TokensAdapter(
            onVerifyClick = { token -> viewModel.verifyToken(token.id) },
            onDeleteClick = { token -> viewModel.deleteToken(token.id) }
        )
        
        binding.rvTokens.apply {
            layoutManager = LinearLayoutManager(context)
            adapter = tokensAdapter
        }
        
        binding.swipeRefresh.setOnRefreshListener {
            viewModel.loadTokens()
        }
    }
    
    private fun observeViewModel() {
        viewModel.tokens.observe(viewLifecycleOwner) { tokens ->
            binding.swipeRefresh.isRefreshing = false
            tokensAdapter.submitList(tokens)
            
            if (tokens.isEmpty()) {
                binding.tvEmpty.visibility = View.VISIBLE
                binding.rvTokens.visibility = View.GONE
            } else {
                binding.tvEmpty.visibility = View.GONE
                binding.rvTokens.visibility = View.VISIBLE
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
