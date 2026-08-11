package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.fragment.app.Fragment
import androidx.recyclerview.widget.LinearLayoutManager
import com.tigeruserwallet.R
import com.tigeruserwallet.adapters.BalanceAdapter
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class DashboardFragment : Fragment() {
    private lateinit var balancesRecyclerView: androidx.recyclerview.widget.RecyclerView

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
                val balances = UserWalletApiService.getBalances()
                withContext(Dispatchers.Main) {
                    balancesRecyclerView.adapter = BalanceAdapter(balances)
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    balancesRecyclerView.adapter = BalanceAdapter(emptyList())
                }
            }
        }
    }
}
