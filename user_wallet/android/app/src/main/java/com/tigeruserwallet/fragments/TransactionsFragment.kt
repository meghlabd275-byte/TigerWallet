package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.fragment.app.Fragment
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.tigeruserwallet.R
import com.tigeruserwallet.adapters.TransactionAdapter
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class TransactionsFragment : Fragment() {
    private lateinit var transactionsRecyclerView: RecyclerView

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_transactions, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        transactionsRecyclerView = view.findViewById(R.id.transactionsRecyclerView)
        transactionsRecyclerView.layoutManager = LinearLayoutManager(requireContext())
        loadTransactions()
    }

    private fun loadTransactions() {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val transactions = UserWalletApiService.getTransactions()
                withContext(Dispatchers.Main) {
                    transactionsRecyclerView.adapter = TransactionAdapter(transactions)
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    transactionsRecyclerView.adapter = TransactionAdapter(emptyList())
                }
            }
        }
    }
}
