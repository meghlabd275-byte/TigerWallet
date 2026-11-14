package com.tigerparty.ui

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.TextView
import androidx.fragment.app.Fragment
import com.tigerparty.R
import com.tigerparty.api.ProjectPartyApiService
import com.tigerparty.util.Fmt
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject

/**
 * Tokens: browse the registry, submit a new token, and — for admin callers —
 * run the listing review workflow.
 *
 * Admin actions are shown only when the backend reports an admin role. That is
 * a display decision, not a security boundary: the backend independently
 * re-checks the role on every privileged route, so a tampered local role
 * changes nothing but the buttons shown.
 */
class TokensFragment : Fragment() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private lateinit var api: ProjectPartyApiService

    private lateinit var list: LinearLayout
    private lateinit var errorView: TextView
    private lateinit var nameInput: EditText
    private lateinit var symbolInput: EditText
    private lateinit var chainInput: EditText
    private lateinit var contractInput: EditText

    override fun onCreateView(
        inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?,
    ): View = inflater.inflate(R.layout.fragment_tokens, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        api = ProjectPartyApiService.getInstance(requireContext())
        list = view.findViewById(R.id.tokensList)
        errorView = view.findViewById(R.id.tokensError)
        nameInput = view.findViewById(R.id.tokenName)
        symbolInput = view.findViewById(R.id.tokenSymbol)
        chainInput = view.findViewById(R.id.tokenChain)
        contractInput = view.findViewById(R.id.tokenContract)

        view.findViewById<Button>(R.id.btnRefreshTokens).setOnClickListener { load() }
        view.findViewById<Button>(R.id.btnCreateToken).setOnClickListener { create() }
        load()
    }

    private fun create() {
        val name = nameInput.text.toString().trim()
        val symbol = symbolInput.text.toString().trim()
        val chain = chainInput.text.toString().trim()
        if (name.isBlank() || symbol.isBlank() || chain.isBlank()) {
            showError(getString(R.string.token_fields_required))
            return
        }
        val payload = JSONObject().apply {
            put("name", name)
            put("symbol", symbol)
            put("chain", chain)
            val contract = contractInput.text.toString().trim()
            if (contract.isNotBlank()) put("contract_address", contract)
        }
        errorView.visibility = View.GONE
        scope.launch {
            try {
                withContext(Dispatchers.IO) { api.createToken(payload) }
                nameInput.text?.clear()
                symbolInput.text?.clear()
                contractInput.text?.clear()
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
                val payload = withContext(Dispatchers.IO) { api.listTokens() }
                render(Fmt.array(payload, "tokens", "items", "data"))
            } catch (t: Throwable) {
                showError(t.message ?: getString(R.string.load_failed))
            }
        }
    }

    private fun render(tokens: org.json.JSONArray) {
        list.removeAllViews()
        if (tokens.length() == 0) {
            list.addView(label(getString(R.string.empty_state)))
            return
        }
        val isAdmin = api.getRole().equals("admin", ignoreCase = true) ||
            api.getRole().equals("super_admin", ignoreCase = true)

        val inflater = LayoutInflater.from(requireContext())
        for (i in 0 until tokens.length()) {
            val token = tokens.optJSONObject(i) ?: continue
            val row = inflater.inflate(R.layout.item_token, list, false)

            row.findViewById<TextView>(R.id.tokenRowName).text =
                Fmt.str(token, "name").ifBlank { Fmt.str(token, "id") }
            row.findViewById<TextView>(R.id.tokenRowSymbol).text =
                Fmt.str(token, "symbol").ifBlank { "—" }
            row.findViewById<TextView>(R.id.tokenRowStatus).text =
                Fmt.str(token, "status").ifBlank { "unknown" }
            val chain = Fmt.str(token, "chain")
            val contract = Fmt.str(token, "contract_address")
            row.findViewById<TextView>(R.id.tokenRowDetail).text =
                listOf(chain, contract).filter { it.isNotBlank() }.joinToString("\n").ifBlank { "—" }

            val id = Fmt.str(token, "id")
            val status = Fmt.status(token)

            // Owner actions: submit a draft token for review.
            row.findViewById<Button>(R.id.tokenRowSubmit).apply {
                isEnabled = status == "draft" || status == "rejected"
                setOnClickListener { action { api.submitToken(id) } }
            }

            // Admin review actions.
            row.findViewById<Button>(R.id.tokenRowApprove).apply {
                visibility = if (isAdmin) View.VISIBLE else View.GONE
                isEnabled = status == "submitted" || status == "in_review" || status == "pending"
                setOnClickListener { action { api.approveToken(id) } }
            }
            row.findViewById<Button>(R.id.tokenRowReject).apply {
                visibility = if (isAdmin) View.VISIBLE else View.GONE
                isEnabled = status == "submitted" || status == "in_review" || status == "pending"
                setOnClickListener { action { api.rejectToken(id) } }
            }
            row.findViewById<Button>(R.id.tokenRowVerify).apply {
                visibility = if (isAdmin) View.VISIBLE else View.GONE
                setOnClickListener { action { api.verifyTokenContract(id) } }
            }

            list.addView(row)
        }
    }

    private fun action(call: suspend () -> Any) {
        errorView.visibility = View.GONE
        scope.launch {
            try {
                withContext(Dispatchers.IO) { call() }
                load()
            } catch (t: Throwable) {
                showError(t.message ?: getString(R.string.action_failed))
            }
        }
    }

    private fun label(text: String) = TextView(requireContext()).apply {
        this.text = text
        setPadding(8, 16, 8, 16)
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
