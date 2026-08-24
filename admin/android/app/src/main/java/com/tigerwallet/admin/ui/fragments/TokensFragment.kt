package com.tigerwallet.admin.ui.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.google.android.material.progressindicator.CircularProgressIndicator
import com.tigerwallet.admin.R
import com.tigerwallet.admin.data.model.Token
import com.tigerwallet.admin.data.repository.AdminRepository
import kotlinx.coroutines.launch

/**
 * Tokens Fragment
 * Display and manage platform tokens
 */
class TokensFragment : Fragment() {
    
    private lateinit var recyclerView: RecyclerView
    private lateinit var progressIndicator: CircularProgressIndicator
    private lateinit var repository: AdminRepository
    
    private var tokens = mutableListOf<Token>()
    private lateinit var adapter: TokensAdapter
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_tokens, container, false)
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        recyclerView = view.findViewById(R.id.tokensRecyclerView)
        progressIndicator = view.findViewById(R.id.progressIndicator)
        
        repository = AdminRepository(requireContext())
        
        adapter = TokensAdapter(tokens) { token ->
            showTokenActions(token)
        }
        
        recyclerView.layoutManager = LinearLayoutManager(requireContext())
        recyclerView.adapter = adapter
        
        loadTokens()
    }
    
    private fun loadTokens() {
        progressIndicator.visibility = View.VISIBLE
        
        viewLifecycleOwner.lifecycleScope.launch {
            try {
                val response = repository.getTokens()
                if (response.isSuccessful) {
                    tokens.clear()
                    response.body()?.let { tokens.addAll(it) }
                    adapter.notifyDataSetChanged()
                } else {
                    Toast.makeText(requireContext(), "Failed to load tokens", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Toast.makeText(requireContext(), "Error: ${e.message}", Toast.LENGTH_SHORT).show()
            } finally {
                progressIndicator.visibility = View.GONE
            }
        }
    }
    
    private fun showTokenActions(token: Token) {
        Toast.makeText(requireContext(), "Token: ${token.name}", Toast.LENGTH_SHORT).show()
    }
}

/**
 * Tokens Adapter
 */
class TokensAdapter(
    private val tokens: List<Token>,
    private val onItemClick: (Token) -> Unit
) : RecyclerView.Adapter<TokensAdapter.ViewHolder>() {
    
    class ViewHolder(view: View) : RecyclerView.ViewHolder(view)
    
    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
        val view = LayoutInflater.from(parent.context)
            .inflate(R.layout.item_token, parent, false)
        return ViewHolder(view)
    }
    
    override fun onBindViewHolder(holder: ViewHolder, position: Int) {
        val token = tokens[position]
        holder.itemView.setOnClickListener { onItemClick(token) }
    }
    
    override fun getItemCount(): Int = tokens.size
}
