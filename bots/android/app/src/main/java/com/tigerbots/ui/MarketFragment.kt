package com.tigerbots.ui

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.LinearLayout
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
import org.json.JSONArray
import org.json.JSONObject

/**
 * Market: public subscription tiers plus the caller's real exchange
 * connections.
 *
 * A CEX connection needs an exchange name and a key/secret pair; a DEX
 * connection needs the venue and the signing key. The backend rejects anything
 * less, so the forms require the same fields rather than sending a partial
 * record the server would refuse.
 */
class MarketFragment : Fragment() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private lateinit var api: BotApiService

    private lateinit var tiersView: LinearLayout
    private lateinit var cexView: LinearLayout
    private lateinit var dexView: LinearLayout
    private lateinit var errorView: TextView

    private lateinit var cexExchange: EditText
    private lateinit var cexKey: EditText
    private lateinit var cexSecret: EditText

    private lateinit var dexName: EditText
    private lateinit var dexChainId: EditText
    private lateinit var dexRpc: EditText
    private lateinit var dexSeed: EditText

    override fun onCreateView(
        inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?,
    ): View = inflater.inflate(R.layout.fragment_market, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        api = BotApiService.getInstance(requireContext())
        tiersView = view.findViewById(R.id.marketTiers)
        cexView = view.findViewById(R.id.marketCex)
        dexView = view.findViewById(R.id.marketDex)
        errorView = view.findViewById(R.id.marketError)

        cexExchange = view.findViewById(R.id.cexExchange)
        cexKey = view.findViewById(R.id.cexKey)
        cexSecret = view.findViewById(R.id.cexSecret)
        dexName = view.findViewById(R.id.dexName)
        dexChainId = view.findViewById(R.id.dexChainId)
        dexRpc = view.findViewById(R.id.dexRpc)
        dexSeed = view.findViewById(R.id.dexSeed)

        view.findViewById<Button>(R.id.btnRefreshMarket).setOnClickListener { load() }
        view.findViewById<Button>(R.id.btnAddCex).setOnClickListener { addCex() }
        view.findViewById<Button>(R.id.btnAddDex).setOnClickListener { addDex() }

        load()
    }

    private fun addCex() {
        val exchange = cexExchange.text.toString().trim()
        val key = cexKey.text.toString().trim()
        val secret = cexSecret.text.toString().trim()
        if (exchange.isBlank() || key.isBlank() || secret.isBlank()) {
            showError(getString(R.string.connector_cex_fields_required))
            return
        }
        errorView.visibility = View.GONE
        scope.launch {
            try {
                withContext(Dispatchers.IO) { api.addCEX(exchange, key, secret) }
                // Clear the credential fields as soon as they are stored; the
                // server keeps them encrypted and the UI has no reason to.
                cexKey.text?.clear()
                cexSecret.text?.clear()
                load()
            } catch (t: Throwable) {
                showError(t.message ?: getString(R.string.action_failed))
            }
        }
    }

    private fun addDex() {
        val dex = dexName.text.toString().trim()
        val seed = dexSeed.text.toString().trim()
        if (dex.isBlank() || seed.isBlank()) {
            showError(getString(R.string.connector_dex_fields_required))
            return
        }
        val chainId = dexChainId.text.toString().trim().toLongOrNull() ?: 0L
        val rpc = dexRpc.text.toString().trim()
        errorView.visibility = View.GONE
        scope.launch {
            try {
                withContext(Dispatchers.IO) { api.addDEX(dex, chainId, rpc, seed) }
                dexSeed.text?.clear()
                load()
            } catch (t: Throwable) {
                showError(t.message ?: getString(R.string.action_failed))
            }
        }
    }

    private fun load() {
        errorView.visibility = View.GONE
        scope.launch {
            try {
                val tiers = withContext(Dispatchers.IO) {
                    runCatching { api.publicTiers() }.getOrDefault(JSONArray())
                }
                val cex = withContext(Dispatchers.IO) { runCatching { api.listCEX() }.getOrNull() }
                val dex = withContext(Dispatchers.IO) { runCatching { api.listDEX() }.getOrNull() }
                renderTiers(tiers)
                renderConnectors(cexView, cex, "cex", "connections", "items")
                renderConnectors(dexView, dex, "dex", "connections", "items")
            } catch (t: Throwable) {
                showError(t.message ?: getString(R.string.load_failed))
            }
        }
    }

    private fun renderTiers(tiers: JSONArray) {
        tiersView.removeAllViews()
        if (tiers.length() == 0) {
            tiersView.addView(label(getString(R.string.empty_state)))
            return
        }
        for (i in 0 until tiers.length()) {
            val tier = tiers.optJSONObject(i) ?: continue
            val name = Fmt.str(tier, "tier").ifBlank { Fmt.str(tier, "name") }
            val price = Fmt.num(tier, "price")
            val maxBots = Fmt.str(tier, "max_bots").ifBlank { Fmt.str(tier, "bot_limit") }
            val fee = Fmt.num(tier, "fee_percent")
            val text = buildString {
                append(name.ifBlank { "tier" })
                if (price > 0) append(" — ").append(Fmt.money(price))
                if (maxBots.isNotBlank()) append("\nBots: ").append(maxBots)
                if (fee > 0) append("\nFee: ").append(Fmt.pct(fee))
            }
            tiersView.addView(label(text))
        }
    }

    private fun renderConnectors(container: LinearLayout, payload: JSONObject?, vararg keys: String) {
        container.removeAllViews()
        val arr = Fmt.array(payload, *keys)
        if (arr.length() == 0) {
            container.addView(label(getString(R.string.empty_state)))
            return
        }
        for (i in 0 until arr.length()) {
            val conn = arr.optJSONObject(i) ?: continue
            val name = Fmt.str(conn, "exchange")
                .ifBlank { Fmt.str(conn, "dex") }
                .ifBlank { Fmt.str(conn, "name") }
                .ifBlank { Fmt.str(conn, "id") }
            val active = Fmt.str(conn, "is_active")
            val line = listOf(name, if (active.isNotBlank()) "active=$active" else "")
                .filter { it.isNotBlank() }.joinToString(" · ")
            container.addView(label(line))
        }
    }

    private fun label(text: String) = TextView(requireContext()).apply {
        this.text = text
        setPadding(16, 20, 16, 20)
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
