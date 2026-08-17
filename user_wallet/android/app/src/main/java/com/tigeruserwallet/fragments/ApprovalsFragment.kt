package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import android.widget.Button
import android.widget.ProgressBar
import android.widget.Spinner
import android.widget.TextView
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.tigeruserwallet.R
import com.tigeruserwallet.adapters.ApprovalsAdapter
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Token approvals: pick a wallet, fetch [getApprovals] for its address/chain,
 * and revoke per row via [revokeApproval].
 */
class ApprovalsFragment : Fragment() {
    private lateinit var walletSpinner: Spinner
    private lateinit var refreshButton: Button
    private lateinit var progressBar: ProgressBar
    private lateinit var statusTextView: TextView
    private lateinit var recyclerView: RecyclerView

    private val adapter = ApprovalsAdapter(mutableListOf()) { revokeApproval(it) }
    private var wallets: List<UserWalletApiService.Wallet> = emptyList()

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_approvals, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        walletSpinner = view.findViewById(R.id.approvalsWalletSpinner)
        refreshButton = view.findViewById(R.id.approvalsRefreshButton)
        progressBar = view.findViewById(R.id.approvalsProgressBar)
        statusTextView = view.findViewById(R.id.approvalsStatusTextView)
        recyclerView = view.findViewById(R.id.approvalsRecyclerView)

        recyclerView.layoutManager = LinearLayoutManager(requireContext())
        recyclerView.adapter = adapter

        walletSpinner.onItemSelectedListener = object : android.widget.AdapterView.OnItemSelectedListener {
            override fun onItemSelected(p: android.widget.AdapterView<*>?, v: View?, pos: Int, id: Long) {
                loadApprovals()
            }
            override fun onNothingSelected(p: android.widget.AdapterView<*>?) {}
        }
        refreshButton.setOnClickListener { loadApprovals() }
        loadWallets()
    }

    private fun loadWallets() {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                wallets = UserWalletApiService.getWallets()
                withContext(Dispatchers.Main) {
                    walletSpinner.adapter = ArrayAdapter(
                        requireContext(),
                        android.R.layout.simple_spinner_dropdown_item,
                        wallets.map { "${it.label} \u00b7 ${it.address.take(10)}..." })
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Failed to load wallets"}"
                }
            }
        }
    }

    private fun loadApprovals() {
        val wallet = wallets.getOrNull(walletSpinner.selectedItemPosition) ?: return
        setLoading(true)
        statusTextView.text = "Loading approvals..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val approvals = UserWalletApiService.getApprovals(wallet.address, wallet.chainId)
                withContext(Dispatchers.Main) {
                    adapter.update(approvals)
                    statusTextView.text =
                        if (approvals.isEmpty()) "No token approvals found" else "${approvals.size} approvals"
                    setLoading(false)
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Failed to load approvals"}"
                    setLoading(false)
                }
            }
        }
    }

    private fun revokeApproval(approval: JSONObject) {
        val id = approval.optString("id", approval.optString("approval_id", ""))
        if (id.isEmpty()) {
            Toast.makeText(requireContext(), "No approval id found", Toast.LENGTH_SHORT).show()
            return
        }
        setLoading(true)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                UserWalletApiService.revokeApproval(id)
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), "\u2713 Approval revoked", Toast.LENGTH_SHORT).show()
                    loadApprovals()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Revoke failed"}"
                    setLoading(false)
                }
            }
        }
    }

    private fun setLoading(loading: Boolean) {
        progressBar.visibility = if (loading) View.VISIBLE else View.GONE
    }
}
