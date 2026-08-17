package com.tigeruserwallet.fragments

import android.app.AlertDialog
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
import androidx.recyclerview.widget.GridLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.tigeruserwallet.R
import com.tigeruserwallet.adapters.NFTAdapter
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * NFT gallery: pick a wallet, fetch real owned tokens via
 * [UserWalletApiService.getNFTs] and expose a per-NFT Transfer action that opens
 * a destination prompt and broadcasts the real [transferNFT] transaction.
 */
class NFTsFragment : Fragment() {
    private lateinit var walletSpinner: Spinner
    private lateinit var refreshButton: Button
    private lateinit var progressBar: ProgressBar
    private lateinit var statusTextView: TextView
    private lateinit var recyclerView: RecyclerView

    private var wallets: List<UserWalletApiService.Wallet> = emptyList()
    private val adapter = NFTAdapter(mutableListOf()) { showTransferDialog(it) }

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_nfts, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        walletSpinner = view.findViewById(R.id.nftWalletSpinner)
        refreshButton = view.findViewById(R.id.nftRefreshButton)
        progressBar = view.findViewById(R.id.nftProgressBar)
        statusTextView = view.findViewById(R.id.nftStatusTextView)
        recyclerView = view.findViewById(R.id.nftRecyclerView)

        recyclerView.layoutManager = GridLayoutManager(requireContext(), 2)
        recyclerView.adapter = adapter

        refreshButton.setOnClickListener { loadNFTs() }
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
                        wallets.map { "${it.label} \u00b7 Chain #${it.chainId}" })
                    loadNFTs()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Failed to load wallets"}"
                }
            }
        }
    }

    private fun loadNFTs() {
        val wallet = wallets.getOrNull(walletSpinner.selectedItemPosition) ?: return
        setLoading(true)
        statusTextView.text = "Loading NFTs..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val nfts = UserWalletApiService.getNFTs(wallet.address, wallet.chainId)
                withContext(Dispatchers.Main) {
                    if (nfts.isEmpty()) {
                        statusTextView.text = "No NFTs owned by this wallet"
                    } else {
                        statusTextView.text = "${nfts.size} NFTs"
                    }
                    adapter.update(nfts)
                    setLoading(false)
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Failed to load NFTs"}"
                    adapter.update(emptyList())
                    setLoading(false)
                }
            }
        }
    }

    private fun showTransferDialog(nft: UserWalletApiService.NFT) {
        val wallet = wallets.getOrNull(walletSpinner.selectedItemPosition) ?: return
        val input = EditText(requireContext()).apply { hint = "Destination address" }
        val pass = EditText(requireContext()).apply {
            hint = "Wallet password"
            inputType = android.text.InputType.TYPE_CLASS_TEXT or android.text.InputType.TYPE_TEXT_VARIATION_PASSWORD
        }
        val container = android.widget.LinearLayout(requireContext()).apply {
            orientation = android.widget.LinearLayout.VERTICAL
            setPadding(48, 24, 48, 24)
            addView(input)
            addView(pass)
        }
        AlertDialog.Builder(requireContext())
            .setTitle("Transfer ${nft.name.ifEmpty { "NFT #${nft.tokenId}" }}")
            .setView(container)
            .setPositiveButton("Transfer") { _, _ ->
                val to = input.text.toString().trim()
                val password = pass.text.toString()
                if (to.isEmpty() || password.isEmpty()) {
                    Toast.makeText(requireContext(), "Enter destination and password", Toast.LENGTH_SHORT).show()
                    return@setPositiveButton
                }
                transferNFT(wallet, password, to, nft)
            }
            .setNegativeButton("Cancel", null)
            .show()
    }

    private fun transferNFT(
        wallet: UserWalletApiService.Wallet,
        password: String,
        to: String,
        nft: UserWalletApiService.NFT
    ) {
        statusTextView.text = "Transferring..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val res = UserWalletApiService.transferNFT(
                    wallet.id, password, to, nft.tokenId, nft.contractAddress, wallet.chainId
                )
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2713 ${res.optString("status", "Transfer sent")}"
                    Toast.makeText(
                        requireContext(),
                        "Transaction submitted to the blockchain network",
                        Toast.LENGTH_LONG
                    ).show()
                    loadNFTs()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Transfer failed"}"
                }
            }
        }
    }

    private fun setLoading(loading: Boolean) {
        progressBar.visibility = if (loading) View.VISIBLE else View.GONE
    }
}
