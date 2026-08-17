package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import android.widget.Button
import android.widget.EditText
import android.widget.ProgressBar
import android.widget.Spinner
import android.widget.TextView
import android.widget.Toast
import androidx.fragment.app.Fragment
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject

/**
 * Identity verification (KYC) screen.
 *
 * Fetches the caller's real KYC status via [UserWalletApiService.getKycStatus]
 * (the backend resolves the user from the Bearer JWT when userId is null) and,
 * when not yet verified, walks the user through the real registration +
 * submission flow ([registerKyc] -> [submitKyc]). No mock data: every state
 * displayed here comes from the backend response.
 *
 * KYC is only required for P2P trading, surfaced as an inline note.
 */
class KycFragment : Fragment() {
    private lateinit var progressBar: ProgressBar
    private lateinit var statusTextView: TextView
    private lateinit var verifiedBanner: TextView
    private lateinit var fullNameInput: EditText
    private lateinit var documentTypeSpinner: Spinner
    private lateinit var documentNumberInput: EditText
    private lateinit var startKycButton: Button
    private lateinit var refreshKycButton: Button

    private val documentTypes = arrayOf("passport", "national_id", "drivers_license")

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_kyc, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        progressBar = view.findViewById(R.id.kycProgressBar)
        statusTextView = view.findViewById(R.id.kycStatusTextView)
        verifiedBanner = view.findViewById(R.id.kycVerifiedBanner)
        fullNameInput = view.findViewById(R.id.kycFullNameInput)
        documentTypeSpinner = view.findViewById(R.id.kycDocumentTypeSpinner)
        documentNumberInput = view.findViewById(R.id.kycDocumentNumberInput)
        startKycButton = view.findViewById(R.id.startKycButton)
        refreshKycButton = view.findViewById(R.id.refreshKycButton)

        documentTypeSpinner.adapter =
            ArrayAdapter(requireContext(), android.R.layout.simple_spinner_dropdown_item, documentTypes)

        startKycButton.setOnClickListener { submitKyc() }
        refreshKycButton.setOnClickListener { loadStatus() }

        loadStatus()
    }

    private fun loadStatus() {
        setLoading(true)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                // userId == null: backend resolves the caller from the Bearer JWT.
                val json = UserWalletApiService.getKycStatus(null)
                val status = json.optString("status", "not_submitted")
                withContext(Dispatchers.Main) {
                    renderStatus(status, json)
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "✗ ${e.message ?: "Failed to load KYC status"}"
                    showSubmission(false)
                    setLoading(false)
                }
            }
        }
    }

    private fun renderStatus(status: String, json: JSONObject) {
        when (status) {
            "verified", "approved" -> {
                statusTextView.text = "Status: verified"
                verifiedBanner.visibility = View.VISIBLE
                showSubmission(false)
            }
            "pending", "under_review" -> {
                statusTextView.text = "Status: pending review"
                verifiedBanner.visibility = View.GONE
                showSubmission(false)
            }
            "rejected", "declined" -> {
                val reason = json.optString("rejection_reason", "")
                statusTextView.text = "Status: rejected" +
                    if (reason.isNotEmpty()) "\nReason: $reason" else ""
                verifiedBanner.visibility = View.GONE
                showSubmission(true)
            }
            else -> {
                statusTextView.text = "Status: not submitted"
                verifiedBanner.visibility = View.GONE
                showSubmission(true)
            }
        }
        refreshKycButton.visibility = View.VISIBLE
        setLoading(false)
    }

    private fun submitKyc() {
        val fullName = fullNameInput.text.toString().trim()
        val documentType = documentTypes[documentTypeSpinner.selectedItemPosition]
        val documentNumber = documentNumberInput.text.toString().trim()

        if (fullName.isEmpty() || documentNumber.isEmpty()) {
            Toast.makeText(requireContext(), "Fill all fields", Toast.LENGTH_SHORT).show()
            return
        }

        startKycButton.isEnabled = false
        statusTextView.text = "Submitting KYC..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                // Step 1: begin the KYC registration session with the provider.
                UserWalletApiService.registerKyc(JSONObject())
                // Step 2: submit the actual identity payload.
                val body = JSONObject()
                    .put("full_name", fullName)
                    .put("document_type", documentType)
                    .put("document_number", documentNumber)
                UserWalletApiService.submitKyc(body)
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), "KYC submitted", Toast.LENGTH_SHORT).show()
                    loadStatus()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "✗ ${e.message ?: "KYC submission failed"}"
                    startKycButton.isEnabled = true
                }
            }
        }
    }

    private fun showSubmission(show: Boolean) {
        val v = if (show) View.VISIBLE else View.GONE
        fullNameInput.visibility = v
        documentTypeSpinner.visibility = v
        documentNumberInput.visibility = v
        startKycButton.visibility = v
    }

    private fun setLoading(loading: Boolean) {
        progressBar.visibility = if (loading) View.VISIBLE else View.GONE
    }
}
