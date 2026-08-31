package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.TextView
import android.widget.Toast
import androidx.fragment.app.Fragment
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONArray

/**
 * dApps & WalletConnect: create pairings from a WalletConnect URI, list
 * pairings (approve/reject), list active sessions, and respond to pending
 * dApp requests. All state comes from the canonical dapp_browser backend via
 * the wallet_api proxy — no fabricated pairings/sessions.
 */
class DAppsFragment : Fragment() {

    private lateinit var container: LinearLayout

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_dapps, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        container = view.findViewById(R.id.dappsList)
        buildUi()
        refresh()
    }

    private fun buildUi() {
        val ctx = requireContext()
        val uriInput = EditText(ctx).apply { hint = "WalletConnect URI (wc:…)" }
        val pairBtn = Button(ctx).apply { text = "Pair" }
        val pairingsHeader = TextView(ctx).apply { text = "Pairings"; textSize = 18f; setPadding(0, 24, 0, 8) }
        val pairingsBox = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; tag = "pairings" }
        val sessionsHeader = TextView(ctx).apply { text = "Sessions"; textSize = 18f; setPadding(0, 24, 0, 8) }
        val sessionsBox = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; tag = "sessions" }
        val catalogHeader = TextView(ctx).apply { text = "dApp Catalog"; textSize = 18f; setPadding(0, 24, 0, 8) }
        val catalogBox = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; tag = "catalog" }
        val refreshBtn = Button(ctx).apply { text = "Refresh" }

        container.addView(uriInput)
        container.addView(pairBtn)
        container.addView(pairingsHeader)
        container.addView(pairingsBox)
        container.addView(sessionsHeader)
        container.addView(sessionsBox)
        container.addView(catalogHeader)
        container.addView(catalogBox)
        container.addView(refreshBtn)

        pairBtn.setOnClickListener {
            val uri = uriInput.text.toString().trim()
            if (uri.isEmpty()) {
                Toast.makeText(ctx, "Paste a WalletConnect URI", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    UserWalletApiService.createDappPairing(uri)
                    withContext(Dispatchers.Main) {
                        Toast.makeText(ctx, "Pairing created — approve it below", Toast.LENGTH_SHORT).show()
                        refresh()
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) {
                        Toast.makeText(ctx, "Pairing failed: ${e.message}", Toast.LENGTH_LONG).show()
                    }
                }
            }
        }

        refreshBtn.setOnClickListener { refresh() }
    }

    private fun refresh() {
        loadPairings()
        loadSessions()
        loadCatalog()
    }

    private fun loadCatalog() {
        val box = container.findViewWithTag<LinearLayout>("catalog") ?: return
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val json = UserWalletApiService.getDappCatalog()
                val list = json.optJSONArray("dapps") ?: json.optJSONArray("data") ?: JSONArray()
                withContext(Dispatchers.Main) {
                    box.removeAllViews()
                    if (list.length() == 0) {
                        box.addView(TextView(requireContext()).apply { text = "No dApps in catalog" })
                    }
                    for (i in 0 until list.length()) {
                        val d = list.getJSONObject(i)
                        box.addView(TextView(requireContext()).apply {
                            text = "${d.optString("name", "?")} · ${d.optString("category", "?")} · ${d.optString("url", "")}"
                            setPadding(0, 12, 0, 12)
                        })
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    box.removeAllViews()
                    box.addView(TextView(requireContext()).apply { text = "Catalog unavailable: ${e.message}" })
                }
            }
        }
    }

    private fun loadPairings() {
        val box = container.findViewWithTag<LinearLayout>("pairings") ?: return
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val json = UserWalletApiService.getDappPairings()
                val list = json.optJSONArray("pairings") ?: json.optJSONArray("data") ?: JSONArray()
                withContext(Dispatchers.Main) {
                    box.removeAllViews()
                    if (list.length() == 0) {
                        box.addView(TextView(requireContext()).apply { text = "No pairings" })
                    }
                    for (i in 0 until list.length()) {
                        val p = list.getJSONObject(i)
                        val topic = p.optString("topic")
                        val row = LinearLayout(requireContext()).apply { orientation = LinearLayout.HORIZONTAL }
                        row.addView(TextView(requireContext()).apply {
                            text = "${p.optString("peer_name", p.optString("name", topic))} · ${p.optString("status", "pending")}"
                            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
                        })
                        row.addView(Button(requireContext()).apply {
                            text = "Approve"
                            setOnClickListener { pairingAction(topic, true) }
                        })
                        row.addView(Button(requireContext()).apply {
                            text = "Reject"
                            setOnClickListener { pairingAction(topic, false) }
                        })
                        box.addView(row)
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    box.removeAllViews()
                    box.addView(TextView(requireContext()).apply { text = "Pairings unavailable: ${e.message}" })
                }
            }
        }
    }

    private fun loadSessions() {
        val box = container.findViewWithTag<LinearLayout>("sessions") ?: return
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val json = UserWalletApiService.getDappSessions()
                val list = json.optJSONArray("sessions") ?: json.optJSONArray("data") ?: JSONArray()
                withContext(Dispatchers.Main) {
                    box.removeAllViews()
                    if (list.length() == 0) {
                        box.addView(TextView(requireContext()).apply { text = "No active sessions" })
                    }
                    for (i in 0 until list.length()) {
                        val s = list.getJSONObject(i)
                        box.addView(TextView(requireContext()).apply {
                            text = "${s.optString("peer_name", s.optString("name", s.optString("topic")))} · ${s.optString("topic")}"
                            setPadding(0, 12, 0, 12)
                        })
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    box.removeAllViews()
                    box.addView(TextView(requireContext()).apply { text = "Sessions unavailable: ${e.message}" })
                }
            }
        }
    }

    private fun pairingAction(topic: String, approve: Boolean) {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                if (approve) {
                    UserWalletApiService.approveDappPairing(topic)
                } else {
                    UserWalletApiService.rejectDappPairing(topic)
                }
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), if (approve) "Pairing approved" else "Pairing rejected", Toast.LENGTH_SHORT).show()
                    refresh()
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    Toast.makeText(requireContext(), "Action failed: ${e.message}", Toast.LENGTH_LONG).show()
                }
            }
        }
    }
}
