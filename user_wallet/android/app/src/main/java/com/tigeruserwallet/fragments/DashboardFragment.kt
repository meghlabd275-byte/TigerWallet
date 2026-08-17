package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import androidx.fragment.app.Fragment
import androidx.recyclerview.widget.LinearLayoutManager
import com.tigeruserwallet.MainActivity
import com.tigeruserwallet.R
import com.tigeruserwallet.adapters.BalanceAdapter
import com.tigeruserwallet.api.UserWalletApiService
import com.tigeruserwallet.fragments.KycFragment
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

        val main = activity as? MainActivity
        view.findViewById<Button>(R.id.navSendButton).setOnClickListener {
            main?.navigateTo(SendFragment())
        }
        view.findViewById<Button>(R.id.navReceiveButton).setOnClickListener {
            main?.navigateTo(ReceiveFragment())
        }
        view.findViewById<Button>(R.id.navTransactionsButton).setOnClickListener {
            main?.navigateTo(TransactionsFragment())
        }
        view.findViewById<Button>(R.id.navSwapButton).setOnClickListener {
            main?.navigateTo(SwapFragment())
        }
        view.findViewById<Button>(R.id.navStakingButton).setOnClickListener {
            main?.navigateTo(StakingFragment())
        }
        view.findViewById<Button>(R.id.navNftsButton).setOnClickListener {
            main?.navigateTo(NFTsFragment())
        }
        view.findViewById<Button>(R.id.navBridgeButton).setOnClickListener {
            main?.navigateTo(BridgeFragment())
        }
        view.findViewById<Button>(R.id.navDeFiButton).setOnClickListener {
            main?.navigateTo(DeFiFragment())
        }
        view.findViewById<Button>(R.id.navKycButton).setOnClickListener {
            main?.navigateTo(KycFragment())
        }
        view.findViewById<Button>(R.id.navAddressBookButton).setOnClickListener {
            main?.navigateTo(AddressBookFragment())
        }
        view.findViewById<Button>(R.id.navApprovalsButton).setOnClickListener {
            main?.navigateTo(ApprovalsFragment())
        }
        view.findViewById<Button>(R.id.navDevicesButton).setOnClickListener {
            main?.navigateTo(DevicesFragment())
        }
        view.findViewById<Button>(R.id.navKeystoreButton).setOnClickListener {
            main?.navigateTo(KeystoreFragment())
        }
        view.findViewById<Button>(R.id.navWalletsButton).setOnClickListener {
            main?.navigateTo(WalletsFragment())
        }
        view.findViewById<Button>(R.id.navSettingsButton).setOnClickListener {
            main?.navigateTo(SettingsFragment())
        }

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
