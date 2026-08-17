package com.tigeruserwallet.fragments

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
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject

/**
 * DeFi hub. One fragment mirroring the web's single DeFi.tsx. Each section
 * loads real data via the backend and exposes the corresponding action forms.
 * All calls run on Dispatchers.IO and hop back to the main thread for UI.
 */
class DeFiFragment : Fragment() {

    private lateinit var progressBar: ProgressBar
    private lateinit var statusTextView: TextView

    // Launchpool
    private lateinit var launchpoolInfo: TextView
    private lateinit var launchpoolWalletSpinner: Spinner
    private lateinit var launchpoolAmount: EditText
    private lateinit var launchpoolPassword: EditText
    private lateinit var launchpoolStakesText: TextView

    // Token sales
    private lateinit var tokenSalesText: TextView
    private lateinit var saleId: EditText
    private lateinit var saleAmount: EditText

    // DAO
    private lateinit var daoProposalsText: TextView
    private lateinit var daoProposalId: EditText
    private lateinit var daoTitle: EditText
    private lateinit var daoDescription: EditText

    // Perpetual
    private lateinit var perpPositionsText: TextView
    private lateinit var perpPair: EditText
    private lateinit var perpSideSpinner: Spinner
    private lateinit var perpSize: EditText
    private lateinit var perpLeverage: EditText
    private lateinit var perpCloseId: EditText

    // Margin
    private lateinit var marginPositionsText: TextView
    private lateinit var marginPair: EditText
    private lateinit var marginSideSpinner: Spinner
    private lateinit var marginSize: EditText
    private lateinit var marginLeverage: EditText
    private lateinit var marginCloseId: EditText

    // Prediction
    private lateinit var predictionMarketsText: TextView
    private lateinit var predMarketId: EditText
    private lateinit var predSideSpinner: Spinner
    private lateinit var predAmount: EditText

    // Lending
    private lateinit var lendingMarketsText: TextView
    private lateinit var lendingWalletSpinner: Spinner
    private lateinit var lendingAsset: EditText
    private lateinit var lendingAmount: EditText
    private lateinit var lendingPassword: EditText

    // Copy trading
    private lateinit var copyTradersText: TextView
    private lateinit var copyTraderId: EditText
    private lateinit var copyAllocation: EditText
    private lateinit var copyStopId: EditText

