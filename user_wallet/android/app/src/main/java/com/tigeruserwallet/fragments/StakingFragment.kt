package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import android.widget.Button
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.ProgressBar
import android.widget.ScrollView
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

/**
 * Staking hub: fetches the real supported-asset list via
 * [UserWalletApiService.getStakingQuote] and exposes real stake/unstake/claim
 * actions against the backend /staking endpoints (which sign + broadcast via
 * /send). No mock assets are rendered.
 */
class StakingFragment : Fragment() {
    private lateinit var progressBar: ProgressBar
    private lateinit var statusTextView: TextView
    private lateinit var assetsContainer: LinearLayout
    private lateinit var walletSpinner: Spinner
    private lateinit var assetInput: EditText
    private lateinit var amountInput: EditText
    private lateinit var passwordInput: EditText
    private lateinit var stakeButton: Button
    private lateinit var unstakeButton: Button
    private lateinit var claimButton: Button
    private lateinit var resultTextView: TextView

    private var wallets: List<UserWalletApiService.Wallet> = emptyList()

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_staking, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        progressBar = view.findViewById(R.id.stakingProgressBar)
        statusTextView = view.findViewById(R.id.stakingStatusTextView)
        assetsContainer = view.findViewById(R.id.stakingAssetsContainer)
        walletSpinner = view.findViewById(R.id.stakingWalletSpinner)
        assetInput = view.findViewById(R.id.stakingAssetInput)
        amountInput = view.findViewById(R.id.stakingAmountInput)
        passwordInput = view.findViewById(R.id.stakingPasswordInput)
        stakeButton = view.findViewById(R.id.stakingStakeButton)
        unstakeButton = view.findViewById(R.id.stakingUnstakeButton)
        claimButton = view.findViewById(R.id.stakingClaimButton)
        resultTextView = view.findViewById(R.id.stakingResultTextView)

        stakeButton.setOnClickListener { performAction(Action.STAKE) }
        unstakeButton.setOnClickListener { performAction(Action.UNSTAKE) }
        claimButton.setOnClickListener { performAction(Action.CLAIM) }

        loadWallets()
        loadQuote()
    }

    private enum class Action { STAKE, UNSTAKE, CLAIM }

    private fun loadWallets() {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                wallets = UserWalletApiService.getWallets()
                withContext(Dispatchers.Main) {
                    walletSpinner.adapter = ArrayAdapter(
                        requireContext(),
                        android.R.layout.simple_spinner_dropdown_item,
                        wallets.map { "${it.label} \u00b7 Chain #${it.chainId}" })
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), "Failed to load wallets", Toast.LENGTH_SHORT).show()
                }
            }
        }
    }

    private fun loadQuote() {
        setLoading(true)
        statusTextView.text = "Loading staking markets..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val quote = UserWalletApiService.getStakingQuote(null)
                withContext(Dispatchers.Main) {
                    renderAssets(quote)
                    setLoading(false)
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    statusTextView.text = "\u2717 ${e.message ?: "Failed to load staking quote"}"
                    setLoading(false)
                }
            }
        }
    }

    private fun renderAssets(quote: UserWalletApiService.StakingQuote) {
        assetsContainer.removeAllViews()
        if (quote.assets.isEmpty()) {
            statusTextView.text = "No staking assets available"
            return
        }
        statusTextView.text = "APY ${quote.apy}  \u00b7  Min ${quote.minStake}"
        for (a in quote.assets) {
            val row = TextView(requireContext()).apply {
                text = "${a.symbol} \u00b7 Chain #${a.chainId} \u00b7 APY ${a.apy} \u00b7 Min ${a.minStake}" +
                    if (a.verified) " \u00b7 verified" else ""
                textSize = 14f
                setPadding(0, 12, 0, 12)
            }
            assetsContainer.addView(row)
        }
    }

    private fun performAction(action: Action) {
        val wallet = wallets.getOrNull(walletSpinner.selectedItemPosition) ?: run {
            Toast.makeText(requireContext(), "Select a wallet", Toast.LENGTH_SHORT).show()
            return
        }
        val asset = assetInput.text.toString().trim()
        val amount = amountInput.text.toString().trim()
        val password = passwordInput.text.toString()
        if (asset.isEmpty() || password.isEmpty()) {
            Toast.makeText(requireContext(), "Enter asset and password", Toast.LENGTH_SHORT).show()
            return
        }
        if (action != Action.CLAIM && amount.isEmpty()) {
            Toast.makeText(requireContext(), "Enter amount", Toast.LENGTH_SHORT).show()
            return
        }

        setButtonsEnabled(false)
        resultTextView.text = "Submitting..."
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val res = when (action) {
                    Action.STAKE -> UserWalletApiService.stake(wallet.id, password, asset, amount, wallet.chainId)
                    Action.UNSTAKE -> UserWalletApiService.unstake(wallet.id, password, asset, amount, wallet.chainId)
                    Action.CLAIM -> UserWalletApiService.claim(wallet.id, password, asset, wallet.chainId)
                }
                withContext(Dispatchers.Main) {
                    resultTextView.text = "\u2713 ${res.optString("status", res.toString())}"
                    Toast.makeText(
                        requireContext(),
                        "Transaction submitted to the blockchain network",
                        Toast.LENGTH_LONG
                    ).show()
                    setButtonsEnabled(true)
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    resultTextView.text = "\u2717 ${e.message ?: "Action failed"}"
                    setButtonsEnabled(true)
                }
            }
        }
    }

    private fun setButtonsEnabled(enabled: Boolean) {
        stakeButton.isEnabled = enabled
        unstakeButton.isEnabled = enabled
        claimButton.isEnabled = enabled
    }

    private fun setLoading(loading: Boolean) {
        progressBar.visibility = if (loading) View.VISIBLE else View.GONE
    }
}
