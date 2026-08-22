package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import androidx.appcompat.widget.AppCompatSpinner
import androidx.fragment.app.Fragment
import androidx.recyclerview.widget.LinearLayoutManager
import com.google.android.material.button.MaterialButton
import com.google.android.material.progressindicator.CircularProgressIndicator
import com.google.android.material.textfield.TextInputEditText
import com.google.android.material.textfield.TextInputLayout
import com.tigeruserwallet.R
import com.tigeruserwallet.adapters.WalletAdapter
import com.tigeruserwallet.api.UserWalletApiService
import com.tigeruserwallet.databinding.FragmentWalletsBinding
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Wallets (mirrors web Wallets.tsx):
 *  - list of local wallets with a per-wallet balance fan-out
 *  - inline "Create Wallet" form -> createWalletTyped(label, password, chainId)
 *    -> BackupMnemonicFragment (the mnemonic is shown only once)
 *
 * No stubs: real backend fetches; the create form reuses the same
 * createWalletTyped path as the onboarding Create flow.
 */
class WalletsFragment : Fragment() {

    private var _binding: FragmentWalletsBinding? = null
    private val binding get() = _binding!!

    private lateinit var nameInput: TextInputEditText
    private lateinit var nameLayout: TextInputLayout
    private lateinit var networkSpinner: AppCompatSpinner
    private lateinit var passwordInput: TextInputEditText
    private lateinit var passwordLayout: TextInputLayout
    private lateinit var createButton: MaterialButton
    private lateinit var createProgress: CircularProgressIndicator

    private var showCreate = false
    private var wallets: List<UserWalletApiService.Wallet> = emptyList()
    private var balances: Map<String, UserWalletApiService.Balance?> = emptyMap()

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentWalletsBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        nameInput = binding.nameInput
        nameLayout = binding.nameLayout
        networkSpinner = binding.networkSpinner
        passwordInput = binding.passwordInput
        passwordLayout = binding.passwordLayout
        createButton = binding.createButton
        createProgress = binding.createProgress

        networkSpinner.adapter = ArrayAdapter(
            requireContext(),
            android.R.layout.simple_spinner_dropdown_item,
            UserWalletApiService.CHAINS.map { "${it.name} (${it.symbol})" }
        )
        createButton.setOnClickListener { onCreateSubmit() }
        binding.addWalletButton.setOnClickListener { toggleCreateForm() }
        binding.walletsRecyclerView.layoutManager = LinearLayoutManager(requireContext())
        binding.swipeRefresh.setOnRefreshListener { loadWallets() }
        applyCreateVisibility()
        loadWallets()
    }

    private fun toggleCreateForm() {
        showCreate = !showCreate
        applyCreateVisibility()
    }

    private fun applyCreateVisibility() {
        binding.createFormCard.visibility = if (showCreate) View.VISIBLE else View.GONE
        binding.addWalletButton.text = getString(
            if (showCreate) R.string.wallets_cancel else R.string.wallets_add
        )
    }

    private fun onCreateSubmit() {
        nameLayout.error = null
        passwordLayout.error = null
        val label = nameInput.text?.toString().orEmpty().trim()
        val password = passwordInput.text?.toString().orEmpty()
        val chainId = UserWalletApiService.CHAINS[networkSpinner.selectedItemPosition].id
        if (label.isEmpty()) { nameLayout.error = getString(R.string.err_name); return }
        if (password.length < 8) {
            passwordLayout.error = getString(R.string.err_password_short)
            return
        }
        setBusy(true)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                UserWalletApiService.ensureSession()
                val wallet = UserWalletApiService.createWalletTyped(label, password, chainId)
                UserWalletApiService.rememberWallet(wallet.id)
                withContext(Dispatchers.Main) {
                    setBusy(false)
                    if (wallet.mnemonic != null) {
                        // Show backup with the real mnemonic before it's gone.
                        parentFragmentManager.beginTransaction()
                            .replace(
                                R.id.fragmentContainer,
                                BackupMnemonicFragment.newInstance(
                                    wallet.id,
                                    wallet.mnemonic,
                                    wallet.chainId,
                                    wallet.label,
                                    password
                                )
                            )
                            .addToBackStack(null)
                            .commit()
                    } else {
                        loadWallets()
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    setBusy(false)
                    passwordLayout.error = e.message ?: getString(R.string.err_create_failed)
                }
            }
        }
    }

    private fun loadWallets() {
        binding.swipeRefresh.isRefreshing = true
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val ws = UserWalletApiService.getWallets()
                val balMap = mutableMapOf<String, UserWalletApiService.Balance?>()
                ws.forEach { w ->
                    balMap[w.id] = try {
                        UserWalletApiService.getBalance(w.id)
                    } catch (e: Exception) {
                        null
                    }
                }
                wallets = ws
                balances = balMap
                withContext(Dispatchers.Main) {
                    binding.walletsRecyclerView.adapter = WalletAdapter(ws, balMap)
                    binding.emptyState.visibility =
                        if (ws.isEmpty()) View.VISIBLE else View.GONE
                    binding.walletsRecyclerView.visibility =
                        if (ws.isEmpty()) View.GONE else View.VISIBLE
                    binding.swipeRefresh.isRefreshing = false
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    binding.walletsRecyclerView.adapter = WalletAdapter(emptyList())
                    binding.emptyState.visibility = View.VISIBLE
                    binding.walletsRecyclerView.visibility = View.GONE
                    binding.swipeRefresh.isRefreshing = false
                }
            }
        }
    }

    private fun setBusy(busy: Boolean) {
        createProgress.visibility = if (busy) View.VISIBLE else View.GONE
        createButton.isEnabled = !busy
    }

    override fun onResume() {
        super.onResume()
        // Refresh after returning from the backup screen.
        loadWallets()
    }

    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
