package com.tigerparty.ui

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.RadioGroup
import android.widget.TextView
import androidx.fragment.app.Fragment
import com.tigerparty.MainActivity
import com.tigerparty.R
import com.tigerparty.TigerPartyApp
import com.tigerparty.api.ProjectPartyApiService
import com.tigerparty.util.Fmt
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Settings: backend endpoint, appearance, account summary, and — for admin
 * callers — the fee-payment verification queue.
 *
 * The verification queue is a real on-chain check performed by the backend;
 * this screen only triggers it.
 */
class SettingsFragment : Fragment() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private lateinit var api: ProjectPartyApiService

    private lateinit var urlInput: EditText
    private lateinit var statusView: TextView
    private lateinit var accountView: TextView
    private lateinit var errorView: TextView
    private lateinit var themeGroup: RadioGroup
    private lateinit var adminCard: View
    private lateinit var feesView: LinearLayout
    private lateinit var feeIdInput: EditText

    override fun onCreateView(
        inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?,
    ): View = inflater.inflate(R.layout.fragment_settings, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        api = ProjectPartyApiService.getInstance(requireContext())
        urlInput = view.findViewById(R.id.settingsBackendUrl)
        statusView = view.findViewById(R.id.settingsStatus)
        accountView = view.findViewById(R.id.settingsAccount)
        errorView = view.findViewById(R.id.settingsError)
        themeGroup = view.findViewById(R.id.settingsThemeGroup)
        adminCard = view.findViewById(R.id.settingsAdminCard)
        feesView = view.findViewById(R.id.settingsFees)
        feeIdInput = view.findViewById(R.id.settingsFeeId)

        urlInput.setText(api.currentBaseUrl())
        view.findViewById<Button>(R.id.btnSaveBackend).setOnClickListener { saveBackend() }
        view.findViewById<Button>(R.id.btnLogout).setOnClickListener {
            (activity as? MainActivity)?.logout()
        }
        view.findViewById<Button>(R.id.btnVerifyFee).setOnClickListener { verifyFee() }

        themeGroup.check(
            when (TigerPartyApp.currentMode(requireContext())) {
                TigerPartyApp.MODE_LIGHT -> R.id.themeLight
                TigerPartyApp.MODE_DARK -> R.id.themeDark
                else -> R.id.themeSystem
            }
        )
        themeGroup.setOnCheckedChangeListener { _, checkedId ->
            val mode = when (checkedId) {
                R.id.themeLight -> TigerPartyApp.MODE_LIGHT
                R.id.themeDark -> TigerPartyApp.MODE_DARK
                else -> TigerPartyApp.MODE_SYSTEM
            }
            TigerPartyApp.writeNightMode(requireContext(), mode)
        }

        load()
    }

    private fun saveBackend() {
        errorView.visibility = View.GONE
        try {
            api.setBaseUrl(urlInput.text.toString())
            statusView.text = getString(R.string.settings_backend_saved)
            load()
        } catch (t: Throwable) {
            errorView.text = t.message ?: getString(R.string.settings_backend_invalid)
            errorView.visibility = View.VISIBLE
        }
    }

    private fun verifyFee() {
        val id = feeIdInput.text.toString().trim()
        if (id.isBlank()) {
            errorView.text = getString(R.string.fee_id_required)
            errorView.visibility = View.VISIBLE
            return
        }
        errorView.visibility = View.GONE
        scope.launch {
            try {
                withContext(Dispatchers.IO) { api.verifyFeePayment(id) }
                feeIdInput.text?.clear()
                load()
            } catch (t: Throwable) {
                errorView.text = t.message ?: getString(R.string.action_failed)
                errorView.visibility = View.VISIBLE
            }
        }
    }

    private fun load() {
        scope.launch {
            try {
                val health = withContext(Dispatchers.IO) { runCatching { api.getHealth() }.getOrNull() }
                statusView.text = if (health == null) {
                    getString(R.string.backend_unreachable)
                } else {
                    getString(R.string.backend_status, Fmt.str(health, "status").ifBlank { "unknown" })
                }

                val role = api.getRole().ifBlank { "user" }
                accountView.text = buildString {
                    append("Signed in as: ").append(role)
                }

                val isAdmin = role.equals("admin", ignoreCase = true) ||
                    role.equals("super_admin", ignoreCase = true)
                adminCard.visibility = if (isAdmin) View.VISIBLE else View.GONE
                if (isAdmin) {
                    val fees = withContext(Dispatchers.IO) {
                        runCatching { api.getListingFees() }.getOrNull()
                    }
                    renderFees(fees)
                }
            } catch (t: Throwable) {
                errorView.text = t.message ?: getString(R.string.load_failed)
                errorView.visibility = View.VISIBLE
            }
        }
    }

    private fun renderFees(payload: org.json.JSONObject?) {
        feesView.removeAllViews()
        val arr = Fmt.array(payload, "fees", "payments", "items")
        if (arr.length() == 0) {
            feesView.addView(label(getString(R.string.empty_state)))
            return
        }
        for (i in 0 until arr.length()) {
            val fee = arr.optJSONObject(i) ?: continue
            val id = Fmt.str(fee, "id")
            val status = Fmt.str(fee, "status").ifBlank { "pending" }
            feesView.addView(label("$id · $status"))
        }
    }

    private fun label(text: String) = TextView(requireContext()).apply {
        this.text = text
        setPadding(8, 12, 8, 12)
    }

    override fun onDestroyView() {
        scope.cancel()
        super.onDestroyView()
    }
}
