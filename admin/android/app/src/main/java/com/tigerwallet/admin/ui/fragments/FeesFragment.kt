package com.tigerwallet.admin.ui.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import com.google.android.material.progressindicator.CircularProgressIndicator
import com.google.android.material.textfield.TextInputEditText
import com.tigerwallet.admin.R
import com.tigerwallet.admin.data.model.FeeConfig
import com.tigerwallet.admin.data.repository.AdminRepository
import kotlinx.coroutines.launch

/**
 * Fees Fragment
 * Configure platform fees
 */
class FeesFragment : Fragment() {
    
    private lateinit var progressIndicator: CircularProgressIndicator
    private lateinit var tradingFeeInput: TextInputEditText
    private lateinit var withdrawalFeeInput: TextInputEditText
    private lateinit var depositFeeInput: TextInputEditText
    private lateinit var makerFeeInput: TextInputEditText
    private lateinit var takerFeeInput: TextInputEditText
    
    private lateinit var repository: AdminRepository
    private var currentFeeConfig: FeeConfig? = null
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_fees, container, false)
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        progressIndicator = view.findViewById(R.id.progressIndicator)
        tradingFeeInput = view.findViewById(R.id.tradingFeeInput)
        withdrawalFeeInput = view.findViewById(R.id.withdrawalFeeInput)
        depositFeeInput = view.findViewById(R.id.depositFeeInput)
        makerFeeInput = view.findViewById(R.id.makerFeeInput)
        takerFeeInput = view.findViewById(R.id.takerFeeInput)
        
        repository = AdminRepository(requireContext())
        
        view.findViewById<View>(R.id.saveButton).setOnClickListener {
            saveFees()
        }
        
        loadFeeConfig()
    }
    
    private fun loadFeeConfig() {
        progressIndicator.visibility = View.VISIBLE
        
        viewLifecycleOwner.lifecycleScope.launch {
            try {
                val response = repository.getFeeConfig()
                if (response.isSuccessful) {
                    currentFeeConfig = response.body()
                    currentFeeConfig?.let { config ->
                        tradingFeeInput.setText(config.tradingFee)
                        withdrawalFeeInput.setText(config.withdrawalFee)
                        depositFeeInput.setText(config.depositFee)
                        makerFeeInput.setText(config.makerFee)
                        takerFeeInput.setText(config.takerFee)
                    }
                } else {
                    Toast.makeText(requireContext(), "Failed to load fees", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Toast.makeText(requireContext(), "Error: ${e.message}", Toast.LENGTH_SHORT).show()
            } finally {
                progressIndicator.visibility = View.GONE
            }
        }
    }
    
    private fun saveFees() {
        val feeConfig = FeeConfig(
            tradingFee = tradingFeeInput.text.toString(),
            withdrawalFee = withdrawalFeeInput.text.toString(),
            depositFee = depositFeeInput.text.toString(),
            makerFee = makerFeeInput.text.toString(),
            takerFee = takerFeeInput.text.toString()
        )
        
        progressIndicator.visibility = View.VISIBLE
        
        viewLifecycleOwner.lifecycleScope.launch {
            try {
                val response = repository.updateFeeConfig(feeConfig)
                if (response.isSuccessful) {
                    Toast.makeText(requireContext(), "Fees updated successfully", Toast.LENGTH_SHORT).show()
                } else {
                    Toast.makeText(requireContext(), "Failed to update fees", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Toast.makeText(requireContext(), "Error: ${e.message}", Toast.LENGTH_SHORT).show()
            } finally {
                progressIndicator.visibility = View.GONE
            }
        }
    }
}
