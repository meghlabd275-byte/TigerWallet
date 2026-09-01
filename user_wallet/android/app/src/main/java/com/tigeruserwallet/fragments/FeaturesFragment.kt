package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.LinearLayout
import androidx.fragment.app.Fragment
import com.tigeruserwallet.MainActivity
import com.tigeruserwallet.R

/**
 * Feature hub ("More" tab): navigates to every feature fragment so all
 * UserWallet functionality is reachable from the UI. One button per feature,
 * mirroring the web sidebar / desktop nav / extension tabs.
 */
class FeaturesFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_features, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val list = view.findViewById<LinearLayout>(R.id.featuresList)
        val features: List<Pair<String, () -> Fragment>> = listOf(
            "Send" to ::SendFragment,
            "Receive" to ::ReceiveFragment,
            "Swap" to ::SwapFragment,
            "Staking" to ::StakingFragment,
            "Bridge" to ::BridgeFragment,
            "NFTs" to ::NFTsFragment,
            "DeFi Hub" to ::DeFiFragment,
            "Fiat Ramp" to ::RampFragment,
            "Crypto Card" to ::CardsFragment,
            "P2P Trading" to ::P2PFragment,
            "Price Alerts" to ::PriceAlertsFragment,
            "Trading (Perp & Margin)" to ::TradingFragment,
            "DAO Governance" to ::DaoFragment,
            "Launchpool" to ::LaunchpoolFragment,
            "Token Sales" to ::TokenSalesFragment,
            "Prediction Markets" to ::PredictionFragment,
            "Copy Trading" to ::CopyTradingFragment,
            "Wallet & Finance" to ::FinanceFragment,
            "Fees" to ::FeesFragment,
            "ENS" to ::ENSFragment,
            "Security Center" to ::SecurityFragment,
            "Trading Terminal" to ::TerminalFragment,
            "Approvals" to ::ApprovalsFragment,
            "Address Book" to ::AddressBookFragment,
            "Devices" to ::DevicesFragment,
            "KYC" to ::KycFragment,
            "Keystore" to ::KeystoreFragment,
            "Multisig" to ::MultisigFragment,
            "Non-EVM Chains" to ::NonEvmFragment,
            "dApps & WalletConnect" to ::DAppsFragment,
        )
        features.forEach { (label, factory) ->
            val btn = Button(requireContext())
            btn.text = label
            btn.layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT
            )
            btn.setOnClickListener {
                (activity as? MainActivity)?.navigateToFeature(factory())
            }
            list.addView(btn)
        }
    }
}
