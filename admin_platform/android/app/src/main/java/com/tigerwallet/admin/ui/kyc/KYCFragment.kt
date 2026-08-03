package com.tigerwallet.admin.ui.kyc

import android.app.AlertDialog
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.EditText
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.lifecycle.ViewModelProvider
import androidx.recyclerview.widget.LinearLayoutManager
import com.tigerwallet.admin.databinding.FragmentKycBinding

class KYCFragment : Fragment() {
    private var _binding: FragmentKycBinding? = null
    private val binding get() = _binding!!
    private lateinit var viewModel: KYCViewModel
    private lateinit var kycAdapter: KYCAdapter
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentKycBinding.inflate(inflater, container, false)
        return binding.root
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        viewModel = ViewModelProvider(this)[KYCViewModel::class.java]
        
        setupRecyclerView()
        setupFilters()
        observeViewModel()
        
        viewModel.loadSubmissions()
    }
    
    private fun setupRecyclerView() {
        kycAdapter = KYCAdapter(
            onApproveClick = { submission -> viewModel.approveKYC(submission.id, null) },
            onRejectClick = { submission -> showRejectDialog(submission.id) }
        )
        
        binding.rvKyc.apply {
            layoutManager = LinearLayoutManager(context)
            adapter = kycAdapter
        }
        
        binding.swipeRefresh.setOnRefreshListener {
            viewModel.loadSubmissions()
        }
    }
    
    private fun setupFilters() {
        binding.btnFilter.setOnClickListener {
            val status = binding.spinnerStatus.selectedItem?.toString()
            val level = binding.spinnerLevel.selectedItem?.toString()?.toIntOrNull()
            viewModel.loadSubmissions(status = status, level = level)
        }
    }
    
    private fun observeViewModel() {
        viewModel.submissions.observe(viewLifecycleOwner) { submissions ->
            binding.swipeRefresh.isRefreshing = false
            kycAdapter.submitList(submissions)
            
            if (submissions.isEmpty()) {
                binding.tvEmpty.visibility = View.VISIBLE
                binding.rvKyc.visibility = View.GONE
            } else {
                binding.tvEmpty.visibility = View.GONE
                binding.rvKyc.visibility = View.VISIBLE
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
    
    private fun showRejectDialog(submissionId: String) {
        val editText = EditText(requireContext())
        editText.hint = "Enter rejection reason"
        
        AlertDialog.Builder(requireContext())
            .setTitle("Reject KYC")
            .setView(editText)
            .setPositiveButton("Reject") { _, _ ->
                val reason = editText.text.toString()
                if (reason.isNotEmpty()) {
                    viewModel.rejectKYC(submissionId, reason)
                } else {
                    Toast.makeText(context, "Please enter a reason", Toast.LENGTH_SHORT).show()
                }
            }
            .setNegativeButton("Cancel", null)
            .show()
    }
    
    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