    private var wallets: List<UserWalletApiService.Wallet> = emptyList()
    private val sides = arrayOf("long", "short")
    private val predSides = arrayOf("yes", "no")

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_defi, container, false)
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        bindViews(view)
        setupSpinners()
        setupListeners()
        loadAll()
    }

    private fun bindViews(view: View) {
        progressBar = view.findViewById(R.id.defiProgressBar)
        statusTextView = view.findViewById(R.id.defiStatusTextView)

        launchpoolInfo = view.findViewById(R.id.launchpoolInfoText)
        launchpoolWalletSpinner = view.findViewById(R.id.launchpoolWalletSpinner)
        launchpoolAmount = view.findViewById(R.id.launchpoolAmountInput)
        launchpoolPassword = view.findViewById(R.id.launchpoolPasswordInput)
        launchpoolStakesText = view.findViewById(R.id.launchpoolStakesText)

        tokenSalesText = view.findViewById(R.id.tokenSalesText)
        saleId = view.findViewById(R.id.saleIdInput)
        saleAmount = view.findViewById(R.id.saleAmountInput)

        daoProposalsText = view.findViewById(R.id.daoProposalsText)
        daoProposalId = view.findViewById(R.id.daoProposalIdInput)
        daoTitle = view.findViewById(R.id.daoTitleInput)
        daoDescription = view.findViewById(R.id.daoDescriptionInput)

        perpPositionsText = view.findViewById(R.id.perpPositionsText)
        perpPair = view.findViewById(R.id.perpPairInput)
        perpSideSpinner = view.findViewById(R.id.perpSideSpinner)
        perpSize = view.findViewById(R.id.perpSizeInput)
        perpLeverage = view.findViewById(R.id.perpLeverageInput)
        perpCloseId = view.findViewById(R.id.perpCloseIdInput)

        marginPositionsText = view.findViewById(R.id.marginPositionsText)
        marginPair = view.findViewById(R.id.marginPairInput)
        marginSideSpinner = view.findViewById(R.id.marginSideSpinner)
        marginSize = view.findViewById(R.id.marginSizeInput)
        marginLeverage = view.findViewById(R.id.marginLeverageInput)
        marginCloseId = view.findViewById(R.id.marginCloseIdInput)

        predictionMarketsText = view.findViewById(R.id.predictionMarketsText)
        predMarketId = view.findViewById(R.id.predMarketIdInput)
        predSideSpinner = view.findViewById(R.id.predSideSpinner)
        predAmount = view.findViewById(R.id.predAmountInput)

        lendingMarketsText = view.findViewById(R.id.lendingMarketsText)
        lendingWalletSpinner = view.findViewById(R.id.lendingWalletSpinner)
        lendingAsset = view.findViewById(R.id.lendingAssetInput)
        lendingAmount = view.findViewById(R.id.lendingAmountInput)
        lendingPassword = view.findViewById(R.id.lendingPasswordInput)

        copyTradersText = view.findViewById(R.id.copyTradersText)
        copyTraderId = view.findViewById(R.id.copyTraderIdInput)
        copyAllocation = view.findViewById(R.id.copyAllocationInput)
        copyStopId = view.findViewById(R.id.copyStopIdInput)
    }

    private fun setupSpinners() {
        perpSideSpinner.adapter =
            ArrayAdapter(requireContext(), android.R.layout.simple_spinner_dropdown_item, sides)
        marginSideSpinner.adapter =
            ArrayAdapter(requireContext(), android.R.layout.simple_spinner_dropdown_item, sides)
        predSideSpinner.adapter =
            ArrayAdapter(requireContext(), android.R.layout.simple_spinner_dropdown_item, predSides)
    }

    private fun setupListeners() {
        // Launchpool
        view?.findViewById<Button>(R.id.launchpoolStakeButton)?.setOnClickListener {
            launchpoolAction(stake = true)
        }
        view?.findViewById<Button>(R.id.launchpoolUnstakeButton)?.setOnClickListener {
            launchpoolAction(stake = false)
        }

        // Token sales
        view?.findViewById<Button>(R.id.participateSaleButton)?.setOnClickListener {
            participateTokenSale()
        }

        // DAO
        view?.findViewById<Button>(R.id.daoVoteYesButton)?.setOnClickListener {
            voteDao(true)
        }
        view?.findViewById<Button>(R.id.daoVoteNoButton)?.setOnClickListener {
            voteDao(false)
        }
        view?.findViewById<Button>(R.id.daoCreateButton)?.setOnClickListener {
            createDaoProposal()
        }

        // Perpetual
        view?.findViewById<Button>(R.id.perpCreateButton)?.setOnClickListener {
            createPerpetual()
        }
        view?.findViewById<Button>(R.id.perpCloseButton)?.setOnClickListener {
            closePosition(perpCloseId, isPerpetual = true)
        }

        // Margin
        view?.findViewById<Button>(R.id.marginCreateButton)?.setOnClickListener {
            createMargin()
        }
        view?.findViewById<Button>(R.id.marginCloseButton)?.setOnClickListener {
            closePosition(marginCloseId, isPerpetual = false)
        }

        // Prediction
        view?.findViewById<Button>(R.id.predBetButton)?.setOnClickListener {
            placePredictionBet()
        }

        // Lending
        view?.findViewById<Button>(R.id.lendingSupplyButton)?.setOnClickListener {
            lendingAction(LendingAction.SUPPLY)
        }
        view?.findViewById<Button>(R.id.lendingBorrowButton)?.setOnClickListener {
            lendingAction(LendingAction.BORROW)
        }
        view?.findViewById<Button>(R.id.lendingWithdrawButton)?.setOnClickListener {
            lendingAction(LendingAction.WITHDRAW)
        }
        view?.findViewById<Button>(R.id.lendingRepayButton)?.setOnClickListener {
            lendingAction(LendingAction.REPAY)
        }

        // Copy trading
        view?.findViewById<Button>(R.id.copyFollowButton)?.setOnClickListener {
            followTrader()
        }
        view?.findViewById<Button>(R.id.copyStopButton)?.setOnClickListener {
            stopCopyTrader()
        }
    }

    private fun loadAll() {
        loadWallets {
            listOf(
                ::loadLaunchpool,
                ::loadTokenSales,
                ::loadDaoProposals,
                ::loadPerpetualPositions,
                ::loadMarginPositions,
                ::loadPredictionMarkets,
                ::loadLendingMarkets,
                ::loadCopyTraders
            ).forEach { it() }
        }
    }

    private fun loadWallets(onLoaded: () -> Unit) {
        io {
            try {
                wallets = UserWalletApiService.getWallets()
            } catch (_: Exception) { }
            ui {
                val labels = if (wallets.isEmpty()) listOf("(no wallets)")
                    else wallets.map { "${it.label} \u00b7 ${it.address.take(10)}..." }
                val ad = ArrayAdapter(requireContext(), android.R.layout.simple_spinner_dropdown_item, labels)
                launchpoolWalletSpinner.adapter = ad
                lendingWalletSpinner.adapter = ad
                onLoaded()
            }
        }
    }

    // ---- Launchpool ----
    private fun loadLaunchpool() {
        io {
            try {
                val lp = UserWalletApiService.getLaunchpool()
                val stakes = UserWalletApiService.getLaunchpoolStakes()
                ui {
                    launchpoolInfo.text = "Pool: ${lp.optString("name", lp.optString("asset", ""))} " +
                        "| APY: ${lp.optString("apy", lp.optString("reward_rate", "?"))}"
                    launchpoolStakesText.text =
                        if (stakes.isEmpty()) "No active stakes"
                        else stakes.joinToString("\n") { "\u2022 ${it.optString("amount", "?")} " +
                            "(status: ${it.optString("status", "?")})" }
                }
            } catch (e: Exception) { status("\u2717 Launchpool: ${e.message ?: "load failed"}") }
        }
    }

    private fun launchpoolAction(stake: Boolean) {
        val wallet = wallets.getOrNull(launchpoolWalletSpinner.selectedItemPosition) ?: run {
            toast("Select a wallet"); return
        }
        val amount = launchpoolAmount.text.toString().trim()
        val password = launchpoolPassword.text.toString()
        if (amount.isEmpty() || password.isEmpty()) { toast("Enter amount and password"); return }
        io {
            try {
                val res = if (stake) UserWalletApiService.launchpoolStake(wallet.id, password, amount)
                    else UserWalletApiService.launchpoolUnstake(wallet.id, password, amount)
                ui {
                    toast(if (stake) "\u2713 Staked" else "\u2713 Unstaked")
                    status("Launchpool: ${res.toString()}")
                    loadLaunchpool()
                }
            } catch (e: Exception) { status("\u2717 ${e.message ?: "action failed"}") }
        }
    }

    // ---- Token sales ----
    private fun loadTokenSales() {
        io {
            try {
                val sales = UserWalletApiService.getTokenSales()
                ui {
                    tokenSalesText.text =
                        if (sales.isEmpty()) "No active token sales"
                        else sales.joinToString("\n") {
                            "\u2022 ${it.optString("id", "?")}: ${it.optString("name", it.optString("token", ""))} " +
                                "@ ${it.optString("price", "?")}"
                        }
                }
            } catch (e: Exception) { status("\u2717 Token sales: ${e.message ?: "load failed"}") }
        }
    }

    private fun participateTokenSale() {
        val id = saleId.text.toString().trim()
        val amount = saleAmount.text.toString().trim()
        if (id.isEmpty() || amount.isEmpty()) { toast("Enter sale ID and amount"); return }
        io {
            try {
                val res = UserWalletApiService.participateTokenSale(id, amount)
                ui { toast("\u2713 Participated"); status("Token sale: ${res.toString()}"); loadTokenSales() }
            } catch (e: Exception) { status("\u2717 ${e.message ?: "participate failed"}") }
        }
    }

    // ---- DAO ----
    private fun loadDaoProposals() {
        io {
            try {
                val proposals = UserWalletApiService.getDaoProposals()
                ui {
                    daoProposalsText.text =
                        if (proposals.isEmpty()) "No proposals"
                        else proposals.joinToString("\n") {
                            "\u2022 ${it.optString("id", "?")}: ${it.optString("title", "?")} " +
                                "[for ${it.optString("for", it.optInt("for_votes", 0))} / " +
                                "against ${it.optString("against", it.optInt("against_votes", 0))}]"
                        }
                }
            } catch (e: Exception) { status("\u2717 DAO: ${e.message ?: "load failed"}") }
        }
    }

    private fun voteDao(support: Boolean) {
        val id = daoProposalId.text.toString().trim()
        if (id.isEmpty()) { toast("Enter proposal ID"); return }
        io {
            try {
                val res = UserWalletApiService.voteDaoProposal(id, support)
                ui { toast("\u2713 Vote submitted"); status("DAO vote: ${res.toString()}"); loadDaoProposals() }
            } catch (e: Exception) { status("\u2717 ${e.message ?: "vote failed"}") }
        }
    }

    private fun createDaoProposal() {
        val title = daoTitle.text.toString().trim()
        val description = daoDescription.text.toString().trim()
        if (title.isEmpty() || description.isEmpty()) { toast("Enter title and description"); return }
        io {
            try {
                val res = UserWalletApiService.createDaoProposal(title, description)
                ui { toast("\u2713 Proposal created"); status("DAO: ${res.toString()}"); loadDaoProposals() }
            } catch (e: Exception) { status("\u2717 ${e.message ?: "create failed"}") }
        }
    }

    // ---- Perpetual ----
    private fun loadPerpetualPositions() {
        io {
            try {
                val positions = UserWalletApiService.getPerpetualPositions()
                ui {
                    perpPositionsText.text =
                        if (positions.isEmpty()) "No open perpetual positions"
                        else positions.joinToString("\n") {
                            "\u2022 ${it.optString("id", "?")}: ${it.optString("pair", "?")} " +
                                "${it.optString("side", "?")} size ${it.optString("size", "?")}"
                        }
                }
            } catch (e: Exception) { status("\u2717 Perpetual: ${e.message ?: "load failed"}") }
        }
    }

    private fun createPerpetual() {
        val pair = perpPair.text.toString().trim()
        val size = perpSize.text.toString().trim()
        val leverage = perpLeverage.text.toString().trim().toIntOrNull() ?: 0
        if (pair.isEmpty() || size.isEmpty() || leverage <= 0) { toast("Enter pair, size, leverage"); return }
        val side = sides[perpSideSpinner.selectedItemPosition]
        val chainId = currentChainId()
        io {
            try {
                val res = UserWalletApiService.createPerpetualPosition(pair, side, size, leverage, chainId)
                ui { toast("\u2713 Position opened"); status("Perpetual: ${res.toString()}"); loadPerpetualPositions() }
            } catch (e: Exception) { status("\u2717 ${e.message ?: "open failed"}") }
        }
    }

    // ---- Margin ----
    private fun loadMarginPositions() {
        io {
            try {
                val positions = UserWalletApiService.getMarginPositions()
                ui {
                    marginPositionsText.text =
                        if (positions.isEmpty()) "No open margin positions"
                        else positions.joinToString("\n") {
                            "\u2022 ${it.optString("id", "?")}: ${it.optString("pair", "?")} " +
                                "${it.optString("side", "?")} size ${it.optString("size", "?")}"
                        }
                }
            } catch (e: Exception) { status("\u2717 Margin: ${e.message ?: "load failed"}") }
        }
    }

    private fun createMargin() {
        val pair = marginPair.text.toString().trim()
        val size = marginSize.text.toString().trim()
        val leverage = marginLeverage.text.toString().trim().toIntOrNull() ?: 0
        if (pair.isEmpty() || size.isEmpty() || leverage <= 0) { toast("Enter pair, size, leverage"); return }
        val side = sides[marginSideSpinner.selectedItemPosition]
        val chainId = currentChainId()
        io {
            try {
                val res = UserWalletApiService.createMarginPosition(pair, side, size, leverage, chainId)
                ui { toast("\u2713 Margin opened"); status("Margin: ${res.toString()}"); loadMarginPositions() }
            } catch (e: Exception) { status("\u2717 ${e.message ?: "open failed"}") }
        }
    }

    private fun closePosition(idField: EditText, isPerpetual: Boolean) {
        val id = idField.text.toString().trim()
        if (id.isEmpty()) { toast("Enter position ID"); return }
        io {
            try {
                val res = if (isPerpetual) UserWalletApiService.closePerpetualPosition(id)
                    else UserWalletApiService.closeMarginPosition(id)
                ui {
                    toast("\u2713 Position closed")
                    status("${if (isPerpetual) "Perpetual" else "Margin"}: ${res.toString()}")
                    if (isPerpetual) loadPerpetualPositions() else loadMarginPositions()
                }
            } catch (e: Exception) { status("\u2717 ${e.message ?: "close failed"}") }
        }
    }

    // ---- Prediction ----
    private fun loadPredictionMarkets() {
        io {
            try {
                val markets = UserWalletApiService.getPredictionMarkets()
                ui {
                    predictionMarketsText.text =
                        if (markets.isEmpty()) "No prediction markets"
                        else markets.joinToString("\n") {
                            "\u2022 ${it.optString("id", "?")}: ${it.optString("question", it.optString("title", "?"))}"
                        }
                }
            } catch (e: Exception) { status("\u2717 Prediction: ${e.message ?: "load failed"}") }
        }
    }

    private fun placePredictionBet() {
        val marketId = predMarketId.text.toString().trim()
        val amount = predAmount.text.toString().trim()
        if (marketId.isEmpty() || amount.isEmpty()) { toast("Enter market ID and amount"); return }
        val side = predSides[predSideSpinner.selectedItemPosition]
        io {
            try {
                val res = UserWalletApiService.placePredictionBet(marketId, side, amount)
                ui { toast("\u2713 Bet placed"); status("Prediction: ${res.toString()}"); loadPredictionMarkets() }
            } catch (e: Exception) { status("\u2717 ${e.message ?: "bet failed"}") }
        }
    }

    // ---- Lending ----
    private fun loadLendingMarkets() {
        io {
            try {
                val markets = UserWalletApiService.getLendingMarkets()
                ui {
                    lendingMarketsText.text =
                        if (markets.isEmpty()) "No lending markets"
                        else markets.joinToString("\n") {
                            "\u2022 ${it.optString("asset", "?")}: supply ${it.optString("supply_rate", "?")} " +
                                "/ borrow ${it.optString("borrow_rate", "?")}"
                        }
                }
            } catch (e: Exception) { status("\u2717 Lending: ${e.message ?: "load failed"}") }
        }
    }

    private enum class LendingAction { SUPPLY, BORROW, WITHDRAW, REPAY }

    private fun lendingAction(action: LendingAction) {
        val wallet = wallets.getOrNull(lendingWalletSpinner.selectedItemPosition) ?: run {
            toast("Select a wallet"); return
        }
        val asset = lendingAsset.text.toString().trim()
        val amount = lendingAmount.text.toString().trim()
        val password = lendingPassword.text.toString()
        if (asset.isEmpty() || amount.isEmpty() || password.isEmpty()) {
            toast("Enter asset, amount, password"); return
        }
        val chainId = wallet.chainId
        io {
            try {
                val res: JSONObject = when (action) {
                    LendingAction.SUPPLY -> UserWalletApiService.lendingSupply(wallet.id, password, asset, amount, chainId)
                    LendingAction.BORROW -> UserWalletApiService.lendingBorrow(wallet.id, password, asset, amount, chainId)
                    LendingAction.WITHDRAW -> UserWalletApiService.lendingWithdraw(wallet.id, password, asset, amount, chainId)
                    LendingAction.REPAY -> UserWalletApiService.lendingRepay(wallet.id, password, asset, amount, chainId)
                }
                ui { toast("\u2713 ${action.name.lowercase()} done"); status("Lending: ${res.toString()}"); loadLendingMarkets() }
            } catch (e: Exception) { status("\u2717 ${e.message ?: "action failed"}") }
        }
    }

    // ---- Copy trading ----
    private fun loadCopyTraders() {
        io {
            try {
                val traders = UserWalletApiService.getCopyTraders()
                ui {
                    copyTradersText.text =
                        if (traders.isEmpty()) "No copy traders"
                        else traders.joinToString("\n") {
                            "\u2022 ${it.optString("id", "?")}: ${it.optString("name", it.optString("trader", "?"))} " +
                                "ROI ${it.optString("roi", it.optString("performance", "?"))}"
                        }
                }
            } catch (e: Exception) { status("\u2717 Copy: ${e.message ?: "load failed"}") }
        }
    }

    private fun followTrader() {
        val traderId = copyTraderId.text.toString().trim()
        if (traderId.isEmpty()) { toast("Enter trader ID"); return }
        val allocation = copyAllocation.text.toString().trim().ifEmpty { null }
        io {
            try {
                val res = UserWalletApiService.followTrader(traderId, allocation)
                ui { toast("\u2713 Following trader"); status("Copy: ${res.toString()}"); loadCopyTraders() }
            } catch (e: Exception) { status("\u2717 ${e.message ?: "follow failed"}") }
        }
    }

    private fun stopCopyTrader() {
        val copierId = copyStopId.text.toString().trim()
        if (copierId.isEmpty()) { toast("Enter copier ID"); return }
        io {
            try {
                val res = UserWalletApiService.stopCopyTrader(copierId)
                ui { toast("\u2713 Stopped copying"); status("Copy: ${res.toString()}"); loadCopyTraders() }
            } catch (e: Exception) { status("\u2717 ${e.message ?: "stop failed"}") }
        }
    }

    // ---- helpers ----
    private fun currentChainId(): Int =
        wallets.getOrNull(launchpoolWalletSpinner.selectedItemPosition)?.chainId ?: 1

    private fun io(block: suspend () -> Unit) {
        CoroutineScope(Dispatchers.IO).launch { block() }
    }

    private suspend fun ui(block: () -> Unit) {
        withContext(Dispatchers.Main) { block() }
    }

    private fun status(text: String) {
        statusTextView.text = text
    }

    private fun toast(text: String) {
        Toast.makeText(requireContext(), text, Toast.LENGTH_SHORT).show()
    }
}
