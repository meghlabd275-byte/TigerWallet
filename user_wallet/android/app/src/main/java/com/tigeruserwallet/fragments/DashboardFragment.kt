package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import androidx.fragment.app.Fragment
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class DashboardFragment : Fragment() {
    private val apiService = UserWalletApiService()
    private lateinit var balancesRecyclerView: RecyclerView

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_dashboard, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        balancesRecyclerView = view.findViewById(R.id.balancesRecyclerView)
        balancesRecyclerView.layoutManager = LinearLayoutManager(requireContext())
        loadBalances()
    }

    private fun loadBalances() {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val balances = apiService.getBalances()
                withContext(Dispatchers.Main) {
                    // Update UI with balances
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    // Show error
                }
            }
        }
    }
}
