package com.tigeradmin.ui.fragments

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
import com.tigeradmin.R
import com.tigeradmin.data.model.KycRecord
import com.tigeradmin.data.repository.AdminRepository
import kotlinx.coroutines.launch

/**
 * KYC Fragment
 * Display and manage KYC verification requests
 */
class KYCFragment : Fragment() {
    
    private lateinit var recyclerView: RecyclerView
    private lateinit var progressIndicator: CircularProgressIndicator
    private lateinit var repository: AdminRepository
    
    private var kycRecords = mutableListOf<KycRecord>()
    private lateinit var adapter: KYCAdapter
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_kyc, container, false)
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        recyclerView = view.findViewById(R.id.kycRecyclerView)
        progressIndicator = view.findViewById(R.id.progressIndicator)
        
        repository = AdminRepository(requireContext())
        
        adapter = KYCAdapter(kycRecords,
            onApprove = { record -> approveKYC(record) },
            onReject = { record -> rejectKYC(record) }
        )
        
        recyclerView.layoutManager = LinearLayoutManager(requireContext())
        recyclerView.adapter = adapter
        
        loadKYCRecords()
    }
    
    private fun loadKYCRecords() {
        progressIndicator.visibility = View.VISIBLE
        
        viewLifecycleOwner.lifecycleScope.launch {
            try {
                val response = repository.getKYCRecords()
                if (response.isSuccessful) {
                    kycRecords.clear()
                    response.body()?.let { kycRecords.addAll(it) }
                    adapter.notifyDataSetChanged()
                } else {
                    Toast.makeText(requireContext(), "Failed to load KYC records", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Toast.makeText(requireContext(), "Error: ${e.message}", Toast.LENGTH_SHORT).show()
            } finally {
                progressIndicator.visibility = View.GONE
            }
        }
    }
    
    private fun approveKYC(record: KycRecord) {
        viewLifecycleOwner.lifecycleScope.launch {
            try {
                val response = repository.approveKYC(record.id)
                if (response.isSuccessful) {
                    Toast.makeText(requireContext(), "KYC approved", Toast.LENGTH_SHORT).show()
                    loadKYCRecords()
                } else {
                    Toast.makeText(requireContext(), "Failed to approve", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Toast.makeText(requireContext(), "Error: ${e.message}", Toast.LENGTH_SHORT).show()
            }
        }
    }
    
    private fun rejectKYC(record: KycRecord) {
        Toast.makeText(requireContext(), "Reject KYC: ${record.id}", Toast.LENGTH_SHORT).show()
    }
}

/**
 * KYC Adapter
 */
class KYCAdapter(
    private val records: List<KycRecord>,
    private val onApprove: (KycRecord) -> Unit,
    private val onReject: (KycRecord) -> Unit
) : RecyclerView.Adapter<KYCAdapter.ViewHolder>() {
    
    class ViewHolder(view: View) : RecyclerView.ViewHolder(view)
    
    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
        val view = LayoutInflater.from(parent.context)
            .inflate(R.layout.item_kyc, parent, false)
        return ViewHolder(view)
    }
    
    override fun onBindViewHolder(holder: ViewHolder, position: Int) {
        val record = records[position]
        // Bind data to views
    }
    
    override fun getItemCount(): Int = records.size
}
