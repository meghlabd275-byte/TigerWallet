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
import com.google.android.material.textfield.TextInputEditText
import com.tigerwallet.admin.R
import com.tigerwallet.admin.data.api.AdminApiService
import com.tigerwallet.admin.data.model.Transaction
import com.tigerwallet.admin.data.repository.AdminRepository
import kotlinx.coroutines.launch

/**
 * Transactions Fragment
 * Display and manage platform transactions
 */
class TransactionsFragment : Fragment() {
    
    private lateinit var recyclerView: RecyclerView
    private lateinit var progressIndicator: CircularProgressIndicator
    private lateinit var repository: AdminRepository
    
    private var transactions = mutableListOf<Transaction>()
    private lateinit var adapter: TransactionsAdapter
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_transactions, container, false)
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        recyclerView = view.findViewById(R.id.transactionsRecyclerView)
        progressIndicator = view.findViewById(R.id.progressIndicator)
        
        repository = AdminRepository(requireContext())
        
        adapter = TransactionsAdapter(transactions) { transaction ->
            // Handle transaction click
            showTransactionDetails(transaction)
        }
        
        recyclerView.layoutManager = LinearLayoutManager(requireContext())
        recyclerView.adapter = adapter
        
        loadTransactions()
    }
    
    private fun loadTransactions() {
        progressIndicator.visibility = View.VISIBLE
        
        viewLifecycleOwner.lifecycleScope.launch {
            try {
                val response = repository.getTransactions()
                if (response.isSuccessful) {
                    transactions.clear()
                    response.body()?.let { transactions.addAll(it) }
                    adapter.notifyDataSetChanged()
                } else {
                    Toast.makeText(requireContext(), "Failed to load transactions", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Toast.makeText(requireContext(), "Error: ${e.message}", Toast.LENGTH_SHORT).show()
            } finally {
                progressIndicator.visibility = View.GONE
            }
        }
    }
    
    private fun showTransactionDetails(transaction: Transaction) {
        // Show transaction details dialog
        Toast.makeText(requireContext(), "Transaction: ${transaction.hash}", Toast.LENGTH_SHORT).show()
    }
}

/**
 * Transactions Adapter
 */
class TransactionsAdapter(
    private val transactions: List<Transaction>,
    private val onItemClick: (Transaction) -> Unit
) : RecyclerView.Adapter<TransactionsAdapter.ViewHolder>() {
    
    class ViewHolder(view: View) : RecyclerView.ViewHolder(view)
    
    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
        val view = LayoutInflater.from(parent.context)
            .inflate(R.layout.item_transaction, parent, false)
        return ViewHolder(view)
    }
    
    override fun onBindViewHolder(holder: ViewHolder, position: Int) {
        val transaction = transactions[position]
        holder.itemView.setOnClickListener { onItemClick(transaction) }
    }
    
    override fun getItemCount(): Int = transactions.size
}
