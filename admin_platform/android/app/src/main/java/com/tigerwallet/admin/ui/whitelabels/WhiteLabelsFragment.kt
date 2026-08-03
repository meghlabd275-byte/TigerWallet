package com.tigerwallet.admin.ui.whitelabels

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
import com.tigerwallet.admin.databinding.FragmentWhiteLabelsBinding

class WhiteLabelsFragment : Fragment() {
    private var _binding: FragmentWhiteLabelsBinding? = null
    private val binding get() = _binding!!
    private lateinit var viewModel: WhiteLabelsViewModel
    private lateinit var whiteLabelsAdapter: WhiteLabelsAdapter
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentWhiteLabelsBinding.inflate(inflater, container, false)
        return binding.root
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        viewModel = ViewModelProvider(this)[WhiteLabelsViewModel::class.java]
        
        setupRecyclerView()
        setupFilters()
        observeViewModel()
        
        viewModel.loadWhiteLabels()
    }
    
    private fun setupRecyclerView() {
        whiteLabelsAdapter = WhiteLabelsAdapter(
            onApproveClick = { wl -> viewModel.approveWhiteLabel(wl.id) },
            onSuspendClick = { wl -> showSuspendDialog(wl.id) }
        )
        
        binding.rvWhiteLabels.apply {
            layoutManager = LinearLayoutManager(context)
            adapter = whiteLabelsAdapter
        }
        
        binding.swipeRefresh.setOnRefreshListener {
            viewModel.loadWhiteLabels()
        }
    }
    
    private fun setupFilters() {
        binding.btnFilter.setOnClickListener {
            val status = binding.spinnerStatus.selectedItem?.toString()
            viewModel.loadWhiteLabels(status = status)
        }
    }
    
    private fun observeViewModel() {
        viewModel.whiteLabels.observe(viewLifecycleOwner) { whiteLabels ->
            binding.swipeRefresh.isRefreshing = false
            whiteLabelsAdapter.submitList(whiteLabels)
            
            if (whiteLabels.isEmpty()) {
                binding.tvEmpty.visibility = View.VISIBLE
                binding.rvWhiteLabels.visibility = View.GONE
            } else {
                binding.tvEmpty.visibility = View.GONE
                binding.rvWhiteLabels.visibility = View.VISIBLE
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
    
    private fun showSuspendDialog(whiteLabelId: String) {
        val editText = EditText(requireContext())
        editText.hint = "Enter suspension reason"
        
        AlertDialog.Builder(requireContext())
            .setTitle("Suspend White Label")
            .setView(editText)
            .setPositiveButton("Suspend") { _, _ ->
                val reason = editText.text.toString()
                if (reason.isNotEmpty()) {
                    viewModel.suspendWhiteLabel(whiteLabelId, reason)
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
