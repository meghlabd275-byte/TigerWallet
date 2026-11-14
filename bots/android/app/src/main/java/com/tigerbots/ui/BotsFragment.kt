package com.tigerbots.ui

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import android.widget.Button
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.Spinner
import android.widget.TextView
import androidx.fragment.app.Fragment
import com.tigerbots.R
import com.tigerbots.api.BotApiService
import com.tigerbots.util.Fmt
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject

/**
 * Bot management: list, create, and drive the lifecycle of every bot the
 * caller owns.
 *
 * The list is rebuilt from the backend on every load and after every action, so
 * what is displayed is always the server's state rather than an optimistic
 * guess. Each row exposes the actions the backend actually supports.
 */
class BotsFragment : Fragment() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private lateinit var api: BotApiService

    private lateinit var list: LinearLayout
    private lateinit var errorView: TextView
    private lateinit var nameInput: EditText
    private lateinit var typeSpinner: Spinner
    private lateinit var exchangeInput: EditText
    private lateinit var pairInput: EditText

    /** Canonical bot types accepted by the backend (mm_bot_platform/bot_api). */
    private val botTypes = listOf(
        "market_maker", "liquidity_provider", "sniper", "front_run", "mev",
        "sandwich", "flash_loan", "cross_chain", "perp_hedge", "grid",
        "dca", "momentum", "mean_reversion", "scalping", "ai_trading",
        "signal", "arbitrage", "custom",
    )

    override fun onCreateView(
        inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?,
    ): View = inflater.inflate(R.layout.fragment_bots, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        api = BotApiService.getInstance(requireContext())
        list = view.findViewById(R.id.botsList)
        errorView = view.findViewById(R.id.botsError)
        nameInput = view.findViewById(R.id.botName)
        typeSpinner = view.findViewById(R.id.botType)
        exchangeInput = view.findViewById(R.id.botExchange)
        pairInput = view.findViewById(R.id.botPair)

        typeSpinner.adapter = ArrayAdapter(
            requireContext(), android.R.layout.simple_spinner_dropdown_item, botTypes,
        )

        view.findViewById<Button>(R.id.btnRefreshBots).setOnClickListener { load() }
        view.findViewById<Button>(R.id.btnCreateBot).setOnClickListener { create() }

        load()
    }

    private fun create() {
        val name = nameInput.text.toString().trim()
        val type = typeSpinner.selectedItem?.toString() ?: return
        if (name.isBlank()) {
            showError(getString(R.string.bot_name_required))
            return
        }
        val exchange = exchangeInput.text.toString().trim().ifBlank { null }
        val pair = pairInput.text.toString().trim().ifBlank { null }

        scope.launch {
            try {
                withContext(Dispatchers.IO) { api.createBot(name, type, null, exchange, pair) }
                nameInput.text?.clear()
                exchangeInput.text?.clear()
                pairInput.text?.clear()
                load()
            } catch (t: Throwable) {
                showError(t.message ?: getString(R.string.bot_create_failed))
            }
        }
    }

    private fun load() {
        errorView.visibility = View.GONE
        scope.launch {
            try {
                val bots = withContext(Dispatchers.IO) { api.listBots() }
                render(Fmt.array(bots, "bots", "items", "data"))
            } catch (t: Throwable) {
                showError(t.message ?: getString(R.string.load_failed))
            }
        }
    }

    private fun render(bots: org.json.JSONArray) {
        list.removeAllViews()
        if (bots.length() == 0) {
            list.addView(label(getString(R.string.empty_state)))
            return
        }
        val inflater = LayoutInflater.from(requireContext())
        for (i in 0 until bots.length()) {
            val bot = bots.optJSONObject(i) ?: continue
            val row = inflater.inflate(R.layout.item_bot, list, false)

            row.findViewById<TextView>(R.id.botRowName).text =
                Fmt.str(bot, "name").ifBlank { Fmt.str(bot, "id") }
            row.findViewById<TextView>(R.id.botRowType).text =
                Fmt.str(bot, "bot_type").ifBlank { "—" }
            row.findViewById<TextView>(R.id.botRowStatus).text =
                Fmt.str(bot, "status").ifBlank { "unknown" }
            val pair = Fmt.str(bot, "pair")
            val exchange = Fmt.str(bot, "exchange")
            row.findViewById<TextView>(R.id.botRowMarket).text =
                listOfNotNull(exchange.ifBlank { null }, pair.ifBlank { null })
                    .joinToString(" · ").ifBlank { "—" }

            val id = Fmt.str(bot, "id")
            val status = Fmt.status(bot)

            // Only offer transitions the backend accepts for the current state.
            row.findViewById<Button>(R.id.botRowStart).apply {
                isEnabled = status != "running"
                setOnClickListener { action(getString(R.string.action_start)) { api.startBot(id) } }
            }
            row.findViewById<Button>(R.id.botRowPause).apply {
                isEnabled = status == "running"
                setOnClickListener { action(getString(R.string.action_pause)) { api.pauseBot(id) } }
            }
            row.findViewById<Button>(R.id.botRowStop).apply {
                isEnabled = status == "running" || status == "paused"
                setOnClickListener { action(getString(R.string.action_stop)) { api.stopBot(id) } }
            }
            row.findViewById<Button>(R.id.botRowDelete).setOnClickListener {
                action(getString(R.string.action_delete)) { api.deleteBot(id) }
            }

            list.addView(row)
        }
    }

    private fun action(verb: String, call: suspend () -> Any) {
        errorView.visibility = View.GONE
        scope.launch {
            try {
                withContext(Dispatchers.IO) { call() }
                load()
            } catch (t: Throwable) {
                showError("$verb: ${t.message ?: getString(R.string.action_failed)}")
            }
        }
    }

    private fun label(text: String) = TextView(requireContext()).apply {
        this.text = text
        setPadding(16, 24, 16, 24)
    }

    private fun showError(message: String) {
        errorView.text = message
        errorView.visibility = View.VISIBLE
    }

    override fun onDestroyView() {
        scope.cancel()
        super.onDestroyView()
    }
}
