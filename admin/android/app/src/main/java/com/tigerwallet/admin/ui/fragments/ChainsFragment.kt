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
import com.google.android.material.floatingactionbutton.FloatingActionButton
import com.tigerwallet.admin.R
import com.tigerwallet.admin.data.model.Blockchain
import com.tigerwallet.admin.data.repository.AdminRepository
import kotlinx.coroutines.launch

/**
 * Chains Fragment
 * Manage blockchain configurations
 */
class ChainsFragment : Fragment() {
    
    private lateinit var recyclerView: RecyclerView
    private lateinit var progressIndicator: CircularProgressIndicator
    private lateinit var addButton: FloatingActionButton
    private lateinit var repository: AdminRepository
    
    private var blockchains = mutableListOf<Blockchain>()
    private lateinit var adapter: ChainsAdapter
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_chains, container, false)
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        recyclerView = view.findViewById(R.id.chainsRecyclerView)
        progressIndicator = view.findViewById(R.id.progressIndicator)
        addButton = view.findViewById(R.id.addChainButton)
        
        repository = AdminRepository(requireContext())
        
        adapter = ChainsAdapter(blockchains) { blockchain ->
            showChainActions(blockchain)
        }
        
        recyclerView.layoutManager = LinearLayoutManager(requireContext())
        recyclerView.adapter = adapter
        
        addButton.setOnClickListener {
            showAddChainDialog()
        }
        
        loadBlockchains()
    }
    
    private fun loadBlockchains() {
        progressIndicator.visibility = View.VISIBLE
        
        viewLifecycleOwner.lifecycleScope.launch {
            try {
                val response = repository.getBlockchains()
                if (response.isSuccessful) {
                    blockchains.clear()
                    response.body()?.let { blockchains.addAll(it) }
                    adapter.notifyDataSetChanged()
                } else {
                    Toast.makeText(requireContext(), "Failed to load blockchains", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Toast.makeText(requireContext(), "Error: ${e.message}", Toast.LENGTH_SHORT).show()
            } finally {
                progressIndicator.visibility = View.GONE
            }
        }
    }
    
    private fun showChainActions(blockchain: Blockchain) {
        Toast.makeText(requireContext(), "Chain: ${blockchain.name}", Toast.LENGTH_SHORT).show()
    }
    
    private fun showAddChainDialog() {
        Toast.makeText(requireContext(), "Add new blockchain", Toast.LENGTH_SHORT).show()
    }
}

/**
 * Chains Adapter
 */
class ChainsAdapter(
    private val blockchains: List<Blockchain>,
    private val onItemClick: (Blockchain) -> Unit
) : RecyclerView.Adapter<ChainsAdapter.ViewHolder>() {
    
    class ViewHolder(view: View) : RecyclerView.ViewHolder(view)
    
    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
        val view = LayoutInflater.from(parent.context)
            .inflate(R.layout.item_blockchain, parent, false)
        return ViewHolder(view)
    }
    
    override fun onBindViewHolder(holder: ViewHolder, position: Int) {
        val blockchain = blockchains[position]
        holder.itemView.setOnClickListener { onItemClick(blockchain) }
    }
    
    override fun getItemCount(): Int = blockchains.size
}
