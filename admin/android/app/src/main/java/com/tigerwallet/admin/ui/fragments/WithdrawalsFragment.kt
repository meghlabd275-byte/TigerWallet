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
import com.tigerwallet.admin.data.model.Withdrawal
import com.tigerwallet.admin.data.repository.AdminRepository
import kotlinx.coroutines.launch

/**
 * Withdrawals Fragment
 * Display and manage withdrawal requests
 */
class WithdrawalsFragment : Fragment() {
    
    private lateinit var recyclerView: RecyclerView
    private lateinit var progressIndicator: CircularProgressIndicator
    private lateinit var repository: AdminRepository
    
    private var withdrawals = mutableListOf<Withdrawal>()
    private lateinit var adapter: WithdrawalsAdapter
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_withdrawals, container, false)
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        recyclerView = view.findViewById(R.id.withdrawalsRecyclerView)
        progressIndicator = view.findViewById(R.id.progressIndicator)
        
        repository = AdminRepository(requireContext())
        
        adapter = WithdrawalsAdapter(withdrawals,
            onApprove = { withdrawal -> approveWithdrawal(withdrawal) },
            onReject = { withdrawal -> rejectWithdrawal(withdrawal) }
        )
        
        recyclerView.layoutManager = LinearLayoutManager(requireContext())
        recyclerView.adapter = adapter
        
        loadWithdrawals()
    }
    
    private fun loadWithdrawals() {
        progressIndicator.visibility = View.VISIBLE
        
        viewLifecycleOwner.lifecycleScope.launch {
            try {
                val response = repository.getWithdrawals()
                if (response.isSuccessful) {
                    withdrawals.clear()
                    response.body()?.let { withdrawals.addAll(it) }
                    adapter.notifyDataSetChanged()
                } else {
                    Toast.makeText(requireContext(), "Failed to load withdrawals", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Toast.makeText(requireContext(), "Error: ${e.message}", Toast.LENGTH_SHORT).show()
            } finally {
                progressIndicator.visibility = View.GONE
            }
        }
    }
    
    private fun approveWithdrawal(withdrawal: Withdrawal) {
        viewLifecycleOwner.lifecycleScope.launch {
            try {
                val response = repository.approveWithdrawal(withdrawal.id)
                if (response.isSuccessful) {
                    Toast.makeText(requireContext(), "Withdrawal approved", Toast.LENGTH_SHORT).show()
                    loadWithdrawals()
                } else {
                    Toast.makeText(requireContext(), "Failed to approve", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Toast.makeText(requireContext(), "Error: ${e.message}", Toast.LENGTH_SHORT).show()
            }
        }
    }
    
    private fun rejectWithdrawal(withdrawal: Withdrawal) {
        Toast.makeText(requireContext(), "Reject: ${withdrawal.id}", Toast.LENGTH_SHORT).show()
    }
}

/**
 * Withdrawals Adapter
 */
class WithdrawalsAdapter(
    private val withdrawals: List<Withdrawal>,
    private val onApprove: (Withdrawal) -> Unit,
    private val onReject: (Withdrawal) -> Unit
) : RecyclerView.Adapter<WithdrawalsAdapter.ViewHolder>() {
    
    class ViewHolder(view: View) : RecyclerView.ViewHolder(view)
    
    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
        val view = LayoutInflater.from(parent.context)
            .inflate(R.layout.item_withdrawal, parent, false)
        return ViewHolder(view)
    }
    
    override fun onBindViewHolder(holder: ViewHolder, position: Int) {
        val withdrawal = withdrawals[position]
        // Bind data to views
    }
    
    override fun getItemCount(): Int = withdrawals.size
}
