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
 * Launchpad: list active sales, contribute, and claim allocated tokens.
 *
 * Contribute and claim hit the backend's on-chain paths, so a contribution is
 * only recorded once the chain confirms it. A failure surfaces the backend's
 * message rather than a success state the chain never reached.
 */
class LaunchpadFragment : Fragment() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private lateinit var api: ProjectPartyApiService

    private lateinit var list: LinearLayout
    private lateinit var errorView: TextView
    private lateinit var contributeAmount: EditText

    override fun onCreateView(
        inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?,
    ): View = inflater.inflate(R.layout.fragment_launchpad, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        api = ProjectPartyApiService.getInstance(requireContext())
        list = view.findViewById(R.id.launchpadList)
        errorView = view.findViewById(R.id.launchpadError)
        contributeAmount = view.findViewById(R.id.launchpadAmount)

        view.findViewById<Button>(R.id.btnRefreshLaunchpad).setOnClickListener { load() }
        load()
    }

    private fun load() {
        errorView.visibility = View.GONE
        scope.launch {
            try {
                val payload = withContext(Dispatchers.IO) { api.listLaunchpads() }
                render(Fmt.array(payload, "launchpads", "items", "data"))
            } catch (t: Throwable) {
                showError(t.message ?: getString(R.string.load_failed))
            }
        }
    }

    private fun render(items: org.json.JSONArray) {
        list.removeAllViews()
        if (items.length() == 0) {
            list.addView(label(getString(R.string.empty_state)))
            return
        }
        val inflater = LayoutInflater.from(requireContext())
        for (i in 0 until items.length()) {
            val sale = items.optJSONObject(i) ?: continue
            val row = inflater.inflate(R.layout.item_launchpad, list, false)

            row.findViewById<TextView>(R.id.launchpadRowName).text =
                Fmt.str(sale, "name").ifBlank { Fmt.str(sale, "token_symbol").ifBlank { Fmt.str(sale, "id") } }
            row.findViewById<TextView>(R.id.launchpadRowStatus).text =
                Fmt.str(sale, "status").ifBlank { "unknown" }

            val price = Fmt.str(sale, "token_price").ifBlank { Fmt.str(sale, "price") }
            val hardCap = Fmt.str(sale, "hard_cap")
            val raised = Fmt.str(sale, "total_raised").ifBlank { Fmt.str(sale, "raised") }
            row.findViewById<TextView>(R.id.launchpadRowDetail).text = buildString {
                if (price.isNotBlank()) append("Price: ").append(price).append("\n")
                if (hardCap.isNotBlank()) append("Hard cap: ").append(hardCap).append("\n")
                if (raised.isNotBlank()) append("Raised: ").append(raised)
            }.trim().ifBlank { "—" }

            val id = Fmt.str(sale, "id")
            val status = Fmt.status(sale)

            row.findViewById<Button>(R.id.launchpadRowContribute).apply {
                isEnabled = status == "active" || status == "upcoming"
                setOnClickListener {
                    val amount = contributeAmount.text.toString().trim()
                    if (amount.isBlank()) {
                        showError(getString(R.string.launchpad_amount_required))
                        return@setOnClickListener
                    }
                    action { api.contribute(id, amount) }
                }
            }
            row.findViewById<Button>(R.id.launchpadRowClaim).apply {
                isEnabled = status == "completed" || status == "ended" || status == "claimable"
                setOnClickListener { action { api.claimTokens(id) } }
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
