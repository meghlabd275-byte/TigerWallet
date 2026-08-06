package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
import androidx.fragment.app.Fragment
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class WalletsFragment : Fragment() {
    private val apiService = UserWalletApiService()
    private lateinit var walletsRecyclerView: RecyclerView
    private lateinit var addWalletButton: Button

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_wallets, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        walletsRecyclerView = view.findViewById(R.id.walletsRecyclerView)
        addWalletButton = view.findViewById(R.id.addWalletButton)
        
        walletsRecyclerView.layoutManager = LinearLayoutManager(requireContext())
        
        addWalletButton.setOnClickListener {
            showAddWalletDialog()
        }
        
        loadWallets()
    }

    private fun loadWallets() {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val wallets = apiService.getWallets()
                withContext(Dispatchers.Main) {
                    // Update adapter
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), "Failed to load wallets", Toast.LENGTH_SHORT).show()
                }
            }
        }
    }

    private fun showAddWalletDialog() {
        val dialogView = layoutInflater.inflate(R.layout.dialog_add_wallet, null)
        val nameInput = dialogView.findViewById<EditText>(R.id.walletNameInput)
        
        AlertDialog.Builder(requireContext())
            .setTitle("Create Wallet")
            .setView(dialogView)
            .setPositiveButton("Create") { _, _ ->
                val name = nameInput.text.toString()
                createWallet(name)
            }
            .setNegativeButton("Cancel", null)
            .show()
    }

    private fun createWallet(name: String) {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                apiService.createWallet(name, "ethereum", listOf("ethereum"))
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), "Wallet created", Toast.LENGTH_SHORT).show()
                    loadWallets()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), "Failed to create wallet", Toast.LENGTH_SHORT).show()
                }
            }
        }
    }
}
