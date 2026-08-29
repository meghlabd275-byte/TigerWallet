package com.tigeruserwallet.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.TextView
import androidx.fragment.app.Fragment
import com.tigeruserwallet.R
import com.tigeruserwallet.api.UserWalletApiService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject

/**
 * Security Center: check URLs/addresses against the threat registry
 * (/security/check-url|check-address) or run a full scan (/security/scan).
 * Results come from the backend's live checkers; an empty threat list means
 * "clean", not "unchecked".
 */
class SecurityFragment : Fragment() {

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? = inflater.inflate(R.layout.fragment_security, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        val targetInput = view.findViewById<EditText>(R.id.securityTargetInput)
        val resultText = view.findViewById<TextView>(R.id.securityResultText)

        view.findViewById<Button>(R.id.securityCheckButton).setOnClickListener {
            val target = targetInput.text.toString().trim()
            if (target.isEmpty()) { resultText.text = "Enter a URL or address"; return@setOnClickListener }
            resultText.text = "Checking…"
            val isUrl = target.startsWith("http://") || target.startsWith("https://")
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val res = if (isUrl) UserWalletApiService.checkUrl(target)
                        else UserWalletApiService.checkAddress(target)
                    withContext(Dispatchers.Main) {
                        resultText.text = if (res.optBoolean("safe")) "✓ Safe: ${res.optString("reason", "no threats")}"
                        else "⚠ Flagged: ${res.optString("reason", "threat detected")}"
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { resultText.text = "Check failed: ${e.message}" }
                }
            }
        }

        view.findViewById<Button>(R.id.securityScanButton).setOnClickListener {
            val target = targetInput.text.toString().trim()
            if (target.isEmpty()) { resultText.text = "Enter a URL or address"; return@setOnClickListener }
            resultText.text = "Scanning…"
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val res: JSONObject = UserWalletApiService.securityScan(target)
                    val threats = res.optJSONArray("threats")
                    withContext(Dispatchers.Main) {
                        resultText.text = if (threats != null && threats.length() > 0)
                            "⚠ Threats: $threats" else "✓ Safe: no threats detected"
                    }
                } catch (e: Exception) {
                    withContext(Dispatchers.Main) { resultText.text = "Scan failed: ${e.message}" }
                }
            }
        }
    }
}
